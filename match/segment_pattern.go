package match

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

func IsSegmentPattern(seg string) bool {
	return seg != DynamicIdentifier && strings.Contains(seg, DynamicIdentifier) && !strings.Contains(seg, ExecArgsWildcard)
}

func afterOneCodePoint(s string, i int) int {
	if i >= len(s) {
		return len(s) + 1
	}
	_, size := utf8.DecodeRuneInString(s[i:])
	return i + size
}

func segmentMatches(pattern, seg string) bool {
	if pattern == seg {
		return true
	}
	if !IsSegmentPattern(pattern) {
		return false
	}
	cut := strings.Index(pattern, DynamicIdentifier)
	first := pattern[:cut]
	rest := pattern[cut+len(DynamicIdentifier):]
	tail := strings.LastIndex(rest, DynamicIdentifier)
	last := rest
	middle := ""
	if tail >= 0 {
		middle, last = rest[:tail], rest[tail+len(DynamicIdentifier):]
	}
	if !strings.HasPrefix(seg, first) || !strings.HasSuffix(seg, last) {
		return false
	}
	pos, end := len(first), len(seg)-len(last)
	if tail >= 0 {
		for {
			mid := middle
			next := strings.Index(middle, DynamicIdentifier)
			if next >= 0 {
				mid = middle[:next]
			}
			pos = afterOneCodePoint(seg, pos)
			if pos > end {
				return false
			}
			i := strings.Index(seg[pos:end], mid)
			if i < 0 {
				return false
			}
			pos += i + len(mid)
			if next < 0 {
				break
			}
			middle = middle[next+len(DynamicIdentifier):]
		}
	}
	return afterOneCodePoint(seg, pos) <= end
}

func segmentPatternRegexp(seg, oneOrMore string, quote func(string) string) string {
	parts := strings.Split(seg, DynamicIdentifier)
	for i, p := range parts {
		parts[i] = quote(p)
	}
	return strings.Join(parts, oneOrMore)
}

func quotePathPart(s string) string {
	return regexp.QuoteMeta(s)
}
