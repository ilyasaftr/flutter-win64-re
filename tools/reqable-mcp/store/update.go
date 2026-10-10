package store

import (
	"encoding/json"
	"errors"
	"fmt"
)

// GetRewriteByID returns the rewrite rule with the given id.
func GetRewriteByID(userDir, id string) (RewriteEntry, error) {
	f, err := loadRewrites(userDir)
	if err != nil {
		return RewriteEntry{}, err
	}
	return GetByID(f.Items, "rewrite", id, rewriteID)
}

// UpdateRewrite replaces the rule matching e.ID in full.
func UpdateRewrite(userDir string, e RewriteEntry) error {
	f, err := loadRewrites(userDir)
	if err != nil {
		return err
	}
	i := indexByID(f.Items, e.ID, rewriteID)
	if i < 0 {
		return fmt.Errorf("rewrite %s not found", e.ID)
	}
	f.Items[i] = e
	return saveRewrites(userDir, f)
}

// rewriteEntryRaw mirrors RewriteEntry with an opaque action so any
// action shape round-trips without loss. Field order matches the app's
// layout so re-saves keep key order stable.
type rewriteEntryRaw struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Method    string          `json:"method"`
	URL       string          `json:"url"`
	Wildcard  bool            `json:"wildcard"`
	Action    json.RawMessage `json:"action"`
	IsEnabled bool            `json:"isEnabled"`
}

func rawRewriteID(e rewriteEntryRaw) string { return e.ID }

// UpdateRewriteRaw replaces a rule while accepting any action shape.
func UpdateRewriteRaw(userDir, id, name, method, url string, wildcard, isEnabled bool, action json.RawMessage) error {
	f, err := loadFolder[rewriteEntryRaw](RewritePath(userDir), "rewrites")
	if err != nil {
		return err
	}
	i := indexByID(f.Items, id, rawRewriteID)
	if i < 0 {
		return fmt.Errorf("rewrite %s not found", id)
	}
	f.Items[i] = rewriteEntryRaw{
		ID: id, Name: name, Method: method, URL: url,
		Wildcard: wildcard, Action: action, IsEnabled: isEnabled,
	}
	return saveFolder(RewritePath(userDir), "rewrites", f)
}

// SetRewriteEnabled toggles the top-level rewrite switch.
func SetRewriteEnabled(userDir string, enabled bool) error {
	return SetEnabled(RewritePath(userDir), "rewrites", enabled)
}

// SetRewriteItemEnabled toggles a single rule.
func SetRewriteItemEnabled(userDir, id string, enabled bool) error {
	f, err := loadRewrites(userDir)
	if err != nil {
		return err
	}
	if err := SetItemEnabled(f.Items, "rewrite", id, enabled, rewriteID, setRewriteOn); err != nil {
		return err
	}
	return saveRewrites(userDir, f)
}

// GetBreakpointByID returns the breakpoint with the given id.
func GetBreakpointByID(userDir, id string) (BreakpointEntry, error) {
	f, err := loadBreakpoints(userDir)
	if err != nil {
		return BreakpointEntry{}, err
	}
	return GetByID(f.Items, "breakpoint", id, breakpointID)
}

// UpdateBreakpoint replaces the breakpoint matching e.ID in full.
func UpdateBreakpoint(userDir string, e BreakpointEntry) error {
	f, err := loadBreakpoints(userDir)
	if err != nil {
		return err
	}
	i := indexByID(f.Items, e.ID, breakpointID)
	if i < 0 {
		return fmt.Errorf("breakpoint %s not found", e.ID)
	}
	f.Items[i] = e
	return saveBreakpoints(userDir, f)
}

// SetBreakpointEnabled toggles the top-level breakpoint switch.
func SetBreakpointEnabled(userDir string, enabled bool) error {
	return SetEnabled(BreakpointPath(userDir), "breakpoints", enabled)
}

// SetBreakpointItemEnabled toggles a single breakpoint.
func SetBreakpointItemEnabled(userDir, id string, enabled bool) error {
	f, err := loadBreakpoints(userDir)
	if err != nil {
		return err
	}
	if err := SetItemEnabled(f.Items, "breakpoint", id, enabled, breakpointID, setBreakpointOn); err != nil {
		return err
	}
	return saveBreakpoints(userDir, f)
}

// No capture_script config file exists; scripts persist in Box LMDB.
const errScriptStore = "scripts live in the Reqable object store; use script_create sidecar + manual import"

// GetScriptByID is an honest stub: script bodies are not in plain files.
func GetScriptByID(userDir, id string) (map[string]any, error) {
	return nil, errors.New(errScriptStore)
}

// UpdateScript is an honest stub: script bodies are not in plain files.
func UpdateScript(userDir, id string, script map[string]any) error {
	return errors.New(errScriptStore)
}

// SetScriptEnabled is an honest stub: script bodies are not in plain files.
func SetScriptEnabled(userDir string, enabled bool) error {
	return errors.New(errScriptStore)
}

// SetScriptItemEnabled is an honest stub: script bodies are not in plain files.
func SetScriptItemEnabled(userDir, id string, enabled bool) error {
	return errors.New(errScriptStore)
}

// ScriptList is an honest stub: script bodies are not in plain files.
func ScriptList(userDir string) ([]map[string]any, error) {
	return nil, errors.New(errScriptStore)
}

// ScriptGet is an honest stub: script bodies are not in plain files.
func ScriptGet(userDir, id string) (map[string]any, error) {
	return nil, errors.New(errScriptStore)
}
