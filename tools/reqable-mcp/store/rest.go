package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// RestHTTPTabs are live tabs in Box (tag 00003c plain JSON); the staged
// dir holds file-backed drafts for manual import.
func stagedDir(userDir string) string {
	return filepath.Join(userDir, "rest", "staged")
}

func httpStagedPath(userDir, id string) string {
	return filepath.Join(stagedDir(userDir), "rest_http_"+id+".json")
}

func wsStagedPath(userDir, id string) string {
	return filepath.Join(stagedDir(userDir), "rest_ws_"+id+".json")
}

// RestHTTPCreateFromURL stages a GET tab draft and returns its id.
func RestHTTPCreateFromURL(userDir, name, url string) (string, error) {
	id := mustUUID()
	if err := os.MkdirAll(stagedDir(userDir), 0o755); err != nil {
		return "", err
	}
	if err := save(httpStagedPath(userDir, id), buildAPIObject(id, name, "GET", url, nil, "")); err != nil {
		return "", err
	}
	return id, nil
}

// RestHTTPCreateFromCurl stages a tab draft parsed from curl.
func RestHTTPCreateFromCurl(userDir, name, curl string) (string, error) {
	p, err := parseCurlCommand(curl)
	if err != nil {
		return "", err
	}
	id := mustUUID()
	if err := os.MkdirAll(stagedDir(userDir), 0o755); err != nil {
		return "", err
	}
	if err := save(httpStagedPath(userDir, id), buildAPIObject(id, name, p.method, p.url, p.headers, p.body)); err != nil {
		return "", err
	}
	return id, nil
}

// RestHTTPUpdate overwrites a staged tab; errors when missing.
func RestHTTPUpdate(userDir, id string, tab map[string]any) error {
	path := httpStagedPath(userDir, id)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("staged HTTP tab %s not found", id)
	}
	tab["id"] = id
	return save(path, tab)
}

// RestHTTPGet loads a staged tab.
func RestHTTPGet(userDir, id string) (map[string]any, error) {
	var tab map[string]any
	if err := load(httpStagedPath(userDir, id), &tab); err != nil {
		return nil, err
	}
	return tab, nil
}

func wsTab(id, name, url string, headers [][2]string) map[string]any {
	if name == "" {
		name = url
	}
	hdrs := []any{}
	for _, h := range headers {
		hdrs = append(hdrs, map[string]any{"key": h[0], "value": h[1]})
	}
	return map[string]any{
		"id": id, "type": "websocket", "name": name,
		"url": map[string]any{"base": url}, "headers": hdrs,
		"sendType":      "text",
		"script":        map[string]any{"isEnabled": false},
		"authorization": map[string]any{"mode": "inherit"},
		"documentation": map[string]any{"content": ""},
	}
}

// RestWSCreateFromURL stages a websocket tab draft and returns its id.
func RestWSCreateFromURL(userDir, name, url string) (string, error) {
	id := mustUUID()
	if err := os.MkdirAll(stagedDir(userDir), 0o755); err != nil {
		return "", err
	}
	if err := save(wsStagedPath(userDir, id), wsTab(id, name, url, nil)); err != nil {
		return "", err
	}
	return id, nil
}

// RestWSUpdate overwrites a staged websocket tab; errors when missing.
func RestWSUpdate(userDir, id string, tab map[string]any) error {
	path := wsStagedPath(userDir, id)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("staged websocket tab %s not found", id)
	}
	tab["id"] = id
	return save(path, tab)
}

// RestWSGet loads a staged websocket tab.
func RestWSGet(userDir, id string) (map[string]any, error) {
	var tab map[string]any
	if err := load(wsStagedPath(userDir, id), &tab); err != nil {
		return nil, err
	}
	return tab, nil
}
