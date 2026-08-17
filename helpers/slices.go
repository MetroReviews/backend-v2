// Package helpers holds small, generic utilities shared by multiple
// packages (routes, the bot, silverpelt, ...). Anything here should be
// free of business logic — just string/slice/HTTP-response plumbing that
// would otherwise get copy-pasted around.
package helpers

// Contains reports whether v is present in s.
func Contains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// DedupeExcluding returns in with duplicates removed and every occurrence
// of exclude dropped, preserving the first-seen order.
func DedupeExcluding[T comparable](in []T, exclude T) []T {
	seen := map[T]struct{}{}
	out := []T{}
	for _, v := range in {
		if v == exclude {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
