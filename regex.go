package dynamicpathdetector

import (
	"regexp"
	"strings"
)

func PathRegexp(pattern string) string {
	if pattern == "" {
		return "^$"
	}
	if pattern == WildcardIdentifier {
		return "^.+$"
	}
	pattern = collapseConsecutiveStars(pattern)
	segs := strings.Split(pattern, "/")
	if len(segs) > 1 && segs[len(segs)-1] == "" {
		segs = segs[:len(segs)-1]
	}
	var b strings.Builder
	b.WriteString("^")
	afterMidStar := false
	for i, seg := range segs {
		last := i == len(segs)-1
		if seg == WildcardIdentifier {
			if last {
				if i > 0 {
					b.WriteString("/")
				}
				b.WriteString(".+")
				continue
			}
			if i > 0 && !afterMidStar {
				b.WriteString("/")
			}
			b.WriteString("(?:[^/]+/)*")
			afterMidStar = true
			continue
		}
		if i > 0 && !afterMidStar {
			b.WriteString("/")
		}
		afterMidStar = false
		if seg == DynamicIdentifier {
			b.WriteString("[^/]+")
			continue
		}
		b.WriteString(regexp.QuoteMeta(seg))
	}
	b.WriteString("$")
	return b.String()
}

func ArgTokenRegexp(token, sep string) string {
	notSep := "[^" + regexp.QuoteMeta(sep) + "]"
	if sep == "" {
		notSep = "."
	}
	notSepOrSlash := "[^/" + regexp.QuoteMeta(sep) + "]"
	if sep == "" {
		notSepOrSlash = "[^/]"
	}
	if token == DynamicIdentifier {
		return notSep + "*"
	}
	if !strings.Contains(token, DynamicIdentifier) {
		return regexp.QuoteMeta(token)
	}
	segs := strings.Split(token, "/")
	parts := make([]string, 0, len(segs))
	for _, seg := range segs {
		switch seg {
		case ExecArgsWildcard:
			parts = append(parts, notSepOrSlash+"*(?:/"+notSepOrSlash+"*)*")
		case DynamicIdentifier:
			parts = append(parts, notSepOrSlash+"*")
		default:
			parts = append(parts, regexp.QuoteMeta(seg))
		}
	}
	return strings.Join(parts, "/")
}

func ArgvRegexp(args []string, sep string) string {
	if len(args) == 0 {
		return "^$"
	}
	s := regexp.QuoteMeta(sep)
	one := "[^" + s + "]*"
	if sep == "" {
		one = ".*"
	}
	var b strings.Builder
	b.WriteString("^")
	skipSep := false
	for i, tok := range args {
		if tok == ExecArgsWildcard {
			if i == 0 {
				if len(args) == 1 {
					b.WriteString("(?:" + one + "(?:" + s + one + ")*)?")
				} else {
					b.WriteString("(?:" + one + s + ")*")
					skipSep = true
				}
			} else {
				b.WriteString("(?:" + s + one + ")*")
			}
			continue
		}
		if i > 0 && !skipSep {
			b.WriteString(s)
		}
		skipSep = false
		b.WriteString(ArgTokenRegexp(tok, sep))
	}
	b.WriteString("$")
	return b.String()
}
