package main

import (
	"fmt"
	"reqable-mcp/store"
)

const scriptFramework = "def onRequest(context, request):\n    return request\ndef onResponse(context, response):\n    return response\n"
const scriptTemplate = "# Reqable script: check Reqable docs for the scripting API.\n# Hook functions below run on matching requests/responses.\ndef onRequest(context, request):\n    return request\n\ndef onResponse(context, response):\n    return response\n"

func init() {
	handlers["script_create"] = handleScriptCreate
	handlers["script_get_config"] = handleScriptGetConfig
	handlers["script_set_enabled"] = handleScriptSetEnabled
	handlers["script_get_by_id"] = handleScriptGetById
	handlers["script_update"] = handleScriptUpdate
	handlers["script_set_item_enabled"] = handleScriptSetItemEnabled
	handlers["script_list"] = handleScriptList
	handlers["script_framework"] = handleScriptFramework
	handlers["script_template"] = handleScriptTemplate
}

func handleScriptCreate(args map[string]any) (string, error) {
	name, url, code := strArg(args, "name", ""), strArg(args, "url", ""), strArg(args, "code", "")
	if name == "" || url == "" || code == "" {
		return "", badArgs("name, url, code are required")
	}
	path, err := store.WriteScriptSidecar(ud(args), name, url, code, true)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("script %s staged -> import %s in Reqable", name, path), nil
}

func handleScriptGetConfig(args map[string]any) (string, error) {
	_, err := store.ScriptGet(ud(args), "")
	if err != nil {
		return "", err
	}
	return "{}", nil
}

func handleScriptSetEnabled(args map[string]any) (string, error) {
	if !hasArg(args, "enabled") {
		return "", badArgs("enabled is required")
	}
	if err := store.SetScriptEnabled(ud(args), boolArg(args, "enabled", false)); err != nil {
		return "", err
	}
	return "scripts updated", nil
}

func handleScriptGetById(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	m, err := store.GetScriptByID(ud(args), id)
	if err != nil {
		return "", err
	}
	return marshalPretty(m), nil
}

func handleScriptUpdate(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	script := objArg(args, "script")
	if script == nil {
		script = map[string]any{}
		for _, k := range []string{"name", "url", "code", "isEnabled", "wildcard"} {
			if hasArg(args, k) {
				script[k] = args[k]
			}
		}
	}
	if err := store.UpdateScript(ud(args), id, script); err != nil {
		return "", err
	}
	return fmt.Sprintf("script %s updated", id), nil
}

func handleScriptSetItemEnabled(args map[string]any) (string, error) {
	id, enabled, err := reqIDFlag(args, "enabled")
	if err != nil {
		return "", err
	}
	if err := store.SetScriptItemEnabled(ud(args), id, enabled); err != nil {
		return "", err
	}
	return fmt.Sprintf("script %s updated", id), nil
}

func handleScriptList(args map[string]any) (string, error) {
	items, err := store.ScriptList(ud(args))
	if err != nil {
		return "", err
	}
	return marshalPretty(items), nil
}

func handleScriptFramework(args map[string]any) (string, error) {
	return scriptFramework + "\n# Check Reqable docs for the scripting API.\n", nil
}

func handleScriptTemplate(args map[string]any) (string, error) {
	return scriptTemplate, nil
}
