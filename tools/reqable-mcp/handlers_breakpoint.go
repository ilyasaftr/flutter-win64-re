package main

import (
	"fmt"
	"os"
	"reqable-mcp/store"
)

func init() {
	handlers["breakpoint_list"] = handleBreakpointList
	handlers["breakpoint_create"] = handleBreakpointCreate
	handlers["breakpoint_delete"] = handleBreakpointDelete
	handlers["breakpoint_get_config"] = handleBreakpointGetConfig
	handlers["breakpoint_set_enabled"] = handleBreakpointSetEnabled
	handlers["breakpoint_get_by_id"] = handleBreakpointGetById
	handlers["breakpoint_update"] = handleBreakpointUpdate
	handlers["breakpoint_set_item_enabled"] = handleBreakpointSetItemEnabled
}

func handleBreakpointList(args map[string]any) (string, error) {
	entries, _, err := store.ListBreakpoints(ud(args))
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, e.ID+" "+e.Name+" "+e.URL)
	}
	return listText("breakpoints", len(entries), lines), nil
}

func handleBreakpointCreate(args map[string]any) (string, error) {
	name, url, err := reqNameURL(args)
	if err != nil {
		return "", err
	}
	id, err := store.AddBreakpoint(ud(args), name, "", url, true)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("breakpoint %s %s -> %s", id, name, restartNote), nil
}

func handleBreakpointDelete(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	before, after, err := store.DeleteBreakpoint(ud(args), id)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("breakpoints %d -> %d", before, after), nil
}

func handleBreakpointGetConfig(args map[string]any) (string, error) {
	raw, err := os.ReadFile(store.BreakpointPath(ud(args)))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func handleBreakpointSetEnabled(args map[string]any) (string, error) {
	if !hasArg(args, "enabled") {
		return "", badArgs("enabled is required")
	}
	enabled := boolArg(args, "enabled", false)
	if err := store.SetBreakpointEnabled(ud(args), enabled); err != nil {
		return "", err
	}
	return fmt.Sprintf("breakpoints enabled=%v -> %s", enabled, restartNote), nil
}

func handleBreakpointGetById(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	e, err := store.GetBreakpointByID(ud(args), id)
	if err != nil {
		return "", err
	}
	return marshalPretty(e), nil
}

func handleBreakpointUpdate(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	cur, err := store.GetBreakpointByID(ud(args), id)
	if err != nil {
		return "", err
	}
	e := store.BreakpointEntry{
		ID: id, Name: strArg(args, "name", cur.Name), Method: strArg(args, "method", cur.Method),
		URL: strArg(args, "url", cur.URL), Wildcard: cur.Wildcard,
		IsRequestEnabled: cur.IsRequestEnabled, IsResponseEnabled: cur.IsResponseEnabled,
		IsEnabled: cur.IsEnabled,
	}
	if hasArg(args, "wildcard") {
		e.Wildcard = boolArg(args, "wildcard", e.Wildcard)
	}
	if hasArg(args, "isRequestEnabled") {
		e.IsRequestEnabled = boolArg(args, "isRequestEnabled", e.IsRequestEnabled)
	}
	if hasArg(args, "isResponseEnabled") {
		e.IsResponseEnabled = boolArg(args, "isResponseEnabled", e.IsResponseEnabled)
	}
	if hasArg(args, "isEnabled") {
		e.IsEnabled = boolArg(args, "isEnabled", e.IsEnabled)
	}
	if err := store.UpdateBreakpoint(ud(args), e); err != nil {
		return "", err
	}
	return fmt.Sprintf("breakpoint %s updated -> %s", id, restartNote), nil
}

func handleBreakpointSetItemEnabled(args map[string]any) (string, error) {
	id, enabled, err := reqIDFlag(args, "enabled")
	if err != nil {
		return "", err
	}
	if err := store.SetBreakpointItemEnabled(ud(args), id, enabled); err != nil {
		return "", err
	}
	return fmt.Sprintf("breakpoint %s enabled=%v -> %s", id, enabled, restartNote), nil
}
