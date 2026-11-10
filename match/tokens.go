package match

import "strings"

const (
	DynamicIdentifier  string = "⋯"  // U+22EF: ⋯ (one segment / one arg)
	WildcardIdentifier string = "*"  // zero-or-more path segments (opens only)
	ExecArgsWildcard   string = "⋯⋯" // zero-or-more whole exec args
)

func HasWildcard(s string) bool {
	return strings.Contains(s, DynamicIdentifier) || strings.Contains(s, WildcardIdentifier)
}
