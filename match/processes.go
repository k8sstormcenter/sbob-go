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
	return strings.Join(normalProcesses(processes), "\x00")
}

func UnionProcesses(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	return normalProcesses(slices.Concat(a, b))
}

func normalProcesses(processes []string) []string {
	out := slices.Clone(processes)
	slices.Sort(out)
	return slices.Compact(out)
}
