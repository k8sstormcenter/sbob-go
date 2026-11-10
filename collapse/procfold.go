package collapse

import (
	"strings"

	"github.com/k8sstormcenter/sbob-go/match"
)

func FoldProcPath(path string) string {
	const prefix = "/proc/"
	if !strings.HasPrefix(path, prefix) {
		return path
	}
	rest := path[len(prefix):]
	n := 0
	for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
		n++
	}
	if n == 0 {
		return path
	}
	return prefix + match.DynamicIdentifier + rest[n:]
}
