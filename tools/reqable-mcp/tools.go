package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"reqable-mcp/capture"
	"reqable-mcp/store"
)

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// toolDefs lists 73 tools.
func toolDefs() []toolDef {
	obj := func(props map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": props, "required": required}
	}
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	boolean := func(desc string, def bool) map[string]any {
		return map[string]any{"type": "boolean", "description": desc, "default": def}
	}
	intArg := func(desc string, def int) map[string]any {
		return map[string]any{"type": "integer", "description": desc, "default": def}
	}
	arr := func(desc string) map[string]any {
		return map[string]any{"type": "array", "description": desc}
	}
	rawObj := func(desc string) map[string]any {
		return map[string]any{"type": "object", "description": desc}
	}
	return []toolDef{
		{
			Name:        "rewrite_list",
			Description: "List Reqable rewrite rules (name, url, id).",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "rewrite_create",
			Description: "Create a Replace-Response rewrite rule (mock a URL with a JSON body). Restart Reqable after to load it.",
			InputSchema: obj(map[string]any{
				"name": str("Rule name"), "url": str("Match URL, e.g. https://api.example.com/*"),
				"payload": str("Response body"), "method": str("HTTP method, empty = any"),
				"wildcard": boolean("Wildcard match", true),
			}, "name", "url"),
		},
		{
			Name:        "rewrite_modify_create",
			Description: "Create a Modify rewrite rule (regex replace on matching traffic). Restart Reqable after.",
			InputSchema: obj(map[string]any{
				"name": str("Rule name"), "url": str("Match URL"),
				"pattern": str("Regex pattern"), "replacement": str("Replacement"),
			}, "name", "url", "pattern", "replacement"),
		},
		{
			Name:        "rewrite_delete",
			Description: "Delete a rewrite rule by id. Restart Reqable after.",
			InputSchema: obj(map[string]any{"id": str("Rewrite id")}, "id"),
		},
		{
			Name:        "rewrite_get_config",
			Description: "Return the full rewrite JSON config.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "rewrite_set_enabled",
			Description: "Toggle the top-level rewrite switch.",
			InputSchema: obj(map[string]any{"enabled": boolean("Enable rewrites", true)}, "enabled"),
		},
		{
			Name:        "rewrite_get_by_id",
			Description: "Get a rewrite rule by id.",
			InputSchema: obj(map[string]any{"id": str("Rewrite id")}, "id"),
		},
		{
			Name:        "rewrite_update",
			Description: "Replace a rewrite rule in full (action is a raw JSON object).",
			InputSchema: obj(map[string]any{
				"id": str("Rewrite id"), "name": str("Rule name"), "method": str("HTTP method"),
				"url": str("Match URL"), "wildcard": boolean("Wildcard match", true),
				"action": rawObj("Raw action JSON object"), "isEnabled": boolean("Rule enabled", true),
			}, "id"),
		},
		{
			Name:        "rewrite_set_item_enabled",
			Description: "Toggle a single rewrite rule.",
			InputSchema: obj(map[string]any{"id": str("Rewrite id"), "enabled": boolean("Enable rule", true)}, "id", "enabled"),
		},
		{
			Name:        "breakpoint_list",
			Description: "List Reqable breakpoint rules.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "breakpoint_create",
			Description: "Create a breakpoint rule (pause matching traffic). Restart Reqable after.",
			InputSchema: obj(map[string]any{
				"name": str("Rule name"), "url": str("Match URL, e.g. https://api.example.com/*"),
			}, "name", "url"),
		},
		{
			Name:        "breakpoint_delete",
			Description: "Delete a breakpoint rule by id. Restart Reqable after.",
			InputSchema: obj(map[string]any{"id": str("Breakpoint id")}, "id"),
		},
		{
			Name:        "breakpoint_get_config",
			Description: "Return the full breakpoint JSON config.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "breakpoint_set_enabled",
			Description: "Toggle the top-level breakpoint switch.",
			InputSchema: obj(map[string]any{"enabled": boolean("Enable breakpoints", true)}, "enabled"),
		},
		{
			Name:        "breakpoint_get_by_id",
			Description: "Get a breakpoint rule by id.",
			InputSchema: obj(map[string]any{"id": str("Breakpoint id")}, "id"),
		},
		{
			Name:        "breakpoint_update",
			Description: "Replace a breakpoint rule in full.",
			InputSchema: obj(map[string]any{
				"id": str("Breakpoint id"), "name": str("Rule name"), "method": str("HTTP method"),
				"url": str("Match URL"), "wildcard": boolean("Wildcard match", true),
				"isRequestEnabled": boolean("Break on request", true),
				"isResponseEnabled": boolean("Break on response", true),
				"isEnabled": boolean("Rule enabled", true),
			}, "id"),
		},
		{
			Name:        "breakpoint_set_item_enabled",
			Description: "Toggle a single breakpoint rule.",
			InputSchema: obj(map[string]any{"id": str("Breakpoint id"), "enabled": boolean("Enable rule", true)}, "id", "enabled"),
		},
		{
			Name:        "collection_list",
			Description: "List Reqable collections.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "collection_create",
			Description: "Create an empty Reqable collection. Restart Reqable after.",
			InputSchema: obj(map[string]any{"name": str("Collection name")}, "name"),
		},
		{
			Name:        "collection_get",
			Description: "Get one collection object by id.",
			InputSchema: obj(map[string]any{"id": str("Collection id")}, "id"),
		},
		{
			Name:        "collection_delete",
			Description: "Delete a collection by id.",
			InputSchema: obj(map[string]any{"id": str("Collection id")}, "id"),
		},
		{
			Name:        "collection_update",
			Description: "Update a collection (optional name).",
			InputSchema: obj(map[string]any{"id": str("Collection id"), "name": str("New name")}, "id"),
		},
		{
			Name:        "collection_structure",
			Description: "Return the full collections JSON (pretty).",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "collection_folder_create",
			Description: "Create a folder ref inside a collection.",
			InputSchema: obj(map[string]any{
				"collectionId": str("Collection id"), "name": str("Folder name"), "parentId": str("Parent folder id"),
			}, "collectionId", "name"),
		},
		{
			Name:        "collection_folder_get",
			Description: "Get a folder ref by id within a collection.",
			InputSchema: obj(map[string]any{"collectionId": str("Collection id"), "id": str("Folder id")}, "collectionId", "id"),
		},
		{
			Name:        "collection_api_create",
			Description: "Create an API in a collection from a curl command (sidecar for manual import).",
			InputSchema: obj(map[string]any{
				"collectionId": str("Collection id"), "curl": str("Curl command"), "parentId": str("Parent folder id"),
			}, "collectionId", "curl"),
		},
		{
			Name:        "collection_api_add",
			Description: "Add an API object to a collection.",
			InputSchema: obj(map[string]any{
				"collectionId": str("Collection id"), "api": rawObj("API object"), "parentId": str("Parent folder id"),
			}, "collectionId", "api"),
		},
		{
			Name:        "collection_api_get",
			Description: "Get a stored API ref by id within a collection.",
			InputSchema: obj(map[string]any{"collectionId": str("Collection id"), "id": str("API id")}, "collectionId", "id"),
		},
		{
			Name:        "collection_api_update",
			Description: "Overwrite the sidecar for a sidecar-managed API.",
			InputSchema: obj(map[string]any{"collectionId": str("Collection id"), "api": rawObj("Full API object with id")}, "collectionId", "api"),
		},
		{
			Name:        "collection_api_delete",
			Description: "Delete an API ref (and sidecar) from a collection.",
			InputSchema: obj(map[string]any{"collectionId": str("Collection id"), "id": str("API id")}, "collectionId", "id"),
		},
		{
			Name:        "collection_api_generate_curl",
			Description: "Build a curl command for a stored API.",
			InputSchema: obj(map[string]any{"collectionId": str("Collection id"), "id": str("API id")}, "collectionId", "id"),
		},
		{
			Name:        "script_create",
			Description: "Stage a Reqable script definition as a JSON sidecar for manual import (scripts live in the object store).",
			InputSchema: obj(map[string]any{
				"name": str("Script name"), "url": str("Match URL"), "code": str("Script source"),
			}, "name", "url", "code"),
		},
		{
			Name:        "script_get_config",
			Description: "Return script config (honest error: scripts live in the object store).",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "script_set_enabled",
			Description: "Toggle scripts (honest stub).",
			InputSchema: obj(map[string]any{"enabled": boolean("Enable scripts", true)}, "enabled"),
		},
		{
			Name:        "script_get_by_id",
			Description: "Get a script by id (honest stub).",
			InputSchema: obj(map[string]any{"id": str("Script id")}, "id"),
		},
		{
			Name:        "script_update",
			Description: "Update a script by id (honest stub).",
			InputSchema: obj(map[string]any{
				"id": str("Script id"), "script": rawObj("Script object"),
				"name": str("Script name"), "url": str("Match URL"), "code": str("Script source"),
			}, "id"),
		},
		{
			Name:        "script_set_item_enabled",
			Description: "Toggle a single script (honest stub).",
			InputSchema: obj(map[string]any{"id": str("Script id"), "enabled": boolean("Enable script", true)}, "id", "enabled"),
		},
		{
			Name:        "script_list",
			Description: "List scripts (honest stub).",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "script_framework",
			Description: "Return a minimal Reqable script framework snippet.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "script_template",
			Description: "Return a basic passthrough script template.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "rest_http_create_from_url",
			Description: "Stage an HTTP tab draft from a URL.",
			InputSchema: obj(map[string]any{"name": str("Tab name"), "url": str("URL")}, "name", "url"),
		},
		{
			Name:        "rest_http_create_from_curl",
			Description: "Stage an HTTP tab draft parsed from curl.",
			InputSchema: obj(map[string]any{"name": str("Tab name"), "curl": str("Curl command")}, "name", "curl"),
		},
		{
			Name:        "rest_http_update",
			Description: "Overwrite a staged HTTP tab.",
			InputSchema: obj(map[string]any{"id": str("Tab id"), "tab": rawObj("Full tab object")}, "id", "tab"),
		},
		{
			Name:        "rest_http_get",
			Description: "Load a staged HTTP tab.",
			InputSchema: obj(map[string]any{"id": str("Tab id")}, "id"),
		},
		{
			Name:        "rest_websocket_create_from_url",
			Description: "Stage a websocket tab draft from a URL.",
			InputSchema: obj(map[string]any{"name": str("Tab name"), "url": str("URL")}, "name", "url"),
		},
		{
			Name:        "rest_websocket_update",
			Description: "Overwrite a staged websocket tab.",
			InputSchema: obj(map[string]any{"id": str("Tab id"), "tab": rawObj("Full tab object")}, "id", "tab"),
		},
		{
			Name:        "rest_websocket_get",
			Description: "Load a staged websocket tab.",
			InputSchema: obj(map[string]any{"id": str("Tab id")}, "id"),
		},
		{
			Name:        "proxy_set",
			Description: "Set the system-proxy flag in global capture_config (proxySystem).",
			InputSchema: obj(map[string]any{"overrideSystem": boolean("Enable system proxy", true)}, "overrideSystem"),
		},
		{
			Name:        "capture_live_status",
			Description: "Report live capture status (app running, capture files, box size).",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "capture_live_set_enabled",
			Description: "Set autoCapture in global capture_config.",
			InputSchema: obj(map[string]any{"enabled": boolean("Enable auto capture", true)}, "enabled"),
		},
		{
			Name:        "capture_live_filter",
			Description: "Filter live capture with keyword/host/method/code shapes (up to 100 IDs).",
			InputSchema: obj(map[string]any{"filters": arr("Filter objects: keyword{pattern}, host{hosts[]}, method{methods[]}, code{codes[]}")}),
		},
		{
			Name:        "capture_live_get_by_id",
			Description: "Get a live capture record by numeric id.",
			InputSchema: obj(map[string]any{"id": intArg("Record id", 0)}, "id"),
		},
		{
			Name:        "capture_live_generate_curl",
			Description: "Generate curl for a live capture record.",
			InputSchema: obj(map[string]any{"id": intArg("Record id", 0)}, "id"),
		},
		{
			Name:        "capture_live_compose",
			Description: "Stage a REST tab from a live capture record.",
			InputSchema: obj(map[string]any{"id": intArg("Record id", 0)}, "id"),
		},
		{
			Name:        "capture_live_collection_add",
			Description: "Stage a capture record into a collection.",
			InputSchema: obj(map[string]any{
				"id": intArg("Record id", 0), "collectionId": str("Collection id"),
				"name": str("API name"), "parentId": str("Parent folder id"),
			}, "id", "collectionId"),
		},
		{
			Name:        "capture_live_clear",
			Description: "Explain that retained records must be cleared in the Reqable UI.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "capture_list_recent",
			Description: "List recent capture records with optional host/method/status filters.",
			InputSchema: obj(map[string]any{
				"host": str("Host substring"), "method": str("HTTP method"),
				"status": intArg("Status code (0 = any)", 0), "limit": intArg("Max records", 20),
			}),
		},
		{
			Name:        "capture_get_request",
			Description: "Get a capture record by id (full includes body text up to 4KB).",
			InputSchema: obj(map[string]any{"id": intArg("Record id", 0), "full": boolean("Include body", false)}, "id"),
		},
		{
			Name:        "capture_search_url",
			Description: "Search capture URLs by substring or regex.",
			InputSchema: obj(map[string]any{
				"pattern": str("Search pattern"), "regex": boolean("Treat as regex", false), "limit": intArg("Max results", 50),
			}, "pattern"),
		},
		{
			Name:        "capture_search_body",
			Description: "Search response bodies (up to 200 records).",
			InputSchema: obj(map[string]any{"pattern": str("Search pattern"), "limit": intArg("Max results", 20)}, "pattern"),
		},
		{
			Name:        "capture_stats",
			Description: "Count records by host/method/status over the last N.",
			InputSchema: obj(map[string]any{"window": intArg("Last N records", 200)}),
		},
		{
			Name:        "capture_diff",
			Description: "Diff two capture records (method/url/status/headers).",
			InputSchema: obj(map[string]any{"idA": intArg("First id", 0), "idB": intArg("Second id", 0)}, "idA", "idB"),
		},
		{
			Name:        "capture_find_dynamic_fields",
			Description: "Find header/query keys whose values vary across a host sample.",
			InputSchema: obj(map[string]any{"host": str("Host substring"), "sample": intArg("Sample size", 20)}, "host"),
		},
		{
			Name:        "capture_extract_auth",
			Description: "Collect distinct auth header values for a host.",
			InputSchema: obj(map[string]any{"host": str("Host substring")}, "host"),
		},
		{
			Name:        "capture_decode_jwt",
			Description: "Decode a JWT header and payload without verification.",
			InputSchema: obj(map[string]any{"token": str("JWT")}, "token"),
		},
		{
			Name:        "capture_to_curl",
			Description: "Render a curl command for a capture record.",
			InputSchema: obj(map[string]any{"id": intArg("Record id", 0)}, "id"),
		},
		{
			Name:        "capture_wait_for",
			Description: "Poll capture until a new matching record appears or timeout.",
			InputSchema: obj(map[string]any{
				"host": str("Host substring"), "method": str("HTTP method"), "path": str("URL path substring"),
				"timeout": intArg("Timeout seconds", 30),
			}),
		},
		{
			Name:        "capture_list_apps",
			Description: "List distinct capture hosts as an app-ish list.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "capture_get_domains",
			Description: "Return top domains by capture count.",
			InputSchema: obj(map[string]any{}),
		},
		{
			Name:        "capture_analyze_api",
			Description: "Infer method/path/status shapes for a domain.",
			InputSchema: obj(map[string]any{"domain": str("Domain/host")}, "domain"),
		},
		{
			Name:        "capture_generate_code",
			Description: "Generate sample client code (python|curl|go) for a record.",
			InputSchema: obj(map[string]any{"id": intArg("Record id", 0), "lang": str("python|curl|go")}, "id"),
		},
		{
			Name:        "capture_import_har",
			Description: "Parse a HAR file and stage each entry as a REST tab.",
			InputSchema: obj(map[string]any{"path": str("HAR file path")}, "path"),
		},
		{
			Name:        "capture_ingest_status",
			Description: "Report box key count, capture file count, and db size.",
			InputSchema: obj(map[string]any{}),
		},
	}
}

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

func countCaptureFiles() (int, error) {
	base, err := appSupportBase()
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(filepath.Join(base, "capture"))
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

func boxDBSize() (int64, error) {
	base, err := appSupportBase()
	if err != nil {
		return 0, err
	}
	st, err := os.Stat(filepath.Join(base, "box", "data.mdb"))
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

func reqableRunning() bool {
	if exec.Command("pgrep", "-x", "Reqable").Run() == nil {
		return true
	}
	return exec.Command("pgrep", "-f", "Reqable").Run() == nil
}

// recordBodyText best-effort matches capture body files by timestamp prefix.
func recordBodyText(rec capture.Record, maxBytes int) string {
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

const scriptFramework = "def onRequest(context, request):\n    return request\ndef onResponse(context, response):\n    return response\n"

const scriptTemplate = "# Reqable script: check Reqable docs for the scripting API.\n# Hook functions below run on matching requests/responses.\ndef onRequest(context, request):\n    return request\n\ndef onResponse(context, response):\n    return response\n"

func callTool(req rpcRequest) rpcResponse {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return fail(req.ID, -32602, "invalid params")
	}
	args := params.Arguments
	if args == nil {
		args = map[string]any{}
	}
	getStr := func(k, def string) string {
		if v, ok := args[k].(string); ok {
			return v
		}
		return def
	}
	getBool := func(k string, def bool) bool {
		if v, ok := args[k].(bool); ok {
			return v
		}
		return def
	}
	has := func(k string) bool {
		_, ok := args[k]
		return ok
	}
	getInt := func(k string, def int) int {
		v, ok := args[k]
		if !ok {
			return def
		}
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		case string:
			if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
				return i
			}
			return def
		default:
			return def
		}
	}
	getObj := func(k string) map[string]any {
		if m, ok := args[k].(map[string]any); ok {
			return m
		}
		return nil
	}
	userDir, err := store.UserDir()
	if err != nil {
		return fail(req.ID, -32000, err.Error())
	}
	note := "restart Reqable to load"
	switch params.Name {
	case "rewrite_list":
		entries, _, err := store.ListRewrites(userDir)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "rewrites (%d):\n", len(entries))
		for _, e := range entries {
			fmt.Fprintf(&sb, "  %s %s %s\n", e.ID, e.Name, e.URL)
		}
		return ok(req.ID, textResult(sb.String()))
	case "rewrite_create":
		name, url := getStr("name", ""), getStr("url", "")
		if name == "" || url == "" {
			return fail(req.ID, -32602, "name and url are required")
		}
		id, err := store.AddRewrite(userDir, name, getStr("method", ""), url,
			getStr("payload", `{"ok": true}`), getBool("wildcard", true))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("rewrite %s %s -> %s", id, name, note)))
	case "rewrite_modify_create":
		name, url, pat, rep := getStr("name", ""), getStr("url", ""), getStr("pattern", ""), getStr("replacement", "")
		if name == "" || url == "" || pat == "" || rep == "" {
			return fail(req.ID, -32602, "name, url, pattern, replacement are required")
		}
		id, err := store.AddRewriteModify(userDir, name, url, pat, rep)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("modify rewrite %s %s -> %s", id, name, note)))
	case "rewrite_delete":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		before, after, err := store.DeleteRewrite(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("rewrites %d -> %d", before, after)))
	case "rewrite_get_config":
		raw, err := os.ReadFile(store.RewritePath(userDir))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(string(raw)))
	case "rewrite_set_enabled":
		if !has("enabled") {
			return fail(req.ID, -32602, "enabled is required")
		}
		enabled := getBool("enabled", false)
		if err := store.SetRewriteEnabled(userDir, enabled); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("rewrites enabled=%v -> %s", enabled, note)))
	case "rewrite_get_by_id":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		e, err := store.GetRewriteByID(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(e)))
	case "rewrite_update":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		cur, err := store.GetRewriteByID(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		name, method, url := getStr("name", cur.Name), getStr("method", cur.Method), getStr("url", cur.URL)
		wildcard, isEnabled := cur.Wildcard, cur.IsEnabled
		if has("wildcard") {
			wildcard = getBool("wildcard", wildcard)
		}
		if has("isEnabled") {
			isEnabled = getBool("isEnabled", isEnabled)
		}
		var actionRaw json.RawMessage
		if m := getObj("action"); m != nil {
			b, err := json.Marshal(m)
			if err != nil {
				return fail(req.ID, -32602, "action must be a JSON object")
			}
			actionRaw = b
		} else {
			b, err := json.Marshal(cur.Action)
			if err != nil {
				return fail(req.ID, -32000, err.Error())
			}
			actionRaw = b
		}
		if err := store.UpdateRewriteRaw(userDir, id, name, method, url, wildcard, isEnabled, actionRaw); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("rewrite %s updated -> %s", id, note)))
	case "rewrite_set_item_enabled":
		id := getStr("id", "")
		if id == "" || !has("enabled") {
			return fail(req.ID, -32602, "id and enabled are required")
		}
		if err := store.SetRewriteItemEnabled(userDir, id, getBool("enabled", false)); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("rewrite %s enabled=%v -> %s", id, getBool("enabled", false), note)))
	case "breakpoint_list":
		entries, _, err := store.ListBreakpoints(userDir)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "breakpoints (%d):\n", len(entries))
		for _, e := range entries {
			fmt.Fprintf(&sb, "  %s %s %s\n", e.ID, e.Name, e.URL)
		}
		return ok(req.ID, textResult(sb.String()))
	case "breakpoint_create":
		name, url := getStr("name", ""), getStr("url", "")
		if name == "" || url == "" {
			return fail(req.ID, -32602, "name and url are required")
		}
		id, err := store.AddBreakpoint(userDir, name, "", url, true)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("breakpoint %s %s -> %s", id, name, note)))
	case "breakpoint_delete":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		before, after, err := store.DeleteBreakpoint(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("breakpoints %d -> %d", before, after)))
	case "breakpoint_get_config":
		raw, err := os.ReadFile(store.BreakpointPath(userDir))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(string(raw)))
	case "breakpoint_set_enabled":
		if !has("enabled") {
			return fail(req.ID, -32602, "enabled is required")
		}
		enabled := getBool("enabled", false)
		if err := store.SetBreakpointEnabled(userDir, enabled); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("breakpoints enabled=%v -> %s", enabled, note)))
	case "breakpoint_get_by_id":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		e, err := store.GetBreakpointByID(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(e)))
	case "breakpoint_update":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		cur, err := store.GetBreakpointByID(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		e := store.BreakpointEntry{
			ID: id, Name: getStr("name", cur.Name), Method: getStr("method", cur.Method),
			URL: getStr("url", cur.URL), Wildcard: cur.Wildcard,
			IsRequestEnabled: cur.IsRequestEnabled, IsResponseEnabled: cur.IsResponseEnabled,
			IsEnabled: cur.IsEnabled,
		}
		if has("wildcard") {
			e.Wildcard = getBool("wildcard", e.Wildcard)
		}
		if has("isRequestEnabled") {
			e.IsRequestEnabled = getBool("isRequestEnabled", e.IsRequestEnabled)
		}
		if has("isResponseEnabled") {
			e.IsResponseEnabled = getBool("isResponseEnabled", e.IsResponseEnabled)
		}
		if has("isEnabled") {
			e.IsEnabled = getBool("isEnabled", e.IsEnabled)
		}
		if err := store.UpdateBreakpoint(userDir, e); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("breakpoint %s updated -> %s", id, note)))
	case "breakpoint_set_item_enabled":
		id := getStr("id", "")
		if id == "" || !has("enabled") {
			return fail(req.ID, -32602, "id and enabled are required")
		}
		if err := store.SetBreakpointItemEnabled(userDir, id, getBool("enabled", false)); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("breakpoint %s enabled=%v -> %s", id, getBool("enabled", false), note)))
	case "collection_list":
		cols, err := store.ListCollections(userDir)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "collections (%d):\n", len(cols))
		for _, c := range cols {
			fmt.Fprintf(&sb, "  %v %v\n", c["id"], c["name"])
		}
		return ok(req.ID, textResult(sb.String()))
	case "collection_create":
		name := getStr("name", "")
		if name == "" {
			return fail(req.ID, -32602, "name is required")
		}
		id, err := store.AddCollection(userDir, name)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("collection %s %s -> %s", id, name, note)))
	case "collection_get":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		c, err := store.CollectionGet(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(c)))
	case "collection_delete":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		found, err := store.CollectionDelete(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		if !found {
			return fail(req.ID, -32000, "collection "+id+" not found")
		}
		return ok(req.ID, textResult(fmt.Sprintf("collection %s deleted", id)))
	case "collection_update":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		patch := map[string]any{}
		if has("name") {
			patch["name"] = getStr("name", "")
		}
		if err := store.CollectionUpdate(userDir, id, patch); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("collection %s updated -> %s", id, note)))
	case "collection_structure":
		cols, err := store.ListCollections(userDir)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(map[string]any{"collections": cols})))
	case "collection_folder_create":
		collectionID, name := getStr("collectionId", ""), getStr("name", "")
		if collectionID == "" || name == "" {
			return fail(req.ID, -32602, "collectionId and name are required")
		}
		id, err := store.CollectionFolderCreate(userDir, collectionID, name, getStr("parentId", ""))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("folder %s %s -> %s", id, name, note)))
	case "collection_folder_get":
		collectionID, id := getStr("collectionId", ""), getStr("id", "")
		if collectionID == "" || id == "" {
			return fail(req.ID, -32602, "collectionId and id are required")
		}
		ref, err := store.CollectionFolderGet(userDir, collectionID, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(ref)))
	case "collection_api_create":
		collectionID, curl := getStr("collectionId", ""), getStr("curl", "")
		if collectionID == "" || curl == "" {
			return fail(req.ID, -32602, "collectionId and curl are required")
		}
		id, err := store.CollectionAPICreateFromCurl(userDir, collectionID, curl, getStr("parentId", ""))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("api %s staged -> %s", id, note)))
	case "collection_api_add":
		collectionID := getStr("collectionId", "")
		api := getObj("api")
		if collectionID == "" || api == nil {
			return fail(req.ID, -32602, "collectionId and api are required")
		}
		id, err := store.CollectionAPIAdd(userDir, collectionID, api, getStr("parentId", ""))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("api %s added -> %s", id, note)))
	case "collection_api_get":
		collectionID, id := getStr("collectionId", ""), getStr("id", "")
		if collectionID == "" || id == "" {
			return fail(req.ID, -32602, "collectionId and id are required")
		}
		ref, err := store.CollectionAPIGet(userDir, collectionID, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(ref)))
	case "collection_api_update":
		collectionID := getStr("collectionId", "")
		api := getObj("api")
		if collectionID == "" || api == nil {
			return fail(req.ID, -32602, "collectionId and api are required")
		}
		if err := store.CollectionAPIUpdate(userDir, collectionID, api); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("api updated -> %s", note)))
	case "collection_api_delete":
		collectionID, id := getStr("collectionId", ""), getStr("id", "")
		if collectionID == "" || id == "" {
			return fail(req.ID, -32602, "collectionId and id are required")
		}
		found, err := store.CollectionAPIDelete(userDir, collectionID, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		if !found {
			return fail(req.ID, -32000, "api "+id+" not found")
		}
		return ok(req.ID, textResult(fmt.Sprintf("api %s deleted", id)))
	case "collection_api_generate_curl":
		collectionID, id := getStr("collectionId", ""), getStr("id", "")
		if collectionID == "" || id == "" {
			return fail(req.ID, -32602, "collectionId and id are required")
		}
		curl, err := store.CollectionAPIGenerateCurl(userDir, collectionID, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(curl))
	case "script_create":
		name, url, code := getStr("name", ""), getStr("url", ""), getStr("code", "")
		if name == "" || url == "" || code == "" {
			return fail(req.ID, -32602, "name, url, code are required")
		}
		path, err := store.WriteScriptSidecar(userDir, name, url, code, true)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("script %s staged -> import %s in Reqable", name, path)))
	case "script_get_config":
		_, err := store.ScriptGet(userDir, "")
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult("{}"))
	case "script_set_enabled":
		if !has("enabled") {
			return fail(req.ID, -32602, "enabled is required")
		}
		if err := store.SetScriptEnabled(userDir, getBool("enabled", false)); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult("scripts updated"))
	case "script_get_by_id":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		m, err := store.GetScriptByID(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(m)))
	case "script_update":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		script := getObj("script")
		if script == nil {
			script = map[string]any{}
			for _, k := range []string{"name", "url", "code", "isEnabled", "wildcard"} {
				if has(k) {
					script[k] = args[k]
				}
			}
		}
		if err := store.UpdateScript(userDir, id, script); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("script %s updated", id)))
	case "script_set_item_enabled":
		id := getStr("id", "")
		if id == "" || !has("enabled") {
			return fail(req.ID, -32602, "id and enabled are required")
		}
		if err := store.SetScriptItemEnabled(userDir, id, getBool("enabled", false)); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("script %s updated", id)))
	case "script_list":
		items, err := store.ScriptList(userDir)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(items)))
	case "script_framework":
		return ok(req.ID, textResult(scriptFramework+"\n# Check Reqable docs for the scripting API.\n"))
	case "script_template":
		return ok(req.ID, textResult(scriptTemplate))
	case "rest_http_create_from_url":
		name, url := getStr("name", ""), getStr("url", "")
		if name == "" || url == "" {
			return fail(req.ID, -32602, "name and url are required")
		}
		id, err := store.RestHTTPCreateFromURL(userDir, name, url)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("http tab %s staged", id)))
	case "rest_http_create_from_curl":
		name, curl := getStr("name", ""), getStr("curl", "")
		if name == "" || curl == "" {
			return fail(req.ID, -32602, "name and curl are required")
		}
		id, err := store.RestHTTPCreateFromCurl(userDir, name, curl)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("http tab %s staged", id)))
	case "rest_http_update":
		id := getStr("id", "")
		tab := getObj("tab")
		if id == "" || tab == nil {
			return fail(req.ID, -32602, "id and tab are required")
		}
		if err := store.RestHTTPUpdate(userDir, id, tab); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("http tab %s updated", id)))
	case "rest_http_get":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		tab, err := store.RestHTTPGet(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(tab)))
	case "rest_websocket_create_from_url":
		name, url := getStr("name", ""), getStr("url", "")
		if name == "" || url == "" {
			return fail(req.ID, -32602, "name and url are required")
		}
		id, err := store.RestWSCreateFromURL(userDir, name, url)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("websocket tab %s staged", id)))
	case "rest_websocket_update":
		id := getStr("id", "")
		tab := getObj("tab")
		if id == "" || tab == nil {
			return fail(req.ID, -32602, "id and tab are required")
		}
		if err := store.RestWSUpdate(userDir, id, tab); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("websocket tab %s updated", id)))
	case "rest_websocket_get":
		id := getStr("id", "")
		if id == "" {
			return fail(req.ID, -32602, "id is required")
		}
		tab, err := store.RestWSGet(userDir, id)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(tab)))
	case "proxy_set":
		key := "overrideSystem"
		if !has(key) && has("enabled") {
			key = "enabled"
		}
		if !has(key) {
			return fail(req.ID, -32602, "overrideSystem is required")
		}
		enabled := getBool(key, false)
		path, err := globalCaptureConfigPath()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		cfg, err := loadJSONMap(path)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		cfg["proxySystem"] = enabled
		if err := saveJSONAtomic(path, cfg); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("proxySystem=%v -> restart or toggle in UI", enabled)))
	case "capture_live_status":
		files, _ := countCaptureFiles()
		size, _ := boxDBSize()
		boxKeys := -1
		if recs, err := capture.List(); err == nil {
			boxKeys = len(recs)
		}
		status := map[string]any{
			"appRunning": reqableRunning(), "boxKeys": boxKeys,
			"captureFiles": files, "boxBytes": size,
			"note": "capture dir file count + box size; restart Reqable after config writes",
		}
		return ok(req.ID, textResult(marshalPretty(status)))
	case "capture_live_set_enabled":
		if !has("enabled") {
			return fail(req.ID, -32602, "enabled is required")
		}
		enabled := getBool("enabled", false)
		path, err := globalCaptureConfigPath()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		cfg, err := loadJSONMap(path)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		cfg["autoCapture"] = enabled
		if err := saveJSONAtomic(path, cfg); err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("autoCapture=%v -> restart Reqable to load", enabled)))
	case "capture_live_filter":
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		var keyword, host, method string
		var code int
		if arr, ok := args["filters"].([]any); ok {
			for _, f := range arr {
				m, ok := f.(map[string]any)
				if !ok {
					continue
				}
				if p, ok := m["pattern"].(string); ok && keyword == "" {
					keyword = p
				}
				if kw, ok := m["keyword"].(string); ok && keyword == "" {
					keyword = kw
				}
				if h, ok := m["host"].(string); ok && host == "" {
					host = h
				}
				if hs, ok := m["hosts"].([]any); ok && host == "" {
					for _, h := range hs {
						if s, ok := h.(string); ok && s != "" {
							host = s
							break
						}
					}
				}
				if mt, ok := m["method"].(string); ok && method == "" {
					method = mt
				}
				if ms, ok := m["methods"].([]any); ok && method == "" {
					for _, v := range ms {
						if s, ok := v.(string); ok && s != "" {
							method = s
							break
						}
					}
				}
				if code == 0 {
					switch c := m["code"].(type) {
					case float64:
						code = int(c)
					case int:
						code = c
					case string:
						if i, err := strconv.Atoi(c); err == nil {
							code = i
						}
					}
					if cs, ok := m["codes"].([]any); ok {
						for _, v := range cs {
							switch c := v.(type) {
							case float64:
								code = int(c)
							case int:
								code = c
							}
							if code != 0 {
								break
							}
						}
					}
				}
			}
		}
		matched := capture.Filter(recs, keyword, host, method, code, "")
		ids := []int{}
		for _, r := range matched {
			if len(ids) >= 100 {
				break
			}
			ids = append(ids, r.ID)
		}
		return ok(req.ID, textResult(marshalPretty(map[string]any{"ids": ids, "count": len(matched)})))
	case "capture_live_get_by_id":
		if !has("id") {
			return fail(req.ID, -32602, "id is required")
		}
		r, err := capture.GetByID(getInt("id", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(r)))
	case "capture_live_generate_curl":
		if !has("id") {
			return fail(req.ID, -32602, "id is required")
		}
		r, err := capture.GetByID(getInt("id", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(capture.ToCurl(r, nil)))
	case "capture_live_compose":
		if !has("id") {
			return fail(req.ID, -32602, "id is required")
		}
		r, err := capture.GetByID(getInt("id", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		name := r.Method + " " + r.URL
		if r.Method == "" {
			name = r.URL
		}
		id, err := store.RestHTTPCreateFromURL(userDir, name, r.URL)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("staged %s", id)))
	case "capture_live_collection_add":
		if !has("id") || getStr("collectionId", "") == "" {
			return fail(req.ID, -32602, "id and collectionId are required")
		}
		r, err := capture.GetByID(getInt("id", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		name := getStr("name", "")
		if name == "" {
			name = r.Method + " " + r.URL
		}
		method := r.Method
		if method == "" {
			method = "GET"
		}
		api := map[string]any{"name": name, "method": method, "url": map[string]any{"base": r.URL}}
		id, err := store.CollectionAPIAdd(userDir, getStr("collectionId", ""), api, getStr("parentId", ""))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(fmt.Sprintf("api %s staged -> %s", id, note)))
	case "capture_live_clear":
		return ok(req.ID, textResult("retained records live in Box; clear in Reqable UI"))
	case "capture_list_recent":
		limit := getInt("limit", 20)
		if limit <= 0 {
			limit = 20
		}
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		filtered := capture.Filter(recs, "", getStr("host", ""), getStr("method", ""), getInt("status", 0), "")
		if len(filtered) > limit {
			filtered = filtered[len(filtered)-limit:]
		}
		return ok(req.ID, textResult(marshalPretty(filtered)))
	case "capture_get_request":
		if !has("id") {
			return fail(req.ID, -32602, "id is required")
		}
		r, err := capture.GetByID(getInt("id", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		if !getBool("full", false) {
			return ok(req.ID, textResult(marshalPretty(r)))
		}
		body := recordBodyText(r, 4096)
		return ok(req.ID, textResult(marshalPretty(map[string]any{"record": r, "body": body})))
	case "capture_search_url":
		pattern := getStr("pattern", "")
		if pattern == "" {
			return fail(req.ID, -32602, "pattern is required")
		}
		limit := getInt("limit", 50)
		if limit <= 0 {
			limit = 50
		}
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		var out []capture.Record
		if getBool("regex", false) {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fail(req.ID, -32602, "invalid regex: "+err.Error())
			}
			for _, r := range recs {
				if re.MatchString(r.URL) {
					out = append(out, r)
					if len(out) >= limit {
						break
					}
				}
			}
		} else {
			lower := strings.ToLower(pattern)
			for _, r := range recs {
				if strings.Contains(strings.ToLower(r.URL), lower) {
					out = append(out, r)
					if len(out) >= limit {
						break
					}
				}
			}
		}
		return ok(req.ID, textResult(marshalPretty(out)))
	case "capture_search_body":
		pattern := getStr("pattern", "")
		if pattern == "" {
			return fail(req.ID, -32602, "pattern is required")
		}
		limit := getInt("limit", 20)
		if limit <= 0 {
			limit = 20
		}
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		if len(recs) > 200 {
			recs = recs[len(recs)-200:]
		}
		var ids []int
		for _, r := range recs {
			if strings.Contains(recordBodyText(r, 65536), pattern) {
				ids = append(ids, r.ID)
				if len(ids) >= limit {
					break
				}
			}
		}
		return ok(req.ID, textResult(marshalPretty(map[string]any{"ids": ids})))
	case "capture_stats":
		window := getInt("window", 200)
		if window <= 0 {
			window = 200
		}
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		if len(recs) > window {
			recs = recs[len(recs)-window:]
		}
		return ok(req.ID, textResult(marshalPretty(capture.Stats(recs))))
	case "capture_diff":
		if !has("idA") || !has("idB") {
			return fail(req.ID, -32602, "idA and idB are required")
		}
		a, err := capture.GetByID(getInt("idA", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		b, err := capture.GetByID(getInt("idB", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(capture.Diff(a, b)))
	case "capture_find_dynamic_fields":
		host := getStr("host", "")
		if host == "" {
			return fail(req.ID, -32602, "host is required")
		}
		sample := getInt("sample", 20)
		if sample <= 0 {
			sample = 20
		}
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		filtered := capture.Filter(recs, "", host, "", 0, "")
		if len(filtered) > sample {
			filtered = filtered[len(filtered)-sample:]
		}
		return ok(req.ID, textResult(marshalPretty(capture.FindDynamicFields(filtered))))
	case "capture_extract_auth":
		host := getStr("host", "")
		if host == "" {
			return fail(req.ID, -32602, "host is required")
		}
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(capture.ExtractAuth(capture.Filter(recs, "", host, "", 0, "")))))
	case "capture_decode_jwt":
		token := getStr("token", "")
		if token == "" {
			return fail(req.ID, -32602, "token is required")
		}
		header, payload, err := capture.DecodeJWT(token)
		if err != nil {
			return fail(req.ID, -32602, err.Error())
		}
		return ok(req.ID, textResult(marshalPretty(map[string]any{"header": header, "payload": payload})))
	case "capture_to_curl":
		if !has("id") {
			return fail(req.ID, -32602, "id is required")
		}
		r, err := capture.GetByID(getInt("id", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		return ok(req.ID, textResult(capture.ToCurl(r, nil)))
	case "capture_wait_for":
		timeout := getInt("timeout", 30)
		if timeout <= 0 {
			timeout = 30
		}
		host, method, path := getStr("host", ""), getStr("method", ""), getStr("path", "")
		start, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		maxID := 0
		for _, r := range start {
			if r.ID > maxID {
				maxID = r.ID
			}
		}
		matches := func(r capture.Record) bool {
			if host != "" && !strings.Contains(strings.ToLower(r.Host+"\x00"+r.URL), strings.ToLower(host)) {
				return false
			}
			if method != "" && !strings.EqualFold(r.Method, method) {
				return false
			}
			if path != "" && !strings.Contains(r.URL, path) {
				return false
			}
			return true
		}
		for i := 0; i <= timeout; i++ {
			recs, err := capture.List()
			if err != nil {
				return fail(req.ID, -32000, err.Error())
			}
			for _, r := range recs {
				if r.ID > maxID && matches(r) {
					return ok(req.ID, textResult(marshalPretty(r)))
				}
			}
			if i < timeout {
				time.Sleep(time.Second)
			}
		}
		return ok(req.ID, textResult("timeout: no new matching record"))
	case "capture_list_apps":
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		seen := map[string]bool{}
		var apps []string
		for _, r := range recs {
			h := r.Host
			if h == "" {
				continue
			}
			if !seen[h] {
				seen[h] = true
				apps = append(apps, h)
			}
		}
		sort.Strings(apps)
		return ok(req.ID, textResult(marshalPretty(apps)))
	case "capture_get_domains":
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		counts := map[string]int{}
		for _, r := range recs {
			h := r.Host
			if h == "" {
				h = "(unknown)"
			}
			counts[h]++
		}
		type kv struct {
			Domain string `json:"domain"`
			Count  int    `json:"count"`
		}
		var out []kv
		for d, c := range counts {
			out = append(out, kv{d, c})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
		return ok(req.ID, textResult(marshalPretty(out)))
	case "capture_analyze_api":
		domain := getStr("domain", "")
		if domain == "" {
			return fail(req.ID, -32602, "domain is required")
		}
		recs, err := capture.List()
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		shapes := map[string]int{}
		for _, r := range capture.Filter(recs, "", domain, "", 0, "") {
			path := r.URL
			if u, err := url.Parse(r.URL); err == nil && u.Path != "" {
				path = u.Path
			}
			key := r.Method + " " + path + " -> " + strconv.Itoa(r.StatusCode)
			shapes[key]++
		}
		return ok(req.ID, textResult(marshalPretty(shapes)))
	case "capture_generate_code":
		if !has("id") {
			return fail(req.ID, -32602, "id is required")
		}
		lang := strings.ToLower(getStr("lang", "curl"))
		r, err := capture.GetByID(getInt("id", 0))
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		method := r.Method
		if method == "" {
			method = "GET"
		}
		switch lang {
		case "python":
			var sb strings.Builder
			sb.WriteString("import requests\nresp = requests.request(\n")
			fmt.Fprintf(&sb, "    %q, %q,\n", method, r.URL)
			sb.WriteString("    headers={\n")
			for _, h := range r.ReqHeaders {
				k, v, _ := strings.Cut(h, ":")
				fmt.Fprintf(&sb, "        %q: %q,\n", strings.TrimSpace(k), strings.TrimSpace(v))
			}
			sb.WriteString("    },\n)\nprint(resp.status_code)\nprint(resp.text)\n")
			return ok(req.ID, textResult(sb.String()))
		case "go":
			var sb strings.Builder
			sb.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc main() {\n")
			fmt.Fprintf(&sb, "\treq, _ := http.NewRequest(%q, %q, nil)\n", method, r.URL)
			for _, h := range r.ReqHeaders {
				k, v, _ := strings.Cut(h, ":")
				fmt.Fprintf(&sb, "\treq.Header.Set(%q, %q)\n", strings.TrimSpace(k), strings.TrimSpace(v))
			}
			sb.WriteString("\tresp, err := http.DefaultClient.Do(req)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tdefer resp.Body.Close()\n\tfmt.Println(resp.Status)\n}\n")
			return ok(req.ID, textResult(sb.String()))
		default:
			return ok(req.ID, textResult(capture.ToCurl(r, nil)))
		}
	case "capture_import_har":
		path := getStr("path", "")
		if path == "" {
			return fail(req.ID, -32602, "path is required")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fail(req.ID, -32000, err.Error())
		}
		var har struct {
			Log struct {
				Entries []struct {
					Request struct {
						Method string `json:"method"`
						URL    string `json:"url"`
					} `json:"request"`
				} `json:"entries"`
			} `json:"log"`
			Entries []struct {
				Request struct {
					Method string `json:"method"`
					URL    string `json:"url"`
				} `json:"request"`
			} `json:"entries"`
		}
		if err := json.Unmarshal(raw, &har); err != nil {
			return fail(req.ID, -32602, "invalid HAR: "+err.Error())
		}
		entries := har.Log.Entries
		if len(entries) == 0 {
			for _, e := range har.Entries {
				entries = append(entries, e)
			}
		}
		count := 0
		for _, e := range entries {
			if e.Request.URL == "" {
				continue
			}
			name := e.Request.Method + " " + e.Request.URL
			if _, err := store.RestHTTPCreateFromURL(userDir, name, e.Request.URL); err == nil {
				count++
			}
		}
		return ok(req.ID, textResult(fmt.Sprintf("staged %d HAR entries", count)))
	case "capture_ingest_status":
		files, _ := countCaptureFiles()
		size, _ := boxDBSize()
		keys := -1
		if recs, err := capture.List(); err == nil {
			keys = len(recs)
		}
		return ok(req.ID, textResult(marshalPretty(map[string]any{
			"boxKeys": keys, "captureFiles": files, "boxBytes": size,
		})))
	default:
		return fail(req.ID, -32601, "unknown tool: "+params.Name)
	}
}
