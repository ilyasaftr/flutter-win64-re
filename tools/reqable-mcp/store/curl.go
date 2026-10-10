package store

import "strings"

// ParseCurl extracts method, URL, headers, and body from a curl command.
// An empty url means no URL was found.
func ParseCurl(curl string) (method, url string, headers [][2]string, body string) {
	toks := tokenizeCurl(curl)
	method = "GET"
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
				method = strings.ToUpper(v)
			}
		case "-H", "--header":
			if h := next(); h != "" {
				if k, v, ok := strings.Cut(h, ":"); ok {
					headers = append(headers, [2]string{strings.TrimSpace(k), strings.TrimSpace(v)})
				}
			}
		case "--data-raw", "--data", "--data-binary", "--data-ascii", "-d", "--data-urlencode":
			body = next()
		case "--url":
			if v := next(); v != "" {
				url = v
			}
		default:
			if url == "" && (strings.HasPrefix(toks[i], "http://") || strings.HasPrefix(toks[i], "https://")) {
				url = toks[i]
			}
		}
	}
	return method, url, headers, body
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
