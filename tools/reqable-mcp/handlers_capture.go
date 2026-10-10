package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"reqable-mcp/capture"
	"reqable-mcp/store"
	"sort"
	"strconv"
	"strings"
)

func init() {
	handlers["capture_list_recent"] = handleCaptureListRecent
	handlers["capture_get_request"] = handleCaptureGetRequest
	handlers["capture_search_url"] = handleCaptureSearchUrl
	handlers["capture_search_body"] = handleCaptureSearchBody
	handlers["capture_stats"] = handleCaptureStats
	handlers["capture_diff"] = handleCaptureDiff
	handlers["capture_find_dynamic_fields"] = handleCaptureFindDynamicFields
	handlers["capture_extract_auth"] = handleCaptureExtractAuth
	handlers["capture_decode_jwt"] = handleCaptureDecodeJwt
	handlers["capture_to_curl"] = handleCaptureToCurl
	handlers["capture_wait_for"] = handleCaptureWaitFor
	handlers["capture_list_apps"] = handleCaptureListApps
	handlers["capture_get_domains"] = handleCaptureGetDomains
	handlers["capture_analyze_api"] = handleCaptureAnalyzeApi
	handlers["capture_generate_code"] = handleCaptureGenerateCode
	handlers["capture_import_har"] = handleCaptureImportHar
	handlers["capture_ingest_status"] = handleCaptureIngestStatus
}

func handleCaptureListRecent(args map[string]any) (string, error) {
	limit := posArg(args, "limit", 20)
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	filtered := capture.Filter(recs, capture.FilterSpec{Host: strArg(args, "host", ""), Method: strArg(args, "method", ""), Code: intArg(args, "status", 0)})
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return marshalPretty(filtered), nil
}

func handleCaptureGetRequest(args map[string]any) (string, error) {
	r, err := reqRecord(args)
	if err != nil {
		return "", err
	}
	if !boolArg(args, "full", false) {
		return marshalPretty(r), nil
	}
	body := captureFileText(r, 4096)
	return marshalPretty(map[string]any{"record": r, "body": body}), nil
}

func handleCaptureSearchUrl(args map[string]any) (string, error) {
	pattern, err := reqPattern(args)
	if err != nil {
		return "", err
	}
	limit := posArg(args, "limit", 50)
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	var out []capture.Record
	if boolArg(args, "regex", false) {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", badArgs("invalid regex: " + err.Error())
		}
		for _, r := range recs {
			if re.MatchString(r.URL) {
				out = append(out, r)
				if len(out) >= limit {
					break
				}
			}
		}
	} else {
		lower := strings.ToLower(pattern)
		for _, r := range recs {
			if strings.Contains(strings.ToLower(r.URL), lower) {
				out = append(out, r)
				if len(out) >= limit {
					break
				}
			}
		}
	}
	return marshalPretty(out), nil
}

func handleCaptureSearchBody(args map[string]any) (string, error) {
	pattern, err := reqPattern(args)
	if err != nil {
		return "", err
	}
	limit := posArg(args, "limit", 20)
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	if len(recs) > 200 {
		recs = recs[len(recs)-200:]
	}
	var ids []int
	for _, r := range recs {
		if strings.Contains(captureFileText(r, 65536), pattern) {
			ids = append(ids, r.ID)
			if len(ids) >= limit {
				break
			}
		}
	}
	return marshalPretty(map[string]any{"ids": ids}), nil
}

func handleCaptureStats(args map[string]any) (string, error) {
	window := posArg(args, "window", 200)
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	if len(recs) > window {
		recs = recs[len(recs)-window:]
	}
	return marshalPretty(capture.Stats(recs)), nil
}

func handleCaptureDiff(args map[string]any) (string, error) {
	if !hasArg(args, "idA") || !hasArg(args, "idB") {
		return "", badArgs("idA and idB are required")
	}
	a, err := capture.GetByID(intArg(args, "idA", 0))
	if err != nil {
		return "", err
	}
	b, err := capture.GetByID(intArg(args, "idB", 0))
	if err != nil {
		return "", err
	}
	return capture.Diff(a, b), nil
}

func handleCaptureFindDynamicFields(args map[string]any) (string, error) {
	host, err := reqHost(args)
	if err != nil {
		return "", err
	}
	sample := posArg(args, "sample", 20)
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	filtered := capture.Filter(recs, capture.FilterSpec{Host: host})
	if len(filtered) > sample {
		filtered = filtered[len(filtered)-sample:]
	}
	return marshalPretty(capture.FindDynamicFields(filtered)), nil
}

func handleCaptureExtractAuth(args map[string]any) (string, error) {
	host, err := reqHost(args)
	if err != nil {
		return "", err
	}
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	return marshalPretty(capture.ExtractAuth(capture.Filter(recs, capture.FilterSpec{Host: host}))), nil
}

func handleCaptureDecodeJwt(args map[string]any) (string, error) {
	token := strArg(args, "token", "")
	if token == "" {
		return "", badArgs("token is required")
	}
	header, payload, err := capture.DecodeJWT(token)
	if err != nil {
		return "", badArgs(err.Error())
	}
	return marshalPretty(map[string]any{"header": header, "payload": payload}), nil
}

func handleCaptureToCurl(args map[string]any) (string, error) {
	r, err := reqRecord(args)
	if err != nil {
		return "", err
	}
	return capture.ToCurl(r, nil), nil
}

func handleCaptureWaitFor(args map[string]any) (string, error) {
	spec := capture.FilterSpec{
		Host: strArg(args, "host", ""), Method: strArg(args, "method", ""),
	}
	path := strArg(args, "path", "")
	r, found, err := capture.WaitFor(spec, path, posArg(args, "timeout", 30))
	if err != nil {
		return "", err
	}
	if !found {
		return "timeout: no new matching record", nil
	}
	return marshalPretty(r), nil
}

func handleCaptureListApps(args map[string]any) (string, error) {
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	var apps []string
	for _, r := range recs {
		h := r.Host
		if h == "" {
			continue
		}
		if !seen[h] {
			seen[h] = true
			apps = append(apps, h)
		}
	}
	sort.Strings(apps)
	return marshalPretty(apps), nil
}

func handleCaptureGetDomains(args map[string]any) (string, error) {
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	counts := map[string]int{}
	for _, r := range recs {
		h := r.Host
		if h == "" {
			h = "(unknown)"
		}
		counts[h]++
	}
	type kv struct {
		Domain string `json:"domain"`
		Count  int    `json:"count"`
	}
	var out []kv
	for d, c := range counts {
		out = append(out, kv{d, c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return marshalPretty(out), nil
}

func handleCaptureAnalyzeApi(args map[string]any) (string, error) {
	domain := strArg(args, "domain", "")
	if domain == "" {
		return "", badArgs("domain is required")
	}
	recs, err := capture.List()
	if err != nil {
		return "", err
	}
	shapes := map[string]int{}
	for _, r := range capture.Filter(recs, capture.FilterSpec{Host: domain}) {
		path := r.URL
		if u, err := url.Parse(r.URL); err == nil && u.Path != "" {
			path = u.Path
		}
		key := r.Method + " " + path + " -> " + strconv.Itoa(r.StatusCode)
		shapes[key]++
	}
	return marshalPretty(shapes), nil
}

func handleCaptureGenerateCode(args map[string]any) (string, error) {
	lang := strings.ToLower(strArg(args, "lang", "curl"))
	r, err := reqRecord(args)
	if err != nil {
		return "", err
	}
	method := r.Method
	if method == "" {
		method = "GET"
	}
	switch lang {
	case "python":
		return pythonSample(method, r.URL, r.ReqHeaders), nil
	case "go":
		return goSample(method, r.URL, r.ReqHeaders), nil
	default:
		return capture.ToCurl(r, nil), nil
	}
}

func pythonSample(method, url string, headers []string) string {
	var sb strings.Builder
	sb.WriteString("import requests\nresp = requests.request(\n")
	fmt.Fprintf(&sb, "    %q, %q,\n", method, url)
	sb.WriteString("    headers={\n")
	for _, h := range headers {
		k, v, _ := strings.Cut(h, ":")
		fmt.Fprintf(&sb, "        %q: %q,\n", strings.TrimSpace(k), strings.TrimSpace(v))
	}
	sb.WriteString("    },\n)\nprint(resp.status_code)\nprint(resp.text)\n")
	return sb.String()
}

func goSample(method, url string, headers []string) string {
	var sb strings.Builder
	sb.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc main() {\n")
	fmt.Fprintf(&sb, "\treq, _ := http.NewRequest(%q, %q, nil)\n", method, url)
	for _, h := range headers {
		k, v, _ := strings.Cut(h, ":")
		fmt.Fprintf(&sb, "\treq.Header.Set(%q, %q)\n", strings.TrimSpace(k), strings.TrimSpace(v))
	}
	sb.WriteString("\tresp, err := http.DefaultClient.Do(req)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tdefer resp.Body.Close()\n\tfmt.Println(resp.Status)\n}\n")
	return sb.String()
}

func handleCaptureImportHar(args map[string]any) (string, error) {
	path := strArg(args, "path", "")
	if path == "" {
		return "", badArgs("path is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var har struct {
		Log struct {
			Entries []harEntry `json:"entries"`
		} `json:"log"`
		Entries []harEntry `json:"entries"`
	}
	if err := json.Unmarshal(raw, &har); err != nil {
		return "", badArgs("invalid HAR: " + err.Error())
	}
	entries := har.Log.Entries
	if len(entries) == 0 {
		entries = har.Entries
	}
	count := 0
	for _, e := range entries {
		if e.Request.URL == "" {
			continue
		}
		name := e.Request.Method + " " + e.Request.URL
		if _, err := store.RestHTTPCreateFromURL(ud(args), name, e.Request.URL); err == nil {
			count++
		}
	}
	return fmt.Sprintf("staged %d HAR entries", count), nil
}

type harEntry struct {
	Request struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"request"`
}

func handleCaptureIngestStatus(args map[string]any) (string, error) {
	keys, files, size := boxStats()
	return marshalPretty(map[string]any{
		"boxKeys": keys, "captureFiles": files, "boxBytes": size,
	}), nil
}
