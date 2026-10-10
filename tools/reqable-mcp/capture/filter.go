package capture

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// maxBodySize caps ReadBody at 5MB.
const maxBodySize = 5 << 20

// Filter narrows records; empty keyword/host/method/appPattern and zero
// code disable that clause. All active clauses must match.
func Filter(records []Record, f FilterSpec) []Record {
	var out []Record
	for _, r := range records {
		if f.Keyword != "" {
			hay := r.URL + "\x00" + r.Host + "\x00" + strings.Join(r.ReqHeaders, "\n") + "\x00" + strings.Join(r.ResHeaders, "\n")
			if !strings.Contains(strings.ToLower(hay), strings.ToLower(f.Keyword)) {
				continue
			}
		}
		if f.Host != "" && !strings.Contains(strings.ToLower(r.Host+"\x00"+r.URL), strings.ToLower(f.Host)) {
			continue
		}
		if f.Method != "" && !strings.EqualFold(r.Method, f.Method) {
			continue
		}
		if f.Code != 0 && r.StatusCode != f.Code {
			continue
		}
		if f.App != "" && !strings.Contains(strings.ToLower(r.URL), strings.ToLower(f.App)) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// FilterSpec selects capture records; zero values disable the clause.
type FilterSpec struct {
	Keyword, Host, Method, App string
	Code                       int
}

// WaitFor polls List until a record newer than the current max matches
// spec + path substring, or timeout seconds elapse. Returns found=false
// on timeout.
func WaitFor(spec FilterSpec, path string, timeout int) (Record, bool, error) {
	start, err := List()
	if err != nil {
		return Record{}, false, err
	}
	maxID := 0
	for _, r := range start {
		if r.ID > maxID {
			maxID = r.ID
		}
	}
	matches := func(r Record) bool {
		if spec.Host != "" && !strings.Contains(strings.ToLower(r.Host+"\x00"+r.URL), strings.ToLower(spec.Host)) {
			return false
		}
		if spec.Method != "" && !strings.EqualFold(r.Method, spec.Method) {
			return false
		}
		if path != "" && !strings.Contains(r.URL, path) {
			return false
		}
		return true
	}
	for i := 0; i <= timeout; i++ {
		recs, err := List()
		if err != nil {
			return Record{}, false, err
		}
		for _, r := range recs {
			if r.ID > maxID && matches(r) {
				return r, true, nil
			}
		}
		if i < timeout {
			time.Sleep(time.Second)
		}
	}
	return Record{}, false, nil
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
