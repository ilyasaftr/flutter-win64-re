# reqable-mcp

MCP server (Go, stdlib only, no dependencies) exposing Reqable control as
73 MCP tools. One stdio binary works with Claude Code, Codex, Grok Build,
OpenCode, and any MCP client.

The stock Reqable `mcp-server` rejects writes without premium
(`Creating rewrite is a premium feature...`). This server writes the app
JSON config files directly and reads capture traffic straight from Box
LMDB instead. Restart Reqable after a write — the app caches config in
Box LMDB at startup.

## Build

```sh
go -C tools/reqable-mcp build -o ../../build/reqable-mcp .
# -> build/reqable-mcp (arm64, ~5M, static, no deps)
```

## Tools (73)

Full stock parity (46 tools, same names minus the `reqable__` prefix so
both servers can coexist) plus 27 capture/analysis tools in the
Wanghaibo10 + ElonJask style.

### Rewrite (9)

| tool | what |
|---|---|
| `rewrite_list` | list rules |
| `rewrite_create` | Replace-Response rule: `name`, `url`, `payload`, `method`, `wildcard` |
| `rewrite_modify_create` | Modify rule: `name`, `url`, `pattern`, `replacement` |
| `rewrite_delete` | delete by `id` |
| `rewrite_get_config` | full JSON config |
| `rewrite_set_enabled` | feature on/off |
| `rewrite_get_by_id` | one rule |
| `rewrite_update` | full replace: `id`, `name`, `method`, `url`, `wildcard`, `action`, `isEnabled` |
| `rewrite_set_item_enabled` | rule on/off |

### Breakpoint (7)

`breakpoint_list`, `breakpoint_create`, `breakpoint_delete`,
`breakpoint_get_config`, `breakpoint_set_enabled`, `breakpoint_get_by_id`,
`breakpoint_update`, `breakpoint_set_item_enabled` — same shapes as rewrite.

### Script (9)

`script_create` (stages JSON sidecar for manual import, scripts live in the
object store), `script_get_config`, `script_set_enabled`, `script_get_by_id`,
`script_update`, `script_set_item_enabled`, `script_list`,
`script_framework`, `script_template` (embedded passthrough template).

### Collection (14)

`collection_list`, `collection_create`, `collection_get`,
`collection_delete`, `collection_update`, `collection_structure`,
`collection_folder_create`, `collection_folder_get`,
`collection_api_create` (from curl), `collection_api_add`,
`collection_api_get`, `collection_api_update`, `collection_api_delete`,
`collection_api_generate_curl`.

### REST tabs (7)

`rest_http_create_from_url`, `rest_http_create_from_curl`,
`rest_http_update`, `rest_http_get`,
`rest_websocket_create_from_url`, `rest_websocket_update`,
`rest_websocket_get`. Tabs stage under `rest/staged/` for manual import;
live tabs live in Box.

### Proxy (1)

`proxy_set` (`overrideSystem`) — direct-writes `proxySystem` in global
`capture_config`.

### Live capture (8)

`capture_live_status`, `capture_live_set_enabled` (writes `autoCapture`),
`capture_live_filter` (keyword/url/host/method/code), `capture_live_get_by_id`,
`capture_live_generate_curl`, `capture_live_compose` (stage as REST tab),
`capture_live_collection_add`, `capture_live_clear` (honest: retained records
live in Box, clear in UI — never deletes LMDB).

### Traffic reads (18, Wanghaibo10 + ElonJask style)

| tool | what |
|---|---|
| `capture_list_recent` | recent records, filter `host`/`method`/`status`/`limit` |
| `capture_get_request` | one record, `full` includes body text |
| `capture_search_url` | URL substring/regex |
| `capture_search_body` | full-text over response bodies |
| `capture_stats` | host/method/status distribution |
| `capture_diff` | field diff of two records |
| `capture_find_dynamic_fields` | varying header/query keys (signatures) |
| `capture_extract_auth` | Authorization/Cookie/token values |
| `capture_decode_jwt` | unverified JWT decode |
| `capture_to_curl` | record as curl |
| `capture_wait_for` | poll for next match (`host`/`method`/`path`/`timeout`) |
| `capture_list_apps` | distinct origins/hosts |
| `capture_get_domains` | top domains by count |
| `capture_analyze_api` | method/path/status shapes per domain |
| `capture_generate_code` | sample client (`curl`/`python`/`go`) |
| `capture_import_har` | HAR file import, stages REST tabs |
| `capture_ingest_status` | box keys + capture files + db size |
| `capture_live_*` | (above) live counterparts |

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

## Verified

- `initialize` returns `2024-11-05`; `tools/list` returns all 73.
- Mutate round-trip: create → get_by_id → update → set_item_enabled →
  delete for rewrite and breakpoint, plus collection/REST/live/capture reads.
- Live traffic: 473 capture records decoded from Box (base64 gzip JSON),
  `capture_list_recent` returns real rows (e.g. `GET https://www.google.com/robots.txt 304`).
- User data untouched: 4 rewrites + 1 breakpoint + 1 collection intact.

## Notes

- Writes are atomic (tmp + rename). The active user dir is auto-selected
  (most rewrites wins), not hardcoded.
- Restart Reqable after writes: `pkill -f ReqableDYLD`, then relaunch with
  `DYLD_INSERT_LIBRARIES` (see `ReqableMacHack/README.md`).
- Capture reads need `python3` + `lmdb` module at runtime (bridge for the
  LMDB dump); everything else is pure Go.
- Gateway/mirror/SSL/throttling/secondary-proxy/access-control/environment/
  report-server groups from third-party READMEs do not exist on the stock
  46-tool server for this build, so no tools are provided for them.
