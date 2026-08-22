package helpers

import (
	"regexp"
	"strings"
)

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a name into a URL-safe slug: lowercased, with runs of
// non-alphanumeric characters collapsed to single hyphens and the ends
// trimmed. Returns "" only when the input has no alphanumeric characters at
// all, so callers should fall back (e.g. to a default or an ID) in that case.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
