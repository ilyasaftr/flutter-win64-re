package capture

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Stats counts records by host, method, and status code.
func Stats(records []Record) map[string]map[string]int {
	out := map[string]map[string]int{"host": {}, "method": {}, "status": {}}
	for _, r := range records {
		host := r.Host
		if host == "" {
			host = "(unknown)"
		}
		out["host"][host]++
		m := r.Method
		if m == "" {
			m = "(unknown)"
		}
		out["method"][m]++
		out["status"][strconv.Itoa(r.StatusCode)]++
	}
	return out
}

func diffHeaders(a, b []string) []string {
	inB := map[string]bool{}
	for _, h := range b {
		inB[h] = true
	}
	inA := map[string]bool{}
	for _, h := range a {
		inA[h] = true
	}
	var lines []string
	for _, h := range a {
		if !inB[h] {
			lines = append(lines, "- "+h)
		}
	}
	for _, h := range b {
		if !inA[h] {
			lines = append(lines, "+ "+h)
		}
	}
	return lines
}

// Diff reports field-level differences of method/url/status/headers.
func Diff(a, b Record) string {
	var lines []string
	if a.Method != b.Method {
		lines = append(lines, fmt.Sprintf("method: %s -> %s", a.Method, b.Method))
	}
	if a.URL != b.URL {
		lines = append(lines, fmt.Sprintf("url: %s -> %s", a.URL, b.URL))
	}
	if a.StatusCode != b.StatusCode || a.StatusMsg != b.StatusMsg {
		lines = append(lines, fmt.Sprintf("status: %d %s -> %d %s", a.StatusCode, a.StatusMsg, b.StatusCode, b.StatusMsg))
	}
	lines = append(lines, diffHeaders(a.ReqHeaders, b.ReqHeaders)...)
	lines = append(lines, diffHeaders(a.ResHeaders, b.ResHeaders)...)
	if len(lines) == 0 {
		return "no differences"
	}
	return strings.Join(lines, "\n")
}

// FindDynamicFields returns header/query keys whose values vary across
// records, sampling up to 30 evenly.
func FindDynamicFields(records []Record) []string {
	sample := records
	if len(sample) > 30 {
		stride := len(sample) / 30
		picked := make([]Record, 0, 30)
		for i := 0; i < len(sample) && len(picked) < 30; i += stride {
			picked = append(picked, sample[i])
		}
		sample = picked
	}
	vals := map[string]map[string]bool{}
	display := map[string]string{}
	add := func(id, name, v string) {
		if vals[id] == nil {
			vals[id] = map[string]bool{}
		}
		vals[id][v] = true
		if _, ok := display[id]; !ok {
			display[id] = name
		}
	}
	for _, r := range sample {
		for _, h := range append(append([]string{}, r.ReqHeaders...), r.ResHeaders...) {
			name, value, _ := strings.Cut(h, ":")
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			add("header:"+strings.ToLower(name), "header:"+name, strings.TrimSpace(value))
		}
		if u, err := url.Parse(r.URL); err == nil {
			for k, vs := range u.Query() {
				add("query:"+k, "query:"+k, strings.Join(vs, ","))
			}
		}
	}
	var out []string
	for id, vs := range vals {
		if len(vs) > 1 {
			out = append(out, display[id])
		}
	}
	sort.Strings(out)
	return out
}

// ExtractAuth collects distinct Authorization/Cookie/X-*-Token values.
func ExtractAuth(records []Record) []string {
	var out []string
	seen := map[string]bool{}
	scan := func(headers []string) {
		for _, h := range headers {
			name, value, _ := strings.Cut(h, ":")
			n := strings.ToLower(strings.TrimSpace(name))
			v := strings.TrimSpace(value)
			if v == "" {
				continue
			}
			auth := n == "authorization" || n == "cookie" ||
				(strings.HasPrefix(n, "x-") && strings.HasSuffix(n, "token"))
			if auth && !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	for _, r := range records {
		scan(r.ReqHeaders)
		scan(r.ResHeaders)
	}
	return out
}

// DecodeJWT decodes a JWT header and payload without verification.
func DecodeJWT(token string) (map[string]any, map[string]any, error) {
	return DecodeJWTUnverified(token)
}
