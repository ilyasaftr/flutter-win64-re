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

// stageTab writes a new staged tab draft and returns its id.
func stageTab(userDir string, path func(userDir, id string) string, tab map[string]any) (string, error) {
	id := mustUUID()
	tab["id"] = id
	if err := os.MkdirAll(stagedDir(userDir), 0o755); err != nil {
		return "", err
	}
	if err := saveFile(path(userDir, id), tab); err != nil {
		return "", err
	}
	return id, nil
}

// updateTab overwrites a staged tab; errors when missing.
func updateTab(userDir, id, kind string, path func(userDir, id string) string, tab map[string]any) error {
	p := path(userDir, id)
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("staged %s tab %s not found", kind, id)
	}
	tab["id"] = id
	return saveFile(p, tab)
}

// loadTab loads a staged tab.
func loadTab(userDir, id string, path func(userDir, id string) string) (map[string]any, error) {
	var tab map[string]any
	if err := loadFile(path(userDir, id), &tab); err != nil {
		return nil, err
	}
	return tab, nil
}

// RestHTTPCreateFromURL stages a GET tab draft and returns its id.
func RestHTTPCreateFromURL(userDir, name, url string) (string, error) {
	return stageTab(userDir, httpStagedPath, buildAPIObject("", name, "GET", url, nil, ""))
}

// RestHTTPCreateFromCurl stages a tab draft parsed from curl.
func RestHTTPCreateFromCurl(userDir, name, curl string) (string, error) {
	method, url, headers, body := ParseCurl(curl)
	if url == "" {
		return "", fmt.Errorf("no URL found in curl command")
	}
	return stageTab(userDir, httpStagedPath, buildAPIObject("", name, method, url, headers, body))
}

// RestHTTPUpdate overwrites a staged tab; errors when missing.
func RestHTTPUpdate(userDir, id string, tab map[string]any) error {
	return updateTab(userDir, id, "HTTP", httpStagedPath, tab)
}

// RestHTTPGet loads a staged tab.
func RestHTTPGet(userDir, id string) (map[string]any, error) {
	return loadTab(userDir, id, httpStagedPath)
}

func wsTab(id, name, url string, headers [][2]string) map[string]any {
	tab := buildTab(id, name, url, headers, "websocket")
	tab["sendType"] = "text"
	return tab
}

// RestWSCreateFromURL stages a websocket tab draft and returns its id.
func RestWSCreateFromURL(userDir, name, url string) (string, error) {
	return stageTab(userDir, wsStagedPath, wsTab("", name, url, nil))
}

// RestWSUpdate overwrites a staged websocket tab; errors when missing.
func RestWSUpdate(userDir, id string, tab map[string]any) error {
	return updateTab(userDir, id, "websocket", wsStagedPath, tab)
}

// RestWSGet loads a staged websocket tab.
func RestWSGet(userDir, id string) (map[string]any, error) {
	return loadTab(userDir, id, wsStagedPath)
}
