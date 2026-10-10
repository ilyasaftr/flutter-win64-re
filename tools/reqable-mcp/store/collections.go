package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findCollection(cfg *CollectionsFile, id string) int {
	return indexByID(cfg.Collections, id, mapID)
}

func mapID(m map[string]any) string {
	v, _ := m["id"].(string)
	return v
}

func collectionItems(col map[string]any) []any {
	items, _ := col["items"].([]any)
	return items
}

func findItem(items []any, id string) (map[string]any, bool) {
	for _, it := range items {
		if m, ok := it.(map[string]any); ok && mapID(m) == id {
			return m, true
		}
	}
	return nil, false
}

// withCollection loads the collections file, resolves id, runs fn on
// the collection, and saves when fn reports dirty.
func withCollection(userDir, id string, fn func(col map[string]any) (any, bool, error)) (any, error) {
	cfg, err := loadCollections(userDir)
	if err != nil {
		return nil, err
	}
	i := findCollection(&cfg, id)
	if i < 0 {
		return nil, fmt.Errorf("collection %s not found", id)
	}
	out, dirty, err := fn(cfg.Collections[i])
	if err != nil {
		return nil, err
	}
	if dirty {
		if err := saveCollections(userDir, cfg); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// CollectionGet returns one raw collection object.
func CollectionGet(userDir, id string) (map[string]any, error) {
	out, err := withCollection(userDir, id, func(col map[string]any) (any, bool, error) {
		return col, false, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(map[string]any), nil
}

// CollectionDelete removes a collection; returns found.
func CollectionDelete(userDir, id string) (bool, error) {
	var cfg CollectionsFile
	var err error
	if cfg, err = loadCollections(userDir); err != nil {
		return false, err
	}
	i := findCollection(&cfg, id)
	if i < 0 {
		return false, nil
	}
	cfg.Collections = append(cfg.Collections[:i], cfg.Collections[i+1:]...)
	return true, saveCollections(userDir, cfg)
}

var collectionPatchKeys = map[string]bool{
	"name": true, "query": true, "headers": true,
	"script": true, "authorization": true, "documentation": true,
}

// CollectionUpdate merges patch keys into the stored collection object.
func CollectionUpdate(userDir, id string, patch map[string]any) error {
	_, err := withCollection(userDir, id, func(col map[string]any) (any, bool, error) {
		for k, v := range patch {
			if collectionPatchKeys[k] {
				col[k] = v
			}
		}
		return nil, true, nil
	})
	return err
}

// CollectionFolderCreate appends a folder ref to the collection's items.
// The full folder tree lives in Box LMDB; this records the ref.
func CollectionFolderCreate(userDir, collectionID, name, parentID string) (string, error) {
	id := mustUUID()
	ref := map[string]any{"id": id, "type": "folder", "name": name}
	if parentID != "" {
		ref["parentId"] = parentID
	}
	_, err := withCollection(userDir, collectionID, func(col map[string]any) (any, bool, error) {
		col["items"] = append(collectionItems(col), ref)
		return nil, true, nil
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// CollectionFolderGet finds a folder ref by id within a collection.
func CollectionFolderGet(userDir, collectionID, folderID string) (map[string]any, error) {
	out, err := withCollection(userDir, collectionID, func(col map[string]any) (any, bool, error) {
		ref, ok := findItem(collectionItems(col), folderID)
		if !ok {
			return nil, false, fmt.Errorf("folder %s not found in collection %s", folderID, collectionID)
		}
		if t, _ := ref["type"].(string); t != "folder" {
			return nil, false, fmt.Errorf("item %s is not a folder", folderID)
		}
		return ref, false, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(map[string]any), nil
}

// CollectionAPIGet returns the stored item ref (id/type); bodies live in Box.
func CollectionAPIGet(userDir, collectionID, apiID string) (map[string]any, error) {
	out, err := withCollection(userDir, collectionID, func(col map[string]any) (any, bool, error) {
		ref, ok := findItem(collectionItems(col), apiID)
		if !ok {
			return nil, false, fmt.Errorf("api %s not found in collection %s", apiID, collectionID)
		}
		return ref, false, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(map[string]any), nil
}

// CollectionAPIGetFull is an honest stub: bodies live in Box LMDB.
func CollectionAPIGetFull(userDir, collectionID, apiID string) (map[string]any, error) {
	return nil, fmt.Errorf("full API bodies live in Box LMDB; use capture tools")
}

func buildAPIObject(id, name, method, url string, headers [][2]string, body string) map[string]any {
	tab := buildTab(id, name, url, headers, "api")
	tab["method"] = method
	bodym := map[string]any{"mode": "none"}
	if body != "" {
		bodym = map[string]any{"mode": "text", "text": body}
	}
	tab["body"] = bodym
	return tab
}

// buildTab is the shared skeleton for api/websocket tab objects.
func buildTab(id, name, url string, headers [][2]string, typ string) map[string]any {
	if name == "" {
		name = url
	}
	hdrs := []any{}
	for _, h := range headers {
		hdrs = append(hdrs, map[string]any{"key": h[0], "value": h[1]})
	}
	return map[string]any{
		"id": id, "type": typ, "name": name,
		"url": map[string]any{"base": url}, "headers": hdrs,
		"script":        map[string]any{"isEnabled": false},
		"authorization": map[string]any{"mode": "inherit"},
		"documentation": map[string]any{"content": ""},
	}
}

func apiSidecarPath(userDir, id string) string {
	return filepath.Join(userDir, "rest", "api_"+id+".json")
}

// appendAPIRef records an api ref in the collection; returns its id.
func appendAPIRef(userDir, collectionID, parentID string, api map[string]any) (string, error) {
	id, _ := api["id"].(string)
	if id == "" {
		id = mustUUID()
		api["id"] = id
	}
	ref := map[string]any{"id": id, "type": "api"}
	if parentID != "" {
		ref["parentId"] = parentID
	}
	_, err := withCollection(userDir, collectionID, func(col map[string]any) (any, bool, error) {
		col["items"] = append(collectionItems(col), ref)
		return nil, true, nil
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// CollectionAPICreateFromCurl parses a curl command, appends an api ref,
// and persists the full api JSON as a sidecar for manual import.
func CollectionAPICreateFromCurl(userDir, collectionID, curl, parentID string) (string, error) {
	method, url, headers, body := ParseCurl(curl)
	if url == "" {
		return "", fmt.Errorf("no URL found in curl command")
	}
	id, err := appendAPIRef(userDir, collectionID, parentID, map[string]any{})
	if err != nil {
		return "", err
	}
	if err := saveFile(apiSidecarPath(userDir, id), buildAPIObject(id, url, method, url, headers, body)); err != nil {
		return "", err
	}
	return id, nil
}

// CollectionAPIAdd appends an api ref; api["id"] is used or generated.
func CollectionAPIAdd(userDir, collectionID string, api map[string]any, parentID string) (string, error) {
	return appendAPIRef(userDir, collectionID, parentID, api)
}

// CollectionAPIUpdate overwrites the sidecar for a sidecar-managed API.
func CollectionAPIUpdate(userDir, collectionID string, api map[string]any) error {
	id, _ := api["id"].(string)
	if id == "" {
		return fmt.Errorf("api.id is required")
	}
	if _, err := withCollection(userDir, collectionID, func(col map[string]any) (any, bool, error) {
		return nil, false, nil
	}); err != nil {
		return err
	}
	if _, err := os.Stat(apiSidecarPath(userDir, id)); err != nil {
		return fmt.Errorf("only sidecar-managed APIs can be updated")
	}
	return saveFile(apiSidecarPath(userDir, id), api)
}

// CollectionAPIDelete removes the ref and its sidecar if present.
func CollectionAPIDelete(userDir, collectionID, apiID string) (bool, error) {
	out, err := withCollection(userDir, collectionID, func(col map[string]any) (any, bool, error) {
		items := collectionItems(col)
		kept := make([]any, 0, len(items))
		found := false
		for _, it := range items {
			if m, ok := it.(map[string]any); ok && mapID(m) == apiID {
				found = true
				continue
			}
			kept = append(kept, it)
		}
		if !found {
			return false, false, nil
		}
		col["items"] = kept
		return true, true, nil
	})
	if err != nil {
		return false, err
	}
	if !out.(bool) {
		return false, nil
	}
	if err := os.Remove(apiSidecarPath(userDir, apiID)); err != nil && !os.IsNotExist(err) {
		return true, err
	}
	return true, nil
}

func apiMethod(m map[string]any) string {
	if v, _ := m["method"].(string); v != "" {
		return v
	}
	return "GET"
}

func apiURL(m map[string]any) string {
	switch u := m["url"].(type) {
	case map[string]any:
		if b, _ := u["base"].(string); b != "" {
			return b
		}
	case string:
		if u != "" {
			return u
		}
	}
	if n, _ := m["name"].(string); strings.HasPrefix(n, "http://") || strings.HasPrefix(n, "https://") {
		return n
	}
	return ""
}

func apiHeaders(m map[string]any) [][2]string {
	var out [][2]string
	hdrs, _ := m["headers"].([]any)
	for _, h := range hdrs {
		if hm, ok := h.(map[string]any); ok {
			k, _ := hm["key"].(string)
			if k == "" {
				k, _ = hm["name"].(string)
			}
			v, _ := hm["value"].(string)
			out = append(out, [2]string{k, v})
		}
	}
	return out
}

func apiBodyText(m map[string]any) string {
	b, _ := m["body"].(map[string]any)
	if b == nil {
		return ""
	}
	for _, k := range []string{"text", "raw", "payload"} {
		if s, _ := b[k].(string); s != "" {
			return s
		}
	}
	return ""
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\\''`) + "'"
}

func buildCurl(method, url string, headers [][2]string, body string) string {
	var sb strings.Builder
	sb.WriteString("curl -X " + method + " " + shellQuote(url))
	for _, h := range headers {
		sb.WriteString(" -H " + shellQuote(h[0]+": "+h[1]))
	}
	if body != "" {
		sb.WriteString(" --data-raw " + shellQuote(body))
	}
	return sb.String()
}

// apiCurl renders a curl command from a stored api object.
func apiCurl(m map[string]any) (string, error) {
	u := apiURL(m)
	if u == "" {
		return "", fmt.Errorf("api has no URL")
	}
	return buildCurl(apiMethod(m), u, apiHeaders(m), apiBodyText(m)), nil
}

// CollectionAPIGenerateCurl builds curl from the sidecar, falling back
// to the stored ref when it carries URL details.
func CollectionAPIGenerateCurl(userDir, collectionID, apiID string) (string, error) {
	var full map[string]any
	if err := loadFile(apiSidecarPath(userDir, apiID), &full); err == nil {
		if c, err := apiCurl(full); err == nil {
			return c, nil
		}
		return "", fmt.Errorf("sidecar api %s has no URL", apiID)
	}
	out, err := withCollection(userDir, collectionID, func(col map[string]any) (any, bool, error) {
		ref, ok := findItem(collectionItems(col), apiID)
		if !ok {
			return nil, false, fmt.Errorf("api %s not found in collection %s", apiID, collectionID)
		}
		return ref, false, nil
	})
	if err != nil {
		return "", err
	}
	if c, err := apiCurl(out.(map[string]any)); err == nil {
		return c, nil
	}
	return "", fmt.Errorf("full API bodies live in Box LMDB; use capture tools")
}
