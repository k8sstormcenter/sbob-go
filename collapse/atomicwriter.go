package collapse

import (
	"strings"

	"github.com/k8sstormcenter/sbob-go/match"
)

const atomicWriterStamp = "..dddd_dd_dd_dd_dd_dd."

func isAtomicWriterSegment(seg string) bool {
	if len(seg) <= len(atomicWriterStamp) {
		return false
	}
	for i := 0; i < len(atomicWriterStamp); i++ {
		want, got := atomicWriterStamp[i], seg[i]
		if want == 'd' {
			if got < '0' || got > '9' {
				return false
			}
			continue
		}
		if got != want {
			return false
		}
	}
	for i := len(atomicWriterStamp); i < len(seg); i++ {
		if seg[i] < '0' || seg[i] > '9' {
			return false
		}
	}
	return true
}

func FoldAtomicWriter(s string) string {
	if !strings.Contains(s, "/..") {
		return s
	}
	segs := strings.Split(s, "/")
	changed := false
	for i := 1; i < len(segs); i++ {
		if isAtomicWriterSegment(segs[i]) {
			segs[i] = match.DynamicIdentifier
			changed = true
		}
	}
	if !changed {
		return s
	}
	return strings.Join(segs, "/")
}
