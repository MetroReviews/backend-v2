package helpers

func Contains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

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
