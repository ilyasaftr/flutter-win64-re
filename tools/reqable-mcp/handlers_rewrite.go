package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reqable-mcp/store"
)

func init() {
	handlers["rewrite_list"] = handleRewriteList
	handlers["rewrite_create"] = handleRewriteCreate
	handlers["rewrite_modify_create"] = handleRewriteModifyCreate
	handlers["rewrite_delete"] = handleRewriteDelete
	handlers["rewrite_get_config"] = handleRewriteGetConfig
	handlers["rewrite_set_enabled"] = handleRewriteSetEnabled
	handlers["rewrite_get_by_id"] = handleRewriteGetById
	handlers["rewrite_update"] = handleRewriteUpdate
	handlers["rewrite_set_item_enabled"] = handleRewriteSetItemEnabled
}

func handleRewriteList(args map[string]any) (string, error) {
	entries, _, err := store.ListRewrites(ud(args))
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, e.ID+" "+e.Name+" "+e.URL)
	}
	return listText("rewrites", len(entries), lines), nil
}

func handleRewriteCreate(args map[string]any) (string, error) {
	name, url, err := reqNameURL(args)
	if err != nil {
		return "", err
	}
	id, err := store.AddRewrite(ud(args), name, strArg(args, "method", ""), url,
		strArg(args, "payload", `{"ok": true}`), boolArg(args, "wildcard", true))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("rewrite %s %s -> %s", id, name, restartNote), nil
}

func handleRewriteModifyCreate(args map[string]any) (string, error) {
	name, url, pat, rep := strArg(args, "name", ""), strArg(args, "url", ""), strArg(args, "pattern", ""), strArg(args, "replacement", "")
	if name == "" || url == "" || pat == "" || rep == "" {
		return "", badArgs("name, url, pattern, replacement are required")
	}
	id, err := store.AddRewriteModify(ud(args), name, url, pat, rep)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("modify rewrite %s %s -> %s", id, name, restartNote), nil
}

func handleRewriteDelete(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	before, after, err := store.DeleteRewrite(ud(args), id)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("rewrites %d -> %d", before, after), nil
}

func handleRewriteGetConfig(args map[string]any) (string, error) {
	raw, err := os.ReadFile(store.RewritePath(ud(args)))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func handleRewriteSetEnabled(args map[string]any) (string, error) {
	if !hasArg(args, "enabled") {
		return "", badArgs("enabled is required")
	}
	enabled := boolArg(args, "enabled", false)
	if err := store.SetRewriteEnabled(ud(args), enabled); err != nil {
		return "", err
	}
	return fmt.Sprintf("rewrites enabled=%v -> %s", enabled, restartNote), nil
}

func handleRewriteGetById(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	e, err := store.GetRewriteByID(ud(args), id)
	if err != nil {
		return "", err
	}
	return marshalPretty(e), nil
}

func handleRewriteUpdate(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	cur, err := store.GetRewriteByID(ud(args), id)
	if err != nil {
		return "", err
	}
	name, method, url := strArg(args, "name", cur.Name), strArg(args, "method", cur.Method), strArg(args, "url", cur.URL)
	wildcard, isEnabled := cur.Wildcard, cur.IsEnabled
	if hasArg(args, "wildcard") {
		wildcard = boolArg(args, "wildcard", wildcard)
	}
	if hasArg(args, "isEnabled") {
		isEnabled = boolArg(args, "isEnabled", isEnabled)
	}
	var actionRaw json.RawMessage
	if m := objArg(args, "action"); m != nil {
		b, err := json.Marshal(m)
		if err != nil {
			return "", badArgs("action must be a JSON object")
		}
		actionRaw = b
	} else {
		b, err := json.Marshal(cur.Action)
		if err != nil {
			return "", err
		}
		actionRaw = b
	}
	if err := store.UpdateRewriteRaw(ud(args), id, name, method, url, wildcard, isEnabled, actionRaw); err != nil {
		return "", err
	}
	return fmt.Sprintf("rewrite %s updated -> %s", id, restartNote), nil
}

func handleRewriteSetItemEnabled(args map[string]any) (string, error) {
	id, enabled, err := reqIDFlag(args, "enabled")
	if err != nil {
		return "", err
	}
	if err := store.SetRewriteItemEnabled(ud(args), id, enabled); err != nil {
		return "", err
	}
	return fmt.Sprintf("rewrite %s enabled=%v -> %s", id, enabled, restartNote), nil
}
