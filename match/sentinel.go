package match

import "path"

const AnySentinel = "*"

func MatchAny(entries []string, observed string) bool {
	if observed == "" {
		return false
	}
	for _, e := range entries {
		if e == AnySentinel || e == observed {
			return true
		}
	}
	return false
}

func SameExecName(recorded, invoked string) bool {
	if recorded == "" || invoked == "" {
		return false
	}
	return path.Base(recorded) == path.Base(invoked)
}
