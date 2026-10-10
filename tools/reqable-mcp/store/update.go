package store

import (
	"encoding/json"
	"errors"
	"fmt"
)

// GetRewriteByID returns the rewrite rule with the given id.
func GetRewriteByID(userDir, id string) (RewriteEntry, error) {
	var cfg RewriteFile
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return RewriteEntry{}, err
	}
	for _, e := range cfg.Rewrites {
		if e.ID == id {
			return e, nil
		}
	}
	return RewriteEntry{}, fmt.Errorf("rewrite %s not found", id)
}

// UpdateRewrite replaces the rule matching e.ID in full.
func UpdateRewrite(userDir string, e RewriteEntry) error {
	var cfg RewriteFile
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return err
	}
	for i, cur := range cfg.Rewrites {
		if cur.ID == e.ID {
			cfg.Rewrites[i] = e
			return save(RewritePath(userDir), cfg)
		}
	}
	return fmt.Errorf("rewrite %s not found", e.ID)
}

// rewriteEntryRaw mirrors RewriteEntry with an opaque action so any
// action shape round-trips without loss.
type rewriteEntryRaw struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Method    string          `json:"method"`
	URL       string          `json:"url"`
	Wildcard  bool            `json:"wildcard"`
	Action    json.RawMessage `json:"action"`
	IsEnabled bool            `json:"isEnabled"`
}

type rewriteFileRaw struct {
	Rewrites  []rewriteEntryRaw `json:"rewrites"`
	IsEnabled bool              `json:"isEnabled"`
}

// UpdateRewriteRaw replaces a rule while accepting any action shape.
func UpdateRewriteRaw(userDir, id, name, method, url string, wildcard, isEnabled bool, action json.RawMessage) error {
	var cfg rewriteFileRaw
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return err
	}
	for i, cur := range cfg.Rewrites {
		if cur.ID == id {
			cfg.Rewrites[i] = rewriteEntryRaw{
				ID: id, Name: name, Method: method, URL: url,
				Wildcard: wildcard, Action: action, IsEnabled: isEnabled,
			}
			return save(RewritePath(userDir), cfg)
		}
	}
	return fmt.Errorf("rewrite %s not found", id)
}

// SetRewriteEnabled toggles the top-level rewrite switch.
func SetRewriteEnabled(userDir string, enabled bool) error {
	var cfg RewriteFile
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return err
	}
	cfg.IsEnabled = enabled
	return save(RewritePath(userDir), cfg)
}

// SetRewriteItemEnabled toggles a single rule.
func SetRewriteItemEnabled(userDir, id string, enabled bool) error {
	var cfg RewriteFile
	if err := load(RewritePath(userDir), &cfg); err != nil {
		return err
	}
	for i, cur := range cfg.Rewrites {
		if cur.ID == id {
			cfg.Rewrites[i].IsEnabled = enabled
			return save(RewritePath(userDir), cfg)
		}
	}
	return fmt.Errorf("rewrite %s not found", id)
}

// GetBreakpointByID returns the breakpoint with the given id.
func GetBreakpointByID(userDir, id string) (BreakpointEntry, error) {
	var cfg BreakpointFile
	if err := load(BreakpointPath(userDir), &cfg); err != nil {
		return BreakpointEntry{}, err
	}
	for _, e := range cfg.Breakpoints {
		if e.ID == id {
			return e, nil
		}
	}
	return BreakpointEntry{}, fmt.Errorf("breakpoint %s not found", id)
}

// UpdateBreakpoint replaces the breakpoint matching e.ID in full.
func UpdateBreakpoint(userDir string, e BreakpointEntry) error {
	var cfg BreakpointFile
	if err := load(BreakpointPath(userDir), &cfg); err != nil {
		return err
	}
	for i, cur := range cfg.Breakpoints {
		if cur.ID == e.ID {
			cfg.Breakpoints[i] = e
			return save(BreakpointPath(userDir), cfg)
		}
	}
	return fmt.Errorf("breakpoint %s not found", e.ID)
}

// SetBreakpointEnabled toggles the top-level breakpoint switch.
func SetBreakpointEnabled(userDir string, enabled bool) error {
	var cfg BreakpointFile
	if err := load(BreakpointPath(userDir), &cfg); err != nil {
		return err
	}
	cfg.IsEnabled = enabled
	return save(BreakpointPath(userDir), cfg)
}

// SetBreakpointItemEnabled toggles a single breakpoint.
func SetBreakpointItemEnabled(userDir, id string, enabled bool) error {
	var cfg BreakpointFile
	if err := load(BreakpointPath(userDir), &cfg); err != nil {
		return err
	}
	for i, cur := range cfg.Breakpoints {
		if cur.ID == id {
			cfg.Breakpoints[i].IsEnabled = enabled
			return save(BreakpointPath(userDir), cfg)
		}
	}
	return fmt.Errorf("breakpoint %s not found", id)
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
