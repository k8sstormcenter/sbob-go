package dynamicpathdetector

import "slices"

type stringSet map[string]struct{}

func newStringSet(items ...string) stringSet {
	s := make(stringSet, len(items))
	s.addAll(items...)
	return s
}

func (s stringSet) add(item string) bool {
	if _, ok := s[item]; ok {
		return false
	}
	s[item] = struct{}{}
	return true
}

func (s stringSet) addAll(items ...string) {
	for _, item := range items {
		s[item] = struct{}{}
	}
}

func (s stringSet) has(item string) bool {
	_, ok := s[item]
	return ok
}

func (s stringSet) sorted() []string {
	out := make([]string, 0, len(s))
	for item := range s {
		out = append(out, item)
	}
	slices.Sort(out)
	return out
}
