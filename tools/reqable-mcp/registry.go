package main

import (
	"encoding/json"

	"reqable-mcp/store"
)

// Handler runs one tool: pure (string, error) over the decoded args.
// The injected "\x00userDir" key carries the resolved Reqable user dir;
// ud() reads it. Only -32602 arg errors use *toolErr; anything else
// becomes -32000.
type Handler func(args map[string]any) (string, error)

// handlers maps all 73 tool names to their implementations. Each
// handlers_*.go file registers its own tools from init().
var handlers = map[string]Handler{}

// callTool dispatches tools/call: parse params, look up the handler,
// run it, and wrap the result.
func callTool(req rpcRequest) rpcResponse {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return fail(req.ID, -32602, "invalid params")
	}
	h, found := handlers[params.Name]
	if !found {
		return fail(req.ID, -32601, "unknown tool: "+params.Name)
	}
	args := params.Arguments
	if args == nil {
		args = map[string]any{}
	}
	dir, err := store.UserDir()
	if err != nil {
		return fail(req.ID, -32000, err.Error())
	}
	args["\x00userDir"] = dir
	text, err := h(args)
	if err != nil {
		if terr, isArg := err.(*toolErr); isArg {
			return fail(req.ID, terr.code, terr.msg)
		}
		return fail(req.ID, -32000, err.Error())
	}
	return ok(req.ID, textResult(text))
}
