package store

// RewriteEntry mirrors the JSON shape the Reqable app persists in
// config/capture_rewrite.
type RewriteEntry struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Method    string        `json:"method"`
	URL       string        `json:"url"`
	Wildcard  bool          `json:"wildcard"`
	Action    RewriteAction `json:"action"`
	IsEnabled bool          `json:"isEnabled"`
}

type RewriteAction struct {
	Type                 int             `json:"type"`
	Code                 *int            `json:"code,omitempty"`
	StatusLineEnabled    *bool           `json:"statusLineEnabled,omitempty"`
	HeadersEnabled       *bool           `json:"headersEnabled,omitempty"`
	BodyEnabled          *bool           `json:"bodyEnabled,omitempty"`
	Headers              []any           `json:"headers,omitempty"`
	Body                 *RewriteBody    `json:"body,omitempty"`
	Modifies             []RewriteModify `json:"modifies,omitempty"`
	ModifyQueryTemplate  *string         `json:"modifyQueryTemplate"`
	ModifyHeaderTemplate *string         `json:"modifyHeaderTemplate"`
	ModifyBodyTemplate   *string         `json:"modifyBodyTemplate"`
}

type RewriteBody struct {
	Type    int    `json:"type"`
	Payload string `json:"payload"`
}

type RewriteModify struct {
	ID        string       `json:"id"`
	Type      int          `json:"type"`
	Modify    ModifyDetail `json:"modify"`
	IsEnabled bool         `json:"isEnabled"`
}

type ModifyDetail struct {
	Pattern       string `json:"pattern"`
	Replacement   string `json:"replacement"`
	Regex         bool   `json:"regex"`
	CaseSensitive bool   `json:"caseSensitive"`
}

type BreakpointEntry struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Method            string `json:"method"`
	URL               string `json:"url"`
	Wildcard          bool   `json:"wildcard"`
	IsRequestEnabled  bool   `json:"isRequestEnabled"`
	IsResponseEnabled bool   `json:"isResponseEnabled"`
	IsEnabled         bool   `json:"isEnabled"`
}

type CollectionsFile struct {
	Collections []map[string]any `json:"collections"`
}

func rewriteID(e RewriteEntry) string       { return e.ID }
func breakpointID(e BreakpointEntry) string { return e.ID }

func setRewriteOn(e *RewriteEntry, b bool)       { e.IsEnabled = b }
func setBreakpointOn(e *BreakpointEntry, b bool) { e.IsEnabled = b }

func loadRewrites(userDir string) (FolderFile[RewriteEntry], error) {
	return loadFolder[RewriteEntry](RewritePath(userDir), "rewrites")
}

func saveRewrites(userDir string, f FolderFile[RewriteEntry]) error {
	return saveFolder(RewritePath(userDir), "rewrites", f)
}

func loadBreakpoints(userDir string) (FolderFile[BreakpointEntry], error) {
	return loadFolder[BreakpointEntry](BreakpointPath(userDir), "breakpoints")
}

func saveBreakpoints(userDir string, f FolderFile[BreakpointEntry]) error {
	return saveFolder(BreakpointPath(userDir), "breakpoints", f)
}

// ListRewrites returns (entries, isEnabled).
func ListRewrites(userDir string) ([]RewriteEntry, bool, error) {
	f, err := loadRewrites(userDir)
	if err != nil {
		return nil, false, err
	}
	return f.Items, f.IsEnabled, nil
}

// AddRewrite appends a Replace-Response (type 2) rule and returns its id.
func AddRewrite(userDir, name, method, url, payload string, wildcard bool) (string, error) {
	f, err := loadRewrites(userDir)
	if err != nil {
		return "", err
	}
	id := mustUUID()
	f.Items = append(f.Items, RewriteEntry{
		ID: id, Name: name, Method: method, URL: url, Wildcard: wildcard,
		Action: RewriteAction{
			Type: 2, Code: intPtr(200),
			StatusLineEnabled: boolPtr(false), HeadersEnabled: boolPtr(false),
			BodyEnabled: boolPtr(true), Headers: []any{},
			Body: &RewriteBody{Type: 1, Payload: payload},
		},
		IsEnabled: true,
	})
	return id, saveRewrites(userDir, f)
}

// AddRewriteModify appends a Modify (type 4) rule and returns its id.
func AddRewriteModify(userDir, name, url, pattern, replacement string) (string, error) {
	f, err := loadRewrites(userDir)
	if err != nil {
		return "", err
	}
	id := mustUUID()
	f.Items = append(f.Items, RewriteEntry{
		ID: id, Name: name, URL: url, Wildcard: true,
		Action: RewriteAction{
			Type: 4,
			Modifies: []RewriteModify{{
				ID: mustUUID(), Type: 0,
				Modify:    ModifyDetail{Pattern: pattern, Replacement: replacement, Regex: true},
				IsEnabled: true,
			}},
		},
		IsEnabled: true,
	})
	return id, saveRewrites(userDir, f)
}

// DeleteRewrite removes a rule by id; returns (before, after).
func DeleteRewrite(userDir, id string) (int, int, error) {
	f, err := loadRewrites(userDir)
	if err != nil {
		return 0, 0, err
	}
	before := len(f.Items)
	f.Items, _ = Delete(f.Items, id, rewriteID)
	return before, len(f.Items), saveRewrites(userDir, f)
}

// ListBreakpoints returns (entries, isEnabled).
func ListBreakpoints(userDir string) ([]BreakpointEntry, bool, error) {
	f, err := loadBreakpoints(userDir)
	if err != nil {
		return nil, false, err
	}
	return f.Items, f.IsEnabled, nil
}

// AddBreakpoint appends a breakpoint rule and returns its id.
func AddBreakpoint(userDir, name, method, url string, wildcard bool) (string, error) {
	f, err := loadBreakpoints(userDir)
	if err != nil {
		return "", err
	}
	id := mustUUID()
	f.Items = append(f.Items, BreakpointEntry{
		ID: id, Name: name, Method: method, URL: url, Wildcard: wildcard,
		IsRequestEnabled: true, IsResponseEnabled: true, IsEnabled: true,
	})
	return id, saveBreakpoints(userDir, f)
}

// DeleteBreakpoint removes a breakpoint by id; returns (before, after).
func DeleteBreakpoint(userDir, id string) (int, int, error) {
	f, err := loadBreakpoints(userDir)
	if err != nil {
		return 0, 0, err
	}
	before := len(f.Items)
	f.Items, _ = Delete(f.Items, id, breakpointID)
	return before, len(f.Items), saveBreakpoints(userDir, f)
}

func loadCollections(userDir string) (CollectionsFile, error) {
	var cfg CollectionsFile
	err := loadFile(CollectionsPath(userDir), &cfg)
	return cfg, err
}

func saveCollections(userDir string, cfg CollectionsFile) error {
	return saveFile(CollectionsPath(userDir), cfg)
}

// AddCollection appends an empty collection and returns its id.
func AddCollection(userDir, name string) (string, error) {
	cfg, err := loadCollections(userDir)
	if err != nil {
		return "", err
	}
	id := mustUUID()
	cfg.Collections = append(cfg.Collections, map[string]any{
		"type": "collection", "id": id, "name": name,
		"items": []any{}, "revision": 1,
	})
	return id, saveCollections(userDir, cfg)
}

// ListCollections returns raw collection objects.
func ListCollections(userDir string) ([]map[string]any, error) {
	cfg, err := loadCollections(userDir)
	if err != nil {
		return nil, err
	}
	return cfg.Collections, nil
}

// WriteScriptSidecar persists a script definition as a JSON sidecar next
// to the config dir for manual import. Scripts live in the object store,
// so there is no plain config file to append to.
func WriteScriptSidecar(userDir, name, url, code string, wildcard bool) (string, error) {
	id := mustUUID()
	path := RewritePath(userDir) + "_grok_" + id + ".json"
	doc := map[string]any{
		"id": id, "name": name, "url": url, "wildcard": wildcard,
		"code": code, "isEnabled": true,
	}
	return path, saveFile(path, doc)
}
