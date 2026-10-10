// Command reqable-mcp serves the reqable-writer operations as a Model
// Context Protocol server over stdio, so any MCP client (Claude Code,
// Codex, Grok Build, OpenCode) can create Reqable rewrites, breakpoints,
// collections, and scripts without hitting the stock premium gate.
//
// The server speaks JSON-RPC 2.0 with Content-Length framing on stdin and
// stdout (LSP-style headers), implementing the MCP 2024-11-05 surface:
// initialize, notifications/initialized, tools/list, tools/call, and
// ping. No external dependencies; stdlib only.
//
// Writes go straight to the app JSON config files; restart Reqable after
// a write so it reloads config from disk (Box LMDB cache).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	mcpVersion = "2024-11-05"
	serverName = "reqable-writer"
	serverVers = "1.0.0"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcErr `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	for {
		msg, err := readFrame(in)
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}
		var req rpcRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if req.Method == "notifications/initialized" {
			continue // no response for notifications
		}
		resp := dispatch(req)
		if req.ID == nil {
			continue
		}
		raw, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		fmt.Fprintf(out, "Content-Length: %d\r\n\r\n%s", len(raw), raw)
		out.Flush()
	}
}

// readFrame reads one LSP-style Content-Length framed message.
func readFrame(r *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			n, err := strconv.Atoi(strings.TrimSpace(line[len("content-length:"):]))
			if err != nil {
				return nil, err
			}
			length = n
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func ok(id any, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func fail(id any, code int, msg string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcErr{Code: code, Message: msg}}
}

func dispatch(req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return ok(req.ID, map[string]any{
			"protocolVersion": mcpVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": serverName, "version": serverVers},
		})
	case "ping":
		return ok(req.ID, map[string]any{})
	case "tools/list":
		return ok(req.ID, map[string]any{"tools": toolDefs()})
	case "tools/call":
		return callTool(req)
	default:
		return fail(req.ID, -32601, "method not found: "+req.Method)
	}
}
