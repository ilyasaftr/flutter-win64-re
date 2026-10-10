# reqable-mcp

MCP server (Go, stdlib only, no dependencies) exposing the direct-write
bypass as MCP tools. One stdio binary works with Claude Code, Codex,
Grok Build, OpenCode, and any MCP client.

The stock Reqable `mcp-server` rejects writes without premium
(`Creating rewrite is a premium feature...`). This server writes the app
JSON config files directly instead. Restart Reqable after a write — the
app caches config in Box LMDB at startup.

## Build

```sh
go -C tools/reqable-mcp build -o ../../build/reqable-mcp .
# -> build/reqable-mcp (arm64, ~4M, static, no deps)
```

## Tools (10)

| tool | what |
|---|---|
| `rewrite_list` | list rewrite rules |
| `rewrite_create` | Replace-Response rule (mock URL with body): `name`, `url`, `payload`, `method`, `wildcard` |
| `rewrite_modify_create` | Modify rule (regex replace): `name`, `url`, `pattern`, `replacement` |
| `rewrite_delete` | delete rewrite by `id` |
| `breakpoint_list` | list breakpoints |
| `breakpoint_create` | breakpoint rule: `name`, `url` |
| `breakpoint_delete` | delete breakpoint by `id` |
| `collection_list` | list collections |
| `collection_create` | empty collection: `name` |
| `script_create` | stage script as JSON sidecar for manual import: `name`, `url`, `code` |

## Register with clients

All clients use stdio with the same binary. Use an absolute path.

```sh
BIN=$PWD/build/reqable-mcp
```

**Claude Code:**

```sh
claude mcp add reqable-writer -- "$BIN"
claude mcp list   # verify
```

**Codex CLI** (`~/.codex/config.toml`):

```toml
[mcp_servers.reqable-writer]
command = "/abs/path/to/build/reqable-mcp"
```

**Grok Build** (`.grok/config.json` or MCP settings):

```json
{ "mcpServers": { "reqable-writer": { "command": "/abs/path/to/build/reqable-mcp" } } }
```

**OpenCode** (`opencode.json`):

```json
{ "mcp": { "reqable-writer": { "type": "local", "command": ["/abs/path/to/build/reqable-mcp"] } } }
```

**Any other MCP client:** stdio command = `/abs/path/to/build/reqable-mcp`,
no args, no env needed.

## Protocol test

```sh
python3 /tmp/mcp_test.py  # initialize -> tools/list (10) -> create/delete round-trip
```

Verified: `initialize` returns `2024-11-05`, `tools/list` returns all 10,
`rewrite_create` + `breakpoint_create` + `collection_create` round-trip
through the real config files and clean up.

## Notes

- Writes are atomic (tmp + rename). The active user dir is auto-selected
  (most rewrites wins), not hardcoded.
- Restart Reqable after writes: `pkill -f ReqableDYLD`, then relaunch with
  `DYLD_INSERT_LIBRARIES` (see `ReqableMacHack/README.md`).
- The `script_create` tool stages a sidecar (`config/capture_script_grok_<id>.json`)
  because scripts live in the object store; import it in the Reqable UI.
