package match

import (
	"errors"
	"slices"
	"strings"
)

var ErrEmptyProcess = errors.New("empty process")

func CheckProcess(process string) error {
	if process == "" {
		return ErrEmptyProcess
	}
	return nil
}

func ProcessSetKey(processes []string) string {
	return strings.Join(NormalProcesses(processes), "\x00")
}

func UnionProcesses(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	return NormalProcesses(slices.Concat(a, b))
}

func NormalProcesses(processes []string) []string {
	out := slices.Clone(processes)
	slices.Sort(out)
	return slices.Compact(out)
}
