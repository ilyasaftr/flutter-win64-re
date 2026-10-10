package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

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

// RewriteEntry mirrors the JSON shape the Reqable app persists in
// config/capture_rewrite.
type RewriteEntry struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Method    string        `json:"method"`
	URL       string        `json:"url"`
	Wildcard  bool          `json:"wildcard"`
	Action    RewriteAction `json:"action"`
	IsEnabled bool          `json:"isEnabled"`
}

type RewriteAction struct {
	Type                 int             `json:"type"`
	Code                 *int            `json:"code,omitempty"`
	StatusLineEnabled    *bool           `json:"statusLineEnabled,omitempty"`
	HeadersEnabled       *bool           `json:"headersEnabled,omitempty"`
	BodyEnabled          *bool           `json:"bodyEnabled,omitempty"`
	Headers              []any           `json:"headers,omitempty"`
	Body                 *RewriteBody    `json:"body,omitempty"`
	Modifies             []RewriteModify `json:"modifies,omitempty"`
	ModifyQueryTemplate  *string         `json:"modifyQueryTemplate"`
	ModifyHeaderTemplate *string         `json:"modifyHeaderTemplate"`
	ModifyBodyTemplate   *string         `json:"modifyBodyTemplate"`
}

type RewriteBody struct {
	Type    int    `json:"type"`
	Payload string `json:"payload"`
}

type RewriteModify struct {
	ID        string       `json:"id"`
	Type      int          `json:"type"`
	Modify    ModifyDetail `json:"modify"`
	IsEnabled bool         `json:"isEnabled"`
}

type ModifyDetail struct {
	Pattern       string `json:"pattern"`
	Replacement   string `json:"replacement"`
	Regex         bool   `json:"regex"`
	CaseSensitive bool   `json:"caseSensitive"`
}

type RewriteFile struct {
	Rewrites  []RewriteEntry `json:"rewrites"`
	IsEnabled bool           `json:"isEnabled"`
}

type BreakpointEntry struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Method            string `json:"method"`
	URL               string `json:"url"`
	Wildcard          bool   `json:"wildcard"`
	IsRequestEnabled  bool   `json:"isRequestEnabled"`
	IsResponseEnabled bool   `json:"isResponseEnabled"`
	IsEnabled         bool   `json:"isEnabled"`
}

type BreakpointFile struct {
	Breakpoints []BreakpointEntry `json:"breakpoints"`
	IsEnabled   bool              `json:"isEnabled"`
}

type CollectionsFile struct {
	Collections []map[string]any `json:"collections"`
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// ListRewrites returns (entries, isEnabled).
func ListRewrites(userDir string) ([]RewriteEntry, bool, error) {
	var cfg RewriteFile
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return nil, false, err
	}
	return cfg.Rewrites, cfg.IsEnabled, nil
}

// AddRewrite appends a Replace-Response (type 2) rule and returns its id.
func AddRewrite(userDir, name, method, url, payload string, wildcard bool) (string, error) {
	var cfg RewriteFile
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return "", err
	}
	id := mustUUID()
	cfg.Rewrites = append(cfg.Rewrites, RewriteEntry{
		ID: id, Name: name, Method: method, URL: url, Wildcard: wildcard,
		Action: RewriteAction{
			Type: 2, Code: intPtr(200),
			StatusLineEnabled: boolPtr(false), HeadersEnabled: boolPtr(false),
			BodyEnabled: boolPtr(true), Headers: []any{},
			Body: &RewriteBody{Type: 1, Payload: payload},
		},
		IsEnabled: true,
	})
	return id, save(RewritePath(userDir), cfg)
}

// AddRewriteModify appends a Modify (type 4) rule and returns its id.
func AddRewriteModify(userDir, name, url, pattern, replacement string) (string, error) {
	var cfg RewriteFile
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return "", err
	}
	id := mustUUID()
	cfg.Rewrites = append(cfg.Rewrites, RewriteEntry{
		ID: id, Name: name, URL: url, Wildcard: true,
		Action: RewriteAction{
			Type: 4,
			Modifies: []RewriteModify{{
				ID: mustUUID(), Type: 0,
				Modify:    ModifyDetail{Pattern: pattern, Replacement: replacement, Regex: true},
				IsEnabled: true,
			}},
		},
		IsEnabled: true,
	})
	return id, save(RewritePath(userDir), cfg)
}

// DeleteRewrite removes a rule by id; returns (before, after).
func DeleteRewrite(userDir, id string) (int, int, error) {
	var cfg RewriteFile
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return 0, 0, err
	}
	before := len(cfg.Rewrites)
	kept := cfg.Rewrites[:0]
	for _, e := range cfg.Rewrites {
		if e.ID != id {
			kept = append(kept, e)
		}
	}
	cfg.Rewrites = kept
	return before, len(kept), save(RewritePath(userDir), cfg)
}

// ListBreakpoints returns (entries, isEnabled).
func ListBreakpoints(userDir string) ([]BreakpointEntry, bool, error) {
	var cfg BreakpointFile
	if err := load(BreakpointPath(userDir), &cfg); err != nil {
		return nil, false, err
	}
	return cfg.Breakpoints, cfg.IsEnabled, nil
}

// AddBreakpoint appends a breakpoint rule and returns its id.
func AddBreakpoint(userDir, name, method, url string, wildcard bool) (string, error) {
	var cfg BreakpointFile
	if err := load(BreakpointPath(userDir), &cfg); err != nil {
		return "", err
	}
	id := mustUUID()
	cfg.Breakpoints = append(cfg.Breakpoints, BreakpointEntry{
		ID: id, Name: name, Method: method, URL: url, Wildcard: wildcard,
		IsRequestEnabled: true, IsResponseEnabled: true, IsEnabled: true,
	})
	return id, save(BreakpointPath(userDir), cfg)
}

// DeleteBreakpoint removes a breakpoint by id; returns (before, after).
func DeleteBreakpoint(userDir, id string) (int, int, error) {
	var cfg BreakpointFile
	if err := load(BreakpointPath(userDir), &cfg); err != nil {
		return 0, 0, err
	}
	before := len(cfg.Breakpoints)
	kept := cfg.Breakpoints[:0]
	for _, e := range cfg.Breakpoints {
		if e.ID != id {
			kept = append(kept, e)
		}
	}
	cfg.Breakpoints = kept
	return before, len(kept), save(BreakpointPath(userDir), cfg)
}

// AddCollection appends an empty collection and returns its id.
func AddCollection(userDir, name string) (string, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return "", err
	}
	id := mustUUID()
	cfg.Collections = append(cfg.Collections, map[string]any{
		"type": "collection", "id": id, "name": name,
		"items": []any{}, "revision": 1,
	})
	return id, save(CollectionsPath(userDir), cfg)
}

// ListCollections returns raw collection objects.
func ListCollections(userDir string) ([]map[string]any, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return nil, err
	}
	return cfg.Collections, nil
}

// WriteScriptSidecar persists a script definition as a JSON sidecar next
// to the config dir for manual import. Scripts live in the object store,
// so there is no plain config file to append to.
func WriteScriptSidecar(userDir, name, url, code string, wildcard bool) (string, error) {
	id := mustUUID()
	path := RewritePath(userDir) + "_grok_" + id + ".json"
	doc := map[string]any{
		"id": id, "name": name, "url": url, "wildcard": wildcard,
		"code": code, "isEnabled": true,
	}
	return path, save(path, doc)
}
