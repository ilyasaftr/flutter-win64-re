package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findCollection(cfg *CollectionsFile, id string) int {
	for i, c := range cfg.Collections {
		if v, _ := c["id"].(string); v == id {
			return i
		}
	}
	return -1
}

func collectionItems(col map[string]any) []any {
	items, _ := col["items"].([]any)
	return items
}

func findItem(items []any, id string) (map[string]any, bool) {
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			if v, _ := m["id"].(string); v == id {
				return m, true
			}
		}
	}
	return nil, false
}

// CollectionGet returns one raw collection object.
func CollectionGet(userDir, id string) (map[string]any, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return nil, err
	}
	if i := findCollection(&cfg, id); i >= 0 {
		return cfg.Collections[i], nil
	}
	return nil, fmt.Errorf("collection %s not found", id)
}

// CollectionDelete removes a collection; returns found.
func CollectionDelete(userDir, id string) (bool, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return false, err
	}
	i := findCollection(&cfg, id)
	if i < 0 {
		return false, nil
	}
	cfg.Collections = append(cfg.Collections[:i], cfg.Collections[i+1:]...)
	return true, save(CollectionsPath(userDir), cfg)
}

var collectionPatchKeys = map[string]bool{
	"name": true, "query": true, "headers": true,
	"script": true, "authorization": true, "documentation": true,
}

// CollectionUpdate merges patch keys into the stored collection object.
func CollectionUpdate(userDir, id string, patch map[string]any) error {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return err
	}
	i := findCollection(&cfg, id)
	if i < 0 {
		return fmt.Errorf("collection %s not found", id)
	}
	for k, v := range patch {
		if collectionPatchKeys[k] {
			cfg.Collections[i][k] = v
		}
	}
	return save(CollectionsPath(userDir), cfg)
}

// CollectionFolderCreate appends a folder ref to the collection's items.
// The full folder tree lives in Box LMDB; this records the ref.
func CollectionFolderCreate(userDir, collectionID, name, parentID string) (string, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return "", err
	}
	i := findCollection(&cfg, collectionID)
	if i < 0 {
		return "", fmt.Errorf("collection %s not found", collectionID)
	}
	id := mustUUID()
	ref := map[string]any{"id": id, "type": "folder", "name": name}
	if parentID != "" {
		ref["parentId"] = parentID
	}
	cfg.Collections[i]["items"] = append(collectionItems(cfg.Collections[i]), ref)
	if err := save(CollectionsPath(userDir), cfg); err != nil {
		return "", err
	}
	return id, nil
}

// CollectionFolderGet finds a folder ref by id within a collection.
func CollectionFolderGet(userDir, collectionID, folderID string) (map[string]any, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return nil, err
	}
	i := findCollection(&cfg, collectionID)
	if i < 0 {
		return nil, fmt.Errorf("collection %s not found", collectionID)
	}
	ref, ok := findItem(collectionItems(cfg.Collections[i]), folderID)
	if !ok {
		return nil, fmt.Errorf("folder %s not found in collection %s", folderID, collectionID)
	}
	if t, _ := ref["type"].(string); t != "folder" {
		return nil, fmt.Errorf("item %s is not a folder", folderID)
	}
	return ref, nil
}

// CollectionAPIGet returns the stored item ref (id/type); bodies live in Box.
func CollectionAPIGet(userDir, collectionID, apiID string) (map[string]any, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return nil, err
	}
	i := findCollection(&cfg, collectionID)
	if i < 0 {
		return nil, fmt.Errorf("collection %s not found", collectionID)
	}
	ref, ok := findItem(collectionItems(cfg.Collections[i]), apiID)
	if !ok {
		return nil, fmt.Errorf("api %s not found in collection %s", apiID, collectionID)
	}
	return ref, nil
}

// CollectionAPIGetFull is an honest stub: bodies live in Box LMDB.
func CollectionAPIGetFull(userDir, collectionID, apiID string) (map[string]any, error) {
	return nil, fmt.Errorf("full API bodies live in Box LMDB; use capture tools")
}

type parsedCurl struct {
	method  string
	url     string
	headers [][2]string
	body    string
}

func tokenizeCurl(s string) []string {
	var toks []string
	var cur strings.Builder
	dirty := false
	inSingle, inDouble, esc := false, false, false
	flush := func() {
		if dirty {
			toks = append(toks, cur.String())
			cur.Reset()
			dirty = false
		}
	}
	for _, r := range s {
		switch {
		case esc:
			cur.WriteRune(r)
			dirty = true
			esc = false
		case r == '\\' && !inSingle:
			esc = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			dirty = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
			dirty = true
		case (r == ' ' || r == '\t' || r == '\n') && !inSingle && !inDouble:
			flush()
		default:
			cur.WriteRune(r)
			dirty = true
		}
	}
	flush()
	return toks
}

func parseCurlCommand(raw string) (parsedCurl, error) {
	toks := tokenizeCurl(raw)
	p := parsedCurl{method: "GET"}
	i := 0
	if len(toks) > 0 && (toks[0] == "curl" || strings.HasSuffix(toks[0], "/curl")) {
		i = 1
	}
	for ; i < len(toks); i++ {
		next := func() string {
			if i+1 < len(toks) {
				i++
				return toks[i]
			}
			return ""
		}
		switch toks[i] {
		case "-X", "--request":
			if v := next(); v != "" {
				p.method = strings.ToUpper(v)
			}
		case "-H", "--header":
			if h := next(); h != "" {
				if k, v, ok := strings.Cut(h, ":"); ok {
					p.headers = append(p.headers, [2]string{strings.TrimSpace(k), strings.TrimSpace(v)})
				}
			}
		case "--data-raw", "--data", "--data-binary", "--data-ascii", "-d", "--data-urlencode":
			p.body = next()
		case "--url":
			if v := next(); v != "" {
				p.url = v
			}
		default:
			if p.url == "" && (strings.HasPrefix(toks[i], "http://") || strings.HasPrefix(toks[i], "https://")) {
				p.url = toks[i]
			}
		}
	}
	if p.url == "" {
		return p, fmt.Errorf("no URL found in curl command")
	}
	return p, nil
}

func buildAPIObject(id, name, method, url string, headers [][2]string, body string) map[string]any {
	if name == "" {
		name = url
	}
	hdrs := []any{}
	for _, h := range headers {
		hdrs = append(hdrs, map[string]any{"key": h[0], "value": h[1]})
	}
	bodym := map[string]any{"mode": "none"}
	if body != "" {
		bodym = map[string]any{"mode": "text", "text": body}
	}
	return map[string]any{
		"id": id, "type": "api", "name": name, "method": method,
		"url": map[string]any{"base": url}, "headers": hdrs, "body": bodym,
		"script": map[string]any{"isEnabled": false},
		"authorization": map[string]any{"mode": "inherit"},
		"documentation": map[string]any{"content": ""},
	}
}

func apiSidecarPath(userDir, id string) string {
	return filepath.Join(userDir, "rest", "api_"+id+".json")
}

// CollectionAPICreateFromCurl parses a curl command, appends an api ref,
// and persists the full api JSON as a sidecar for manual import.
func CollectionAPICreateFromCurl(userDir, collectionID, curl, parentID string) (string, error) {
	p, err := parseCurlCommand(curl)
	if err != nil {
		return "", err
	}
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return "", err
	}
	i := findCollection(&cfg, collectionID)
	if i < 0 {
		return "", fmt.Errorf("collection %s not found", collectionID)
	}
	id := mustUUID()
	ref := map[string]any{"id": id, "type": "api"}
	if parentID != "" {
		ref["parentId"] = parentID
	}
	cfg.Collections[i]["items"] = append(collectionItems(cfg.Collections[i]), ref)
	if err := save(CollectionsPath(userDir), cfg); err != nil {
		return "", err
	}
	if err := save(apiSidecarPath(userDir, id), buildAPIObject(id, p.url, p.method, p.url, p.headers, p.body)); err != nil {
		return "", err
	}
	return id, nil
}

// CollectionAPIAdd appends an api ref; api["id"] is used or generated.
func CollectionAPIAdd(userDir, collectionID string, api map[string]any, parentID string) (string, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return "", err
	}
	i := findCollection(&cfg, collectionID)
	if i < 0 {
		return "", fmt.Errorf("collection %s not found", collectionID)
	}
	id, _ := api["id"].(string)
	if id == "" {
		id = mustUUID()
		api["id"] = id
	}
	ref := map[string]any{"id": id, "type": "api"}
	if parentID != "" {
		ref["parentId"] = parentID
	}
	cfg.Collections[i]["items"] = append(collectionItems(cfg.Collections[i]), ref)
	if err := save(CollectionsPath(userDir), cfg); err != nil {
		return "", err
	}
	return id, nil
}

// CollectionAPIUpdate overwrites the sidecar for a sidecar-managed API.
func CollectionAPIUpdate(userDir, collectionID string, api map[string]any) error {
	id, _ := api["id"].(string)
	if id == "" {
		return fmt.Errorf("api.id is required")
	}
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return err
	}
	if findCollection(&cfg, collectionID) < 0 {
		return fmt.Errorf("collection %s not found", collectionID)
	}
	if _, err := os.Stat(apiSidecarPath(userDir, id)); err != nil {
		return fmt.Errorf("only sidecar-managed APIs can be updated")
	}
	return save(apiSidecarPath(userDir, id), api)
}

// CollectionAPIDelete removes the ref and its sidecar if present.
func CollectionAPIDelete(userDir, collectionID, apiID string) (bool, error) {
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return false, err
	}
	i := findCollection(&cfg, collectionID)
	if i < 0 {
		return false, fmt.Errorf("collection %s not found", collectionID)
	}
	items := collectionItems(cfg.Collections[i])
	kept := make([]any, 0, len(items))
	found := false
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			if v, _ := m["id"].(string); v == apiID {
				found = true
				continue
			}
		}
		kept = append(kept, it)
	}
	if !found {
		return false, nil
	}
	cfg.Collections[i]["items"] = kept
	if err := save(CollectionsPath(userDir), cfg); err != nil {
		return false, err
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

// CollectionAPIGenerateCurl builds curl from the sidecar, falling back
// to the stored ref when it carries URL details.
func CollectionAPIGenerateCurl(userDir, collectionID, apiID string) (string, error) {
	var full map[string]any
	if err := load(apiSidecarPath(userDir, apiID), &full); err == nil {
		if u := apiURL(full); u != "" {
			return buildCurl(apiMethod(full), u, apiHeaders(full), apiBodyText(full)), nil
		}
		return "", fmt.Errorf("sidecar api %s has no URL", apiID)
	}
	var cfg CollectionsFile
	if err := load(CollectionsPath(userDir), &cfg); err != nil {
		return "", err
	}
	i := findCollection(&cfg, collectionID)
	if i < 0 {
		return "", fmt.Errorf("collection %s not found", collectionID)
	}
	ref, ok := findItem(collectionItems(cfg.Collections[i]), apiID)
	if !ok {
		return "", fmt.Errorf("api %s not found in collection %s", apiID, collectionID)
	}
	if u := apiURL(ref); u != "" {
		return buildCurl(apiMethod(ref), u, apiHeaders(ref), apiBodyText(ref)), nil
	}
	return "", fmt.Errorf("full API bodies live in Box LMDB; use capture tools")
}
