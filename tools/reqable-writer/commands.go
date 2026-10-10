package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// newUUID returns a random v4 UUID string without external dependencies.
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		fatal(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	hexed := hex.EncodeToString(b[:])
	return hexed[:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:]
}

// rewriteEntry mirrors the JSON shape the Reqable app persists in
// config/capture_rewrite. Field order matches the Python helper so diffs
// stay comparable.
type rewriteEntry struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Method    string        `json:"method"`
	URL       string        `json:"url"`
	Wildcard  bool          `json:"wildcard"`
	Action    rewriteAction `json:"action"`
	IsEnabled bool          `json:"isEnabled"`
}

type rewriteAction struct {
	Type                int             `json:"type"`
	Code                *int            `json:"code,omitempty"`
	StatusLineEnabled   *bool           `json:"statusLineEnabled,omitempty"`
	HeadersEnabled      *bool           `json:"headersEnabled,omitempty"`
	BodyEnabled         *bool           `json:"bodyEnabled,omitempty"`
	Headers             []any           `json:"headers,omitempty"`
	Body                *rewriteBody    `json:"body,omitempty"`
	Modifies            []rewriteModify `json:"modifies,omitempty"`
	ModifyQueryTemplate *string         `json:"modifyQueryTemplate"`
	ModifyHeaderTemplate *string        `json:"modifyHeaderTemplate"`
	ModifyBodyTemplate  *string         `json:"modifyBodyTemplate"`
}

type rewriteBody struct {
	Type    int    `json:"type"`
	Payload string `json:"payload"`
}

type rewriteModify struct {
	ID        string       `json:"id"`
	Type      int          `json:"type"`
	Modify    modifyDetail `json:"modify"`
	IsEnabled bool         `json:"isEnabled"`
}

type modifyDetail struct {
	Pattern       string `json:"pattern"`
	Replacement   string `json:"replacement"`
	Regex         bool   `json:"regex"`
	CaseSensitive bool   `json:"caseSensitive"`
}

type rewriteFile struct {
	Rewrites  []rewriteEntry `json:"rewrites"`
	IsEnabled bool           `json:"isEnabled"`
}

type breakpointEntry struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Method           string `json:"method"`
	URL              string `json:"url"`
	Wildcard         bool   `json:"wildcard"`
	IsRequestEnabled bool   `json:"isRequestEnabled"`
	IsResponseEnabled bool  `json:"isResponseEnabled"`
	IsEnabled        bool   `json:"isEnabled"`
}

type breakpointFile struct {
	Breakpoints []breakpointEntry `json:"breakpoints"`
	IsEnabled   bool              `json:"isEnabled"`
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func rewritePath(userDir string) string {
	return filepath.Join(userDir, "config", "capture_rewrite")
}

func breakpointPath(userDir string) string {
	return filepath.Join(userDir, "config", "capture_breakpoint")
}

func cmdList(_ *flag.FlagSet) {
	userDir := resolveUserDir()
	var rw rewriteFile
	loadJSON(rewritePath(userDir), &rw)
	fmt.Printf("capture_rewrite (%d):\n", len(rw.Rewrites))
	for _, e := range rw.Rewrites {
		fmt.Printf("  %s %s %s\n", e.ID, e.Name, e.URL)
	}
	var bp breakpointFile
	loadJSON(breakpointPath(userDir), &bp)
	fmt.Printf("capture_breakpoint (%d):\n", len(bp.Breakpoints))
	for _, e := range bp.Breakpoints {
		fmt.Printf("  %s %s %s\n", e.ID, e.Name, e.URL)
	}
}

func cmdRewrite(fs *flag.FlagSet) {
	name := fs.String("name", "", "rule name")
	url := fs.String("url", "", "match URL")
	payload := fs.String("payload", `{"ok": true}`, "response body")
	method := fs.String("method", "", "HTTP method (empty = any)")
	noWildcard := fs.Bool("nowildcard", false, "disable wildcard match")
	_ = fs.Parse(os.Args[2:])
	require(*name, "name")
	require(*url, "url")
	userDir := resolveUserDir()
	var cfg rewriteFile
	loadJSON(rewritePath(userDir), &cfg)
	id := newUUID()
	cfg.Rewrites = append(cfg.Rewrites, rewriteEntry{
		ID:       id,
		Name:     *name,
		Method:   *method,
		URL:      *url,
		Wildcard: !*noWildcard,
		Action: rewriteAction{
			Type:              2,
			Code:              intPtr(200),
			StatusLineEnabled: boolPtr(false),
			HeadersEnabled:    boolPtr(false),
			BodyEnabled:       boolPtr(true),
			Headers:           []any{},
			Body:              &rewriteBody{Type: 1, Payload: *payload},
		},
		IsEnabled: true,
	})
	saveJSON(rewritePath(userDir), cfg)
	fmt.Printf("rewrite %s %s -> restart Reqable to load\n", id, *name)
}

func cmdRewriteModify(fs *flag.FlagSet) {
	name := fs.String("name", "", "rule name")
	url := fs.String("url", "", "match URL")
	pattern := fs.String("pattern", "", "regex pattern")
	replacement := fs.String("replacement", "", "replacement")
	_ = fs.Parse(os.Args[2:])
	require(*name, "name")
	require(*url, "url")
	require(*pattern, "pattern")
	require(*replacement, "replacement")
	userDir := resolveUserDir()
	var cfg rewriteFile
	loadJSON(rewritePath(userDir), &cfg)
	id := newUUID()
	cfg.Rewrites = append(cfg.Rewrites, rewriteEntry{
		ID:       id,
		Name:     *name,
		URL:      *url,
		Wildcard: true,
		Action: rewriteAction{
			Type: 4,
			Modifies: []rewriteModify{{
				ID:   newUUID(),
				Type: 0,
				Modify: modifyDetail{
					Pattern:       *pattern,
					Replacement:   *replacement,
					Regex:         true,
					CaseSensitive: false,
				},
				IsEnabled: true,
			}},
			ModifyQueryTemplate:  nil,
			ModifyHeaderTemplate: nil,
			ModifyBodyTemplate:   nil,
		},
		IsEnabled: true,
	})
	saveJSON(rewritePath(userDir), cfg)
	fmt.Printf("modify rewrite %s %s\n", id, *name)
}

func cmdBreakpoint(fs *flag.FlagSet) {
	name := fs.String("name", "", "rule name")
	url := fs.String("url", "", "match URL")
	_ = fs.Parse(os.Args[2:])
	require(*name, "name")
	require(*url, "url")
	userDir := resolveUserDir()
	var cfg breakpointFile
	loadJSON(breakpointPath(userDir), &cfg)
	id := newUUID()
	cfg.Breakpoints = append(cfg.Breakpoints, breakpointEntry{
		ID:               id,
		Name:             *name,
		URL:              *url,
		Wildcard:         true,
		IsRequestEnabled: true,
		IsResponseEnabled: true,
		IsEnabled:        true,
	})
	saveJSON(breakpointPath(userDir), cfg)
	fmt.Printf("breakpoint %s %s\n", id, *name)
}

func cmdCollection(fs *flag.FlagSet) {
	name := fs.String("name", "", "collection name")
	_ = fs.Parse(os.Args[2:])
	require(*name, "name")
	userDir := resolveUserDir()
	path := filepath.Join(userDir, "rest", "collections")
	var cfg struct {
		Collections []map[string]any `json:"collections"`
	}
	loadJSON(path, &cfg)
	id := newUUID()
	cfg.Collections = append(cfg.Collections, map[string]any{
		"type":     "collection",
		"id":       id,
		"name":     *name,
		"items":    []any{},
		"revision": 1,
	})
	saveJSON(path, cfg)
	fmt.Printf("collection %s %s\n", id, *name)
}

func cmdScript(fs *flag.FlagSet) {
	name := fs.String("name", "", "script name")
	url := fs.String("url", "", "match URL")
	codeFile := fs.String("code", "", "file with script body")
	_ = fs.Parse(os.Args[2:])
	require(*name, "name")
	require(*url, "url")
	require(*codeFile, "code")
	code, err := os.ReadFile(*codeFile)
	if err != nil {
		fatal(err)
	}
	userDir := resolveUserDir()
	// Scripts live in the object store; persist the definition next to the
	// config dir as a JSON sidecar the app picks up on restart. The sidecar
	// keeps id, name, url, and code together for manual import.
	path := filepath.Join(userDir, "config", "capture_script_grok_"+newUUID()+".json")
	saveJSON(path, map[string]any{
		"id":        newUUID(),
		"name":      *name,
		"url":       *url,
		"wildcard":  true,
		"code":      string(code),
		"isEnabled": true,
	})
	fmt.Printf("script %s %s -> import %s in Reqable script list\n", *name, *url, path)
}

func cmdDeleteRewrite(fs *flag.FlagSet) {
	id := fs.String("id", "", "rewrite id")
	_ = fs.Parse(os.Args[2:])
	require(*id, "id")
	userDir := resolveUserDir()
	var cfg rewriteFile
	loadJSON(rewritePath(userDir), &cfg)
	before := len(cfg.Rewrites)
	kept := cfg.Rewrites[:0]
	for _, e := range cfg.Rewrites {
		if e.ID != *id {
			kept = append(kept, e)
		}
	}
	cfg.Rewrites = kept
	saveJSON(rewritePath(userDir), cfg)
	fmt.Printf("rewrites %d -> %d\n", before, len(cfg.Rewrites))
}

func cmdDeleteBreakpoint(fs *flag.FlagSet) {
	id := fs.String("id", "", "breakpoint id")
	_ = fs.Parse(os.Args[2:])
	require(*id, "id")
	userDir := resolveUserDir()
	var cfg breakpointFile
	loadJSON(breakpointPath(userDir), &cfg)
	before := len(cfg.Breakpoints)
	kept := cfg.Breakpoints[:0]
	for _, e := range cfg.Breakpoints {
		if e.ID != *id {
			kept = append(kept, e)
		}
	}
	cfg.Breakpoints = kept
	saveJSON(breakpointPath(userDir), cfg)
	fmt.Printf("breakpoints %d -> %d\n", before, len(cfg.Breakpoints))
}

func require(v, name string) {
	if v == "" {
		fatalf("missing --%s", name)
	}
}
