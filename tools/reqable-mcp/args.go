package main

import (
	"reqable-mcp/capture"
	"strconv"
	"strings"
)

const restartNote = "restart Reqable to load"

// toolErr carries an MCP error code; any other error from a Handler
// surfaces as -32000 (server error).
type toolErr struct {
	code int
	msg  string
}

func (e *toolErr) Error() string { return e.msg }

func badArgs(msg string) *toolErr { return &toolErr{code: -32602, msg: msg} }

func ud(args map[string]any) string {
	s, _ := args["\x00userDir"].(string)
	return s
}

func hasArg(args map[string]any, k string) bool {
	_, ok := args[k]
	return ok
}

func strArg(args map[string]any, k, def string) string {
	if s, ok := args[k].(string); ok {
		return s
	}
	return def
}

func boolArg(args map[string]any, k string, def bool) bool {
	if v, ok := args[k].(bool); ok {
		return v
	}
	return def
}

func intArg(args map[string]any, k string, def int) int {
	return intOf(args[k], def)
}

func objArg(args map[string]any, k string) map[string]any {
	if m, ok := args[k].(map[string]any); ok {
		return m
	}
	return nil
}

// posArg reads a positive int arg, falling back to def when missing or
// non-positive.
func posArg(args map[string]any, k string, def int) int {
	if n := intArg(args, k, def); n > 0 {
		return n
	}
	return def
}

// firstStr returns the first non-empty string among keys.
func firstStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// firstListStr returns the first non-empty string in list values.
func firstListStr(m map[string]any, keys ...string) (string, bool) {
	for _, k := range keys {
		if arr, ok := m[k].([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok && s != "" {
					return s, true
				}
			}
		}
	}
	return "", false
}

// firstInt reads the first non-zero int among keys or their lists:
// for each key it tries the scalar, then each list element.
func firstInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if n := intArg(m, k, 0); n != 0 {
			return n
		}
		if arr, ok := m[k].([]any); ok {
			for _, v := range arr {
				if n := intOf(v, 0); n != 0 {
					return n
				}
			}
		}
	}
	return 0
}

// intOf coerces a decoded JSON value to int.
func intOf(v any, def int) int {
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

func reqStr(args map[string]any, k string) (string, error) {
	if s := strArg(args, k, ""); s != "" {
		return s, nil
	}
	return "", badArgs(k + " is required")
}

func reqID(args map[string]any) (string, error) {
	return reqStr(args, "id")
}

// reqHost extracts a required "host" arg.
func reqHost(args map[string]any) (string, error) {
	return reqStr(args, "host")
}

// reqPattern extracts a required "pattern" arg.
func reqPattern(args map[string]any) (string, error) {
	return reqStr(args, "pattern")
}

// reqNameURL extracts required "name" + "url" args.
func reqNameURL(args map[string]any) (string, string, error) {
	name, url := strArg(args, "name", ""), strArg(args, "url", "")
	if name == "" || url == "" {
		return "", "", badArgs("name and url are required")
	}
	return name, url, nil
}

// reqFlag extracts a required bool flag arg.
func reqFlag(args map[string]any, flag string) (bool, error) {
	if !hasArg(args, flag) {
		return false, badArgs(flag + " is required")
	}
	return boolArg(args, flag, false), nil
}

// reqIDFlag extracts a required "id" plus a required bool flag arg.
func reqIDFlag(args map[string]any, flag string) (string, bool, error) {
	id := strArg(args, "id", "")
	if id == "" || !hasArg(args, flag) {
		return "", false, badArgs("id and " + flag + " are required")
	}
	return id, boolArg(args, flag, false), nil
}

// reqColItem extracts required "collectionId" + "id" pair.
func reqColItem(args map[string]any) (string, string, error) {
	collectionID, id := strArg(args, "collectionId", ""), strArg(args, "id", "")
	if collectionID == "" || id == "" {
		return "", "", badArgs("collectionId and id are required")
	}
	return collectionID, id, nil
}

// reqRecords dumps all capture records.
func reqRecords() ([]capture.Record, error) {
	return capture.List()
}

// reqRecord loads the capture record named by the required int "id" arg.
func reqRecord(args map[string]any) (capture.Record, error) {
	if !hasArg(args, "id") {
		return capture.Record{}, badArgs("id is required")
	}
	return capture.GetByID(intArg(args, "id", 0))
}
