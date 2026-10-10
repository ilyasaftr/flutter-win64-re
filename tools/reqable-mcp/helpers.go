package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"reqable-mcp/capture"
)

func textResult(text string) map[string]any {
	return map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}}
}

func globalCaptureConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "com.reqable.macosx", "config", "capture_config"), nil
}

func loadJSONMap(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func saveJSONAtomic(path string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func marshalPretty(v any) string {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(raw)
}

func appSupportBase() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "com.reqable.macosx"), nil
}

// listText renders "kind (n):" plus one "  id name url" line per entry.
func listText(kind string, n int, lines []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (%d):\n", kind, n)
	for _, l := range lines {
		sb.WriteString("  " + l + "\n")
	}
	return sb.String()
}

// boxStats reports box key count, capture file count, and db size,
// tolerating missing pieces (keys -1 when the dump fails).
func boxStats() (keys int, files int, size int64) {
	keys = -1
	if recs, err := capture.List(); err == nil {
		keys = len(recs)
	}
	if base, err := appSupportBase(); err == nil {
		if entries, err := os.ReadDir(filepath.Join(base, "capture")); err == nil {
			files = len(entries)
		}
		if st, err := os.Stat(filepath.Join(base, "box", "data.mdb")); err == nil {
			size = st.Size()
		}
	}
	return keys, files, size
}

func reqableRunning() bool {
	if exec.Command("pgrep", "-x", "Reqable").Run() == nil {
		return true
	}
	return exec.Command("pgrep", "-f", "Reqable").Run() == nil
}

// captureFileText best-effort matches capture body files by timestamp prefix.
func captureFileText(rec capture.Record, maxBytes int) string {
	base, err := appSupportBase()
	if err != nil {
		return ""
	}
	entries, err := os.ReadDir(filepath.Join(base, "capture"))
	if err != nil {
		return ""
	}
	prefixes := []string{}
	for _, n := range []int64{rec.TimestampMicros, rec.ReqStartMicros} {
		if n != 0 {
			prefixes = append(prefixes, strconv.FormatInt(n, 10))
		}
	}
	if len(prefixes) == 0 {
		return ""
	}
	var fallback string
	for _, e := range entries {
		name := e.Name()
		matched := false
		for _, px := range prefixes {
			if strings.HasPrefix(name, px+"-") {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(base, "capture", name))
		if err != nil {
			continue
		}
		if len(raw) > maxBytes {
			raw = raw[:maxBytes]
		}
		text := string(raw)
		if strings.Contains(name, "res-raw-body") {
			return text
		}
		if fallback == "" {
			fallback = text
		}
	}
	return fallback
}
