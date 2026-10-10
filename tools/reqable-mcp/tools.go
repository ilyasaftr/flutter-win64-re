package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"reqable-mcp/store"
)

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func toolDefs() []toolDef {
	obj := func(props map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": props, "required": required}
	}
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	boolean := func(desc string, def bool) map[string]any {
		return map[string]any{"type": "boolean", "description": desc, "default": def}
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
			Name:        "script_create",
			Description: "Stage a Reqable script definition as a JSON sidecar for manual import (scripts live in the object store).",
			InputSchema: obj(map[string]any{
				"name": str("Script name"), "url": str("Match URL"), "code": str("Script source"),
			}, "name", "url", "code"),
		},
	}
}

func textResult(text string) map[string]any {
	return map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}}
}

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
	default:
		return fail(req.ID, -32601, "unknown tool: "+params.Name)
	}
}
