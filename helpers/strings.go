package helpers

import "strings"

func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

func HTTPSify(v string) (string, bool) {
	if strings.HasPrefix(v, "https://") {
		return v, true
	}
	v = strings.Replace(v, "http://", "https://", 1)
	if strings.HasPrefix(v, "https://") {
		return v, true
	}
	return "", false
}

func ValidateImageURLs(urls []string, max int) (out []string, ok bool) {
	if len(urls) > max {
		return nil, false
	}
	out = make([]string, 0, len(urls))
	for _, u := range urls {
		httpsURL, valid := HTTPSify(u)
		if !valid {
			return nil, false
		}
		out = append(out, httpsURL)
	}
	return out, true
}
