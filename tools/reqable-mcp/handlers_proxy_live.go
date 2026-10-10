package main

import (
	"fmt"
	"reqable-mcp/capture"
	"reqable-mcp/store"
)

func init() {
	handlers["proxy_set"] = handleProxySet
	handlers["capture_live_status"] = handleCaptureLiveStatus
	handlers["capture_live_set_enabled"] = handleCaptureLiveSetEnabled
	handlers["capture_live_filter"] = handleCaptureLiveFilter
	handlers["capture_live_get_by_id"] = handleCaptureLiveGetById
	handlers["capture_live_generate_curl"] = handleCaptureLiveGenerateCurl
	handlers["capture_live_compose"] = handleCaptureLiveCompose
	handlers["capture_live_collection_add"] = handleCaptureLiveCollectionAdd
	handlers["capture_live_clear"] = handleCaptureLiveClear
}

func handleProxySet(args map[string]any) (string, error) {
	key := "overrideSystem"
	if !hasArg(args, key) && hasArg(args, "enabled") {
		key = "enabled"
	}
	if !hasArg(args, key) {
		return "", badArgs("overrideSystem is required")
	}
	enabled := boolArg(args, key, false)
	if err := setCaptureFlag("proxySystem", enabled); err != nil {
		return "", err
	}
	return fmt.Sprintf("proxySystem=%v -> restart or toggle in UI", enabled), nil
}

func handleCaptureLiveStatus(args map[string]any) (string, error) {
	keys, files, size := boxStats()
	status := map[string]any{
		"appRunning": reqableRunning(), "boxKeys": keys,
		"captureFiles": files, "boxBytes": size,
		"note": "capture dir file count + box size; restart Reqable after config writes",
	}
	return marshalPretty(status), nil
}

func handleCaptureLiveSetEnabled(args map[string]any) (string, error) {
	if !hasArg(args, "enabled") {
		return "", badArgs("enabled is required")
	}
	enabled := boolArg(args, "enabled", false)
	if err := setCaptureFlag("autoCapture", enabled); err != nil {
		return "", err
	}
	return fmt.Sprintf("autoCapture=%v -> restart Reqable to load", enabled), nil
}

// setCaptureFlag flips one key in the global capture_config atomically.
func setCaptureFlag(key string, enabled bool) error {
	path, err := globalCaptureConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadJSONMap(path)
	if err != nil {
		return err
	}
	cfg[key] = enabled
	return saveJSONAtomic(path, cfg)
}

func handleCaptureLiveFilter(args map[string]any) (string, error) {
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	var keyword, host, method string
	var code int
	if arr, ok := args["filters"].([]any); ok {
		for _, f := range arr {
			m, ok := f.(map[string]any)
			if !ok {
				continue
			}
			if keyword == "" {
				keyword = firstStr(m, "pattern", "keyword")
			}
			if host == "" {
				host = firstStr(m, "host")
				if host == "" {
					host, _ = firstListStr(m, "hosts")
				}
			}
			if method == "" {
				method = firstStr(m, "method")
				if method == "" {
					method, _ = firstListStr(m, "methods")
				}
			}
			if code == 0 {
				code = firstInt(m, "code", "codes")
			}
		}
	}
	matched := capture.Filter(recs, capture.FilterSpec{Keyword: keyword, Host: host, Method: method, Code: code})
	ids := []int{}
	for _, r := range matched {
		if len(ids) >= 100 {
			break
		}
		ids = append(ids, r.ID)
	}
	return marshalPretty(map[string]any{"ids": ids, "count": len(matched)}), nil
}

func handleCaptureLiveGetById(args map[string]any) (string, error) {
	r, err := reqRecord(args)
	if err != nil {
		return "", err
	}
	return marshalPretty(r), nil
}

func handleCaptureLiveGenerateCurl(args map[string]any) (string, error) {
	r, err := reqRecord(args)
	if err != nil {
		return "", err
	}
	return capture.ToCurl(r, nil), nil
}

func handleCaptureLiveCompose(args map[string]any) (string, error) {
	r, err := reqRecord(args)
	if err != nil {
		return "", err
	}
	name := r.Method + " " + r.URL
	if r.Method == "" {
		name = r.URL
	}
	id, err := store.RestHTTPCreateFromURL(ud(args), name, r.URL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("staged %s", id), nil
}

func handleCaptureLiveCollectionAdd(args map[string]any) (string, error) {
	collectionID := strArg(args, "collectionId", "")
	if !hasArg(args, "id") || collectionID == "" {
		return "", badArgs("id and collectionId are required")
	}
	r, err := capture.GetByID(intArg(args, "id", 0))
	if err != nil {
		return "", err
	}
	name := strArg(args, "name", "")
	if name == "" {
		name = r.Method + " " + r.URL
	}
	method := r.Method
	if method == "" {
		method = "GET"
	}
	api := map[string]any{"name": name, "method": method, "url": map[string]any{"base": r.URL}}
	id, err := store.CollectionAPIAdd(ud(args), collectionID, api, strArg(args, "parentId", ""))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("api %s staged -> %s", id, restartNote), nil
}

func handleCaptureLiveClear(args map[string]any) (string, error) {
	return "retained records live in Box; clear in Reqable UI", nil
}
