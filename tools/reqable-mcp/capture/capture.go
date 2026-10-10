package capture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Record is one Reqable capture entry, flattened from the Box LMDB JSON.
type Record struct {
	ID              int      `json:"ID"`
	UID             string   `json:"UID"`
	Method          string   `json:"Method"`
	URL             string   `json:"URL"`
	Host            string   `json:"Host"`
	Port            int      `json:"Port"`
	HTTPS           bool     `json:"HTTPS"`
	StatusCode      int      `json:"StatusCode"`
	StatusMsg       string   `json:"StatusMsg"`
	Protocol        string   `json:"Protocol"`
	ReqHeaderSize   int64    `json:"ReqHeaderSize"`
	ReqBodySize     int64    `json:"ReqBodySize"`
	ResHeaderSize   int64    `json:"ResHeaderSize"`
	ResBodySize     int64    `json:"ResBodySize"`
	ReqStartMicros  int64    `json:"ReqStartMicros"`
	ResEndMicros    int64    `json:"ResEndMicros"`
	TimestampMicros int64    `json:"TimestampMicros"`
	Error           string   `json:"Error"`
	ReqHeaders      []string `json:"ReqHeaders"`
	ResHeaders      []string `json:"ResHeaders"`
}

// maxBodySize caps ReadBody at 5MB.
const maxBodySize = 5 << 20

// dumpScript opens the Box LMDB readonly and prints matching records as a
// JSON array. Capture values are ObjectBox binary wrapping a base64 gzip
// JSON blob, so the script extracts the long base64 run, gunzips it, and
// maps the session/connection/request/response shape.
const dumpScript = `
import json, os, sys, re, base64, gzip
try:
    import lmdb
except ImportError:
    sys.stderr.write('lmdb module missing: pip install lmdb\n')
    sys.exit(3)
path = os.path.expanduser('~/Library/Application Support/com.reqable.macosx/box')
try:
    env = lmdb.open(path, readonly=True, lock=False)
except Exception as e:
    sys.stderr.write('open box failed: %s\n' % e)
    sys.exit(4)
prefix = bytes([0x18, 0x00, 0x00, 0x2c])
def hdrs(h):
    r = []
    for x in (h or []):
        if isinstance(x, dict):
            r.append('%s: %s' % (x.get('key', ''), x.get('value', '')))
        else:
            r.append(str(x))
    return r
def num(x):
    try:
        return int(x)
    except Exception:
        return 0
def decode(v):
    for m in re.findall(rb'[A-Za-z0-9+/=]{200,}', v):
        try:
            raw = base64.b64decode(m)
        except Exception:
            continue
        if raw[:2] != b'\x1f\x8b':
            continue
        try:
            return json.loads(gzip.decompress(raw).decode('utf-8'))
        except Exception:
            continue
    return None
out = []
with env.begin() as txn:
    with txn.cursor() as cur:
        for k, v in cur:
            if not k.startswith(prefix):
                continue
            obj = decode(v)
            if obj is None:
                continue
            try:
                sess = obj.get('session') or {}
                conn = sess.get('connection') or {}
                req = sess.get('request') or {}
                rl = req.get('requestLine') or {}
                res = sess.get('response') or {}
                sl = res.get('statusLine') or {}
                sec = conn.get('security')
                if isinstance(sec, bool):
                    https = sec
                else:
                    https = str(sec).lower() in ('https', 'tls', 'ssl', 'true', '1')
                host = conn.get('originHost') or ''
                p = rl.get('path') or ''
                if p.startswith('http://') or p.startswith('https://'):
                    u = p
                elif host:
                    scheme = 'https' if https else 'http'
                    port = num(conn.get('originPort'))
                    default = (scheme == 'https' and port == 443) or (scheme == 'http' and port == 80)
                    if port and not default:
                        u = '%s://%s:%s%s' % (scheme, host, port, p)
                    else:
                        u = '%s://%s%s' % (scheme, host, p)
                else:
                    u = p
                out.append({
                    'ID': obj.get('id', 0),
                    'UID': obj.get('uid', ''),
                    'Method': rl.get('method', ''),
                    'URL': u,
                    'Host': host,
                    'Port': num(conn.get('originPort')),
                    'HTTPS': https,
                    'StatusCode': num(sl.get('code')),
                    'StatusMsg': sl.get('message', ''),
                    'Protocol': sl.get('protocol', '') or rl.get('protocol', ''),
                    'ReqHeaderSize': num(req.get('headerSize')),
                    'ReqBodySize': num(req.get('bodySize')),
                    'ResHeaderSize': num(res.get('headerSize')),
                    'ResBodySize': num(res.get('bodySize')),
                    'ReqStartMicros': num(req.get('startTimestamp')),
                    'ResEndMicros': num(res.get('endTimestamp')),
                    'TimestampMicros': num(sess.get('timestamp')),
                    'Error': str(obj.get('error', '') or sess.get('error', '') or ''),
                    'ReqHeaders': hdrs(req.get('headers')),
                    'ResHeaders': hdrs(res.get('headers')),
                })
            except Exception:
                continue
            if len(out) >= 2000:
                break
print(json.dumps(out))
`

func dumpViaPython() ([]byte, error) {
	cmd := exec.Command("python3", "-c", dumpScript)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.Error); ok {
			return nil, fmt.Errorf("python3 not found in PATH (required for LMDB dump): %w", err)
		}
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "No module named lmdb") || strings.Contains(msg, "lmdb module missing") {
			return nil, fmt.Errorf("python lmdb module missing (pip install lmdb): %s", msg)
		}
		if msg != "" {
			return nil, fmt.Errorf("capture dump failed: %s", msg)
		}
		return nil, fmt.Errorf("capture dump failed: %w", err)
	}
	return stdout.Bytes(), nil
}

// List dumps all capture records via the python lmdb bridge.
func List() ([]Record, error) {
	raw, err := dumpViaPython()
	if err != nil {
		return nil, err
	}
	var recs []Record
	if err := json.Unmarshal(raw, &recs); err != nil {
		return nil, fmt.Errorf("decode capture dump: %w", err)
	}
	return recs, nil
}

// GetByID returns the record with the given ID.
func GetByID(id int) (Record, error) {
	recs, err := List()
	if err != nil {
		return Record{}, err
	}
	for _, r := range recs {
		if r.ID == id {
			return r, nil
		}
	}
	return Record{}, fmt.Errorf("record %d not found", id)
}

// Filter narrows records; empty keyword/host/method/appPattern and zero
// code disable that clause. All active clauses must match.
func Filter(records []Record, keyword, host, method string, code int, appPattern string) []Record {
	var out []Record
	for _, r := range records {
		if keyword != "" {
			hay := r.URL + "\x00" + r.Host + "\x00" + strings.Join(r.ReqHeaders, "\n") + "\x00" + strings.Join(r.ResHeaders, "\n")
			if !strings.Contains(strings.ToLower(hay), strings.ToLower(keyword)) {
				continue
			}
		}
		if host != "" && !strings.Contains(strings.ToLower(r.Host+"\x00"+r.URL), strings.ToLower(host)) {
			continue
		}
		if method != "" && !strings.EqualFold(r.Method, method) {
			continue
		}
		if code != 0 && r.StatusCode != code {
			continue
		}
		if appPattern != "" && !strings.Contains(strings.ToLower(r.URL), strings.ToLower(appPattern)) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func captureDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library/Application Support/com.reqable.macosx/capture"), nil
}

// BodyPath builds the capture/ body filename for a record.
func BodyPath(timestampMicros, sessionID, recordIndex int, direction, kind string) string {
	name := fmt.Sprintf("%d-%d-%d-%s-%s.reqable", timestampMicros, sessionID, recordIndex, direction, kind)
	dir, err := captureDir()
	if err != nil {
		return name
	}
	return filepath.Join(dir, name)
}

// ReadBody reads a capture body file, refusing files over the 5MB cap.
func ReadBody(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBodySize {
		return nil, fmt.Errorf("%s exceeds 5MB body cap", path)
	}
	return raw, nil
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ToCurl renders a curl command replaying the request; body bytes stream
// via stdin (--data-binary @-). Output never embeds body content.
func ToCurl(r Record, body []byte) string {
	method := r.Method
	if method == "" {
		method = "GET"
	}
	var sb strings.Builder
	sb.WriteString("curl --noproxy '*' -X " + method + " " + shQuote(r.URL))
	for _, h := range r.ReqHeaders {
		sb.WriteString(" -H " + shQuote(h))
	}
	if len(body) > 0 {
		sb.WriteString(" --data-binary @-")
	}
	return sb.String()
}

// Stats counts records by host, method, and status code.
func Stats(records []Record) map[string]map[string]int {
	out := map[string]map[string]int{"host": {}, "method": {}, "status": {}}
	for _, r := range records {
		host := r.Host
		if host == "" {
			host = "(unknown)"
		}
		out["host"][host]++
		m := r.Method
		if m == "" {
			m = "(unknown)"
		}
		out["method"][m]++
		out["status"][strconv.Itoa(r.StatusCode)]++
	}
	return out
}

func diffHeaders(a, b []string) []string {
	inB := map[string]bool{}
	for _, h := range b {
		inB[h] = true
	}
	inA := map[string]bool{}
	for _, h := range a {
		inA[h] = true
	}
	var lines []string
	for _, h := range a {
		if !inB[h] {
			lines = append(lines, "- "+h)
		}
	}
	for _, h := range b {
		if !inA[h] {
			lines = append(lines, "+ "+h)
		}
	}
	return lines
}

// Diff reports field-level differences of method/url/status/headers.
func Diff(a, b Record) string {
	var lines []string
	if a.Method != b.Method {
		lines = append(lines, fmt.Sprintf("method: %s -> %s", a.Method, b.Method))
	}
	if a.URL != b.URL {
		lines = append(lines, fmt.Sprintf("url: %s -> %s", a.URL, b.URL))
	}
	if a.StatusCode != b.StatusCode || a.StatusMsg != b.StatusMsg {
		lines = append(lines, fmt.Sprintf("status: %d %s -> %d %s", a.StatusCode, a.StatusMsg, b.StatusCode, b.StatusMsg))
	}
	lines = append(lines, diffHeaders(a.ReqHeaders, b.ReqHeaders)...)
	lines = append(lines, diffHeaders(a.ResHeaders, b.ResHeaders)...)
	if len(lines) == 0 {
		return "no differences"
	}
	return strings.Join(lines, "\n")
}

// FindDynamicFields returns header/query keys whose values vary across
// records, sampling up to 30 evenly.
func FindDynamicFields(records []Record) []string {
	sample := records
	if len(sample) > 30 {
		stride := len(sample) / 30
		picked := make([]Record, 0, 30)
		for i := 0; i < len(sample) && len(picked) < 30; i += stride {
			picked = append(picked, sample[i])
		}
		sample = picked
	}
	vals := map[string]map[string]bool{}
	display := map[string]string{}
	add := func(id, name, v string) {
		if vals[id] == nil {
			vals[id] = map[string]bool{}
		}
		vals[id][v] = true
		if _, ok := display[id]; !ok {
			display[id] = name
		}
	}
	for _, r := range sample {
		for _, h := range append(append([]string{}, r.ReqHeaders...), r.ResHeaders...) {
			name, value, _ := strings.Cut(h, ":")
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			add("header:"+strings.ToLower(name), "header:"+name, strings.TrimSpace(value))
		}
		if u, err := url.Parse(r.URL); err == nil {
			for k, vs := range u.Query() {
				add("query:"+k, "query:"+k, strings.Join(vs, ","))
			}
		}
	}
	var out []string
	for id, vs := range vals {
		if len(vs) > 1 {
			out = append(out, display[id])
		}
	}
	sort.Strings(out)
	return out
}

// ExtractAuth collects distinct Authorization/Cookie/X-*-Token values.
func ExtractAuth(records []Record) []string {
	var out []string
	seen := map[string]bool{}
	scan := func(headers []string) {
		for _, h := range headers {
			name, value, _ := strings.Cut(h, ":")
			n := strings.ToLower(strings.TrimSpace(name))
			v := strings.TrimSpace(value)
			if v == "" {
				continue
			}
			auth := n == "authorization" || n == "cookie" ||
				(strings.HasPrefix(n, "x-") && strings.HasSuffix(n, "token"))
			if auth && !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	for _, r := range records {
		scan(r.ReqHeaders)
		scan(r.ResHeaders)
	}
	return out
}

// DecodeJWT decodes a JWT header and payload without verification.
func DecodeJWT(token string) (map[string]any, map[string]any, error) {
	return DecodeJWTUnverified(token)
}
