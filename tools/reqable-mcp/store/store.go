// Package store reads and writes the Reqable app JSON config files
// directly, bypassing the MCP premium gate in the stock mcp-server.
// The app caches config in Box LMDB at startup, so a restart is needed
// after any write before the new entry is live.
package store

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const appSupport = "Library/Application Support/com.reqable.macosx"

// UserDir finds the active Reqable user directory: the one holding the
// most rewrites (the real account), falling back to the first dir that
// has a capture_rewrite file.
func UserDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(home, appSupport, "user")
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", err
	}
	best := ""
	bestCount := -1
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(base, e.Name(), "config", "capture_rewrite")
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg struct {
			Rewrites []json.RawMessage `json:"rewrites"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			continue
		}
		if len(cfg.Rewrites) > bestCount {
			bestCount = len(cfg.Rewrites)
			best = filepath.Join(base, e.Name())
		}
	}
	if best == "" {
		return "", fmt.Errorf("no Reqable user dir with config/capture_rewrite under %s", base)
	}
	return best, nil
}

func loadFile(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// saveFile writes atomically (tmp + rename) so a crash cannot leave a
// half written config behind.
func saveFile(path string, v any) error {
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

func RewritePath(userDir string) string {
	return filepath.Join(userDir, "config", "capture_rewrite")
}

func BreakpointPath(userDir string) string {
	return filepath.Join(userDir, "config", "capture_breakpoint")
}

func CollectionsPath(userDir string) string {
	return filepath.Join(userDir, "rest", "collections")
}

// NewUUID returns a random v4 UUID string, stdlib only.
func NewUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:], nil
}

func mustUUID() string {
	id, err := NewUUID()
	if err != nil {
		fmt.Fprintln(os.Stderr, "reqable-mcp:", err)
		os.Exit(1)
	}
	return id
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }
