package main

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// def builds one tool definition.
func def(name, desc string, schema map[string]any) toolDef {
	return toolDef{Name: name, Description: desc, InputSchema: schema}
}

// toolDefs lists 75 tools.
func toolDefs() []toolDef {
	return []toolDef{
		def("rewrite_list", "List Reqable rewrite rules (name, url, id).", obj(map[string]any{})),
		def("rewrite_create", "Create a Replace-Response rewrite rule (mock a URL with a JSON body). Restart Reqable after to load it.",
			obj(map[string]any{
				"name": str("Rule name"), "url": str("Match URL, e.g. https://api.example.com/*"),
				"payload": str("Response body"), "method": str("HTTP method, empty = any"),
				"wildcard": boolean("Wildcard match", true),
			}, "name", "url")),
		def("rewrite_modify_create", "Create a Modify rewrite rule (regex replace on matching traffic). Restart Reqable after.",
			obj(map[string]any{
				"name": str("Rule name"), "url": str("Match URL"),
				"pattern": str("Regex pattern"), "replacement": str("Replacement"),
			}, "name", "url", "pattern", "replacement")),
		def("rewrite_delete", "Delete a rewrite rule by id. Restart Reqable after.", obj(map[string]any{"id": str("Rewrite id")}, "id")),
		def("rewrite_get_config", "Return the full rewrite JSON config.", obj(map[string]any{})),
		def("rewrite_set_enabled", "Toggle the top-level rewrite switch.", obj(map[string]any{"enabled": boolean("Enable rewrites", true)}, "enabled")),
		def("rewrite_get_by_id", "Get a rewrite rule by id.", obj(map[string]any{"id": str("Rewrite id")}, "id")),
		def("rewrite_update", "Replace a rewrite rule in full (action is a raw JSON object).",
			obj(map[string]any{
				"id": str("Rewrite id"), "name": str("Rule name"), "method": str("HTTP method"),
				"url": str("Match URL"), "wildcard": boolean("Wildcard match", true),
				"action": rawObj("Raw action JSON object"), "isEnabled": boolean("Rule enabled", true),
			}, "id")),
		def("rewrite_set_item_enabled", "Toggle a single rewrite rule.", obj(map[string]any{"id": str("Rewrite id"), "enabled": boolean("Enable rule", true)}, "id", "enabled")),
		def("breakpoint_list", "List Reqable breakpoint rules.", obj(map[string]any{})),
		def("breakpoint_create", "Create a breakpoint rule (pause matching traffic). Restart Reqable after.",
			obj(map[string]any{
				"name": str("Rule name"), "url": str("Match URL, e.g. https://api.example.com/*"),
			}, "name", "url")),
		def("breakpoint_delete", "Delete a breakpoint rule by id. Restart Reqable after.", obj(map[string]any{"id": str("Breakpoint id")}, "id")),
		def("breakpoint_get_config", "Return the full breakpoint JSON config.", obj(map[string]any{})),
		def("breakpoint_set_enabled", "Toggle the top-level breakpoint switch.", obj(map[string]any{"enabled": boolean("Enable breakpoints", true)}, "enabled")),
		def("breakpoint_get_by_id", "Get a breakpoint rule by id.", obj(map[string]any{"id": str("Breakpoint id")}, "id")),
		def("breakpoint_update", "Replace a breakpoint rule in full.",
			obj(map[string]any{
				"id": str("Breakpoint id"), "name": str("Rule name"), "method": str("HTTP method"),
				"url": str("Match URL"), "wildcard": boolean("Wildcard match", true),
				"isRequestEnabled":  boolean("Break on request", true),
				"isResponseEnabled": boolean("Break on response", true),
				"isEnabled":         boolean("Rule enabled", true),
			}, "id")),
		def("breakpoint_set_item_enabled", "Toggle a single breakpoint rule.", obj(map[string]any{"id": str("Breakpoint id"), "enabled": boolean("Enable rule", true)}, "id", "enabled")),
		def("collection_list", "List Reqable collections.", obj(map[string]any{})),
		def("collection_create", "Create an empty Reqable collection. Restart Reqable after.", obj(map[string]any{"name": str("Collection name")}, "name")),
		def("collection_get", "Get one collection object by id.", obj(map[string]any{"id": str("Collection id")}, "id")),
		def("collection_delete", "Delete a collection by id.", obj(map[string]any{"id": str("Collection id")}, "id")),
		def("collection_update", "Update a collection (optional name).", obj(map[string]any{"id": str("Collection id"), "name": str("New name")}, "id")),
		def("collection_structure", "Return the full collections JSON (pretty).", obj(map[string]any{})),
		def("collection_folder_create", "Create a folder ref inside a collection.",
			obj(map[string]any{
				"collectionId": str("Collection id"), "name": str("Folder name"), "parentId": str("Parent folder id"),
			}, "collectionId", "name")),
		def("collection_folder_get", "Get a folder ref by id within a collection.", obj(map[string]any{"collectionId": str("Collection id"), "id": str("Folder id")}, "collectionId", "id")),
		def("collection_folder_update", "Rename a folder ref within a collection.", obj(map[string]any{"collectionId": str("Collection id"), "id": str("Folder id"), "name": str("New name")}, "collectionId", "id", "name")),
		def("collection_folder_delete", "Delete a folder ref and its descendants within a collection.", obj(map[string]any{"collectionId": str("Collection id"), "id": str("Folder id")}, "collectionId", "id")),
		def("collection_api_create", "Create an API in a collection from a curl command (sidecar for manual import).",
			obj(map[string]any{
				"collectionId": str("Collection id"), "curl": str("Curl command"), "parentId": str("Parent folder id"),
			}, "collectionId", "curl")),
		def("collection_api_add", "Add an API object to a collection.",
			obj(map[string]any{
				"collectionId": str("Collection id"), "api": rawObj("API object"), "parentId": str("Parent folder id"),
			}, "collectionId", "api")),
		def("collection_api_get", "Get a stored API ref by id within a collection.", obj(map[string]any{"collectionId": str("Collection id"), "id": str("API id")}, "collectionId", "id")),
		def("collection_api_update", "Overwrite the sidecar for a sidecar-managed API.", obj(map[string]any{"collectionId": str("Collection id"), "api": rawObj("Full API object with id")}, "collectionId", "api")),
		def("collection_api_delete", "Delete an API ref (and sidecar) from a collection.", obj(map[string]any{"collectionId": str("Collection id"), "id": str("API id")}, "collectionId", "id")),
		def("collection_api_generate_curl", "Build a curl command for a stored API.", obj(map[string]any{"collectionId": str("Collection id"), "id": str("API id")}, "collectionId", "id")),
		def("script_create", "Stage a Reqable script definition as a JSON sidecar for manual import (scripts live in the object store).",
			obj(map[string]any{
				"name": str("Script name"), "url": str("Match URL"), "code": str("Script source"),
			}, "name", "url", "code")),
		def("script_get_config", "Return script config (honest error: scripts live in the object store).", obj(map[string]any{})),
		def("script_set_enabled", "Toggle scripts (honest stub).", obj(map[string]any{"enabled": boolean("Enable scripts", true)}, "enabled")),
		def("script_get_by_id", "Get a script by id (honest stub).", obj(map[string]any{"id": str("Script id")}, "id")),
		def("script_update", "Update a script by id (honest stub).",
			obj(map[string]any{
				"id": str("Script id"), "script": rawObj("Script object"),
				"name": str("Script name"), "url": str("Match URL"), "code": str("Script source"),
			}, "id")),
		def("script_set_item_enabled", "Toggle a single script (honest stub).", obj(map[string]any{"id": str("Script id"), "enabled": boolean("Enable script", true)}, "id", "enabled")),
		def("script_list", "List scripts (honest stub).", obj(map[string]any{})),
		def("script_framework", "Return a minimal Reqable script framework snippet.", obj(map[string]any{})),
		def("script_template", "Return a basic passthrough script template.", obj(map[string]any{})),
		def("rest_http_create_from_url", "Stage an HTTP tab draft from a URL.", obj(map[string]any{"name": str("Tab name"), "url": str("URL")}, "name", "url")),
		def("rest_http_create_from_curl", "Stage an HTTP tab draft parsed from curl.", obj(map[string]any{"name": str("Tab name"), "curl": str("Curl command")}, "name", "curl")),
		def("rest_http_update", "Overwrite a staged HTTP tab.", obj(map[string]any{"id": str("Tab id"), "tab": rawObj("Full tab object")}, "id", "tab")),
		def("rest_http_get", "Load a staged HTTP tab.", obj(map[string]any{"id": str("Tab id")}, "id")),
		def("rest_websocket_create_from_url", "Stage a websocket tab draft from a URL.", obj(map[string]any{"name": str("Tab name"), "url": str("URL")}, "name", "url")),
		def("rest_websocket_update", "Overwrite a staged websocket tab.", obj(map[string]any{"id": str("Tab id"), "tab": rawObj("Full tab object")}, "id", "tab")),
		def("rest_websocket_get", "Load a staged websocket tab.", obj(map[string]any{"id": str("Tab id")}, "id")),
		def("proxy_set", "Set the system-proxy flag in global capture_config (proxySystem).", obj(map[string]any{"overrideSystem": boolean("Enable system proxy", true)}, "overrideSystem")),
		def("capture_live_status", "Report live capture status (app running, capture files, box size).", obj(map[string]any{})),
		def("capture_live_set_enabled", "Set autoCapture in global capture_config.", obj(map[string]any{"enabled": boolean("Enable auto capture", true)}, "enabled")),
		def("capture_live_filter", "Filter live capture with keyword/host/method/code shapes (up to 100 IDs).", obj(map[string]any{"filters": arr("Filter objects: keyword{pattern}, host{hosts[]}, method{methods[]}, code{codes[]}")})),
		def("capture_live_get_by_id", "Get a live capture record by numeric id.", obj(map[string]any{"id": intSchema("Record id", 0)}, "id")),
		def("capture_live_generate_curl", "Generate curl for a live capture record.", obj(map[string]any{"id": intSchema("Record id", 0)}, "id")),
		def("capture_live_compose", "Stage a REST tab from a live capture record.", obj(map[string]any{"id": intSchema("Record id", 0)}, "id")),
		def("capture_live_collection_add", "Stage a capture record into a collection.",
			obj(map[string]any{
				"id": intSchema("Record id", 0), "collectionId": str("Collection id"),
				"name": str("API name"), "parentId": str("Parent folder id"),
			}, "id", "collectionId")),
		def("capture_live_clear", "Explain that retained records must be cleared in the Reqable UI.", obj(map[string]any{})),
		def("capture_list_recent", "List recent capture records with optional host/method/status filters.",
			obj(map[string]any{
				"host": str("Host substring"), "method": str("HTTP method"),
				"status": intSchema("Status code (0 = any)", 0), "limit": intSchema("Max records", 20),
			})),
		def("capture_get_request", "Get a capture record by id (full includes body text up to 4KB).", obj(map[string]any{"id": intSchema("Record id", 0), "full": boolean("Include body", false)}, "id")),
		def("capture_search_url", "Search capture URLs by substring or regex.",
			obj(map[string]any{
				"pattern": str("Search pattern"), "regex": boolean("Treat as regex", false), "limit": intSchema("Max results", 50),
			}, "pattern")),
		def("capture_search_body", "Search response bodies (up to 200 records).", obj(map[string]any{"pattern": str("Search pattern"), "limit": intSchema("Max results", 20)}, "pattern")),
		def("capture_stats", "Count records by host/method/status over the last N.", obj(map[string]any{"window": intSchema("Last N records", 200)})),
		def("capture_diff", "Diff two capture records (method/url/status/headers).", obj(map[string]any{"idA": intSchema("First id", 0), "idB": intSchema("Second id", 0)}, "idA", "idB")),
		def("capture_find_dynamic_fields", "Find header/query keys whose values vary across a host sample.", obj(map[string]any{"host": str("Host substring"), "sample": intSchema("Sample size", 20)}, "host")),
		def("capture_extract_auth", "Collect distinct auth header values for a host.", obj(map[string]any{"host": str("Host substring")}, "host")),
		def("capture_decode_jwt", "Decode a JWT header and payload without verification.", obj(map[string]any{"token": str("JWT")}, "token")),
		def("capture_to_curl", "Render a curl command for a capture record.", obj(map[string]any{"id": intSchema("Record id", 0)}, "id")),
		def("capture_wait_for", "Poll capture until a new matching record appears or timeout.",
			obj(map[string]any{
				"host": str("Host substring"), "method": str("HTTP method"), "path": str("URL path substring"),
				"timeout": intSchema("Timeout seconds", 30),
			})),
		def("capture_list_apps", "List distinct capture hosts as an app-ish list.", obj(map[string]any{})),
		def("capture_get_domains", "Return top domains by capture count.", obj(map[string]any{})),
		def("capture_analyze_api", "Infer method/path/status shapes for a domain.", obj(map[string]any{"domain": str("Domain/host")}, "domain")),
		def("capture_generate_code", "Generate sample client code (python|curl|go) for a record.", obj(map[string]any{"id": intSchema("Record id", 0), "lang": str("python|curl|go")}, "id")),
		def("capture_import_har", "Parse a HAR file and stage each entry as a REST tab.", obj(map[string]any{"path": str("HAR file path")}, "path")),
		def("capture_ingest_status", "Report box key count, capture file count, and db size.", obj(map[string]any{})),
	}
}
