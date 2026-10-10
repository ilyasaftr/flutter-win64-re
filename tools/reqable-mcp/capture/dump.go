package capture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
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
