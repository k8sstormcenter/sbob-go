package dynamicpathdetector

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const runcLikePath = "/var/lib/rancher/k3s/data/4a9584f4b72a328dc77a958636e91442257c1890aaa66f0508b3604bf676dbdd/bin/runc"

func hexID(r *rand.Rand, n int) string {
	const digits = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = digits[r.Intn(len(digits))]
	}
	return string(b)
}

func runcLikeExecs(n int, seed int64) []Exec {
	r := rand.New(rand.NewSource(seed))
	containers := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		containers = append(containers, hexID(r, 64))
	}
	out := make([]Exec, 0, n)
	for i := 0; i < n; i++ {
		cid := containers[r.Intn(len(containers))]
		execID := hexID(r, 64)
		last := execID
		if i%2 == 1 {
			cid = DynamicIdentifier
			last = DynamicIdentifier
		}
		task := "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/" + cid
		out = append(out, Exec{Path: runcLikePath, Args: []string{
			runcLikePath,
			"--root", "/run/containerd/runc/k8s.io",
			"--log", task + "/log.json",
			"--log-format", "json",
			"--systemd-cgroup",
			"exec",
			"--process", fmt.Sprintf("/tmp/runc-process%d", r.Intn(1<<31)),
			"--detach",
			"--pid-file", task + "/" + execID + ".pid",
			"--console-socket", fmt.Sprintf("/tmp/pty%d/pty.sock", r.Intn(1<<31)),
			last,
		}})
	}
	return out
}

func TestAnalyzeExecsRuncLikeCollapsesNearLinearly(t *testing.T) {
	execs := runcLikeExecs(10000, 1)
	analyzer := NewExecAnalyzer(ExecDynamicThreshold, nil)

	start := time.Now()
	first := AnalyzeExecs(execs, analyzer)
	firstPass := time.Since(start)

	start = time.Now()
	second := AnalyzeExecs(first, analyzer)
	secondPass := time.Since(start)
	t.Logf("first pass %d -> %d in %s, second pass %d -> %d in %s", len(execs), len(first), firstPass, len(first), len(second), secondPass)

	require.Equal(t, first, second)
	require.LessOrEqual(t, len(first), ExecDynamicThreshold)
	for _, e := range execs {
		covered := false
		for _, p := range first {
			if CompareExecArgs(p.Args, e.Args) {
				covered = true
				break
			}
		}
		require.True(t, covered, "input %v not matched by the collapsed profile", e.Args)
	}
	require.Less(t, firstPass, 5*time.Second)
	require.Less(t, secondPass, 2*time.Second)
}

func BenchmarkAnalyzeExecsRuncLike(b *testing.B) {
	execs := runcLikeExecs(10000, 1)
	analyzer := NewExecAnalyzer(ExecDynamicThreshold, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AnalyzeExecs(execs, analyzer)
	}
}

func BenchmarkAnalyzeExecsRuncLikeSecondPass(b *testing.B) {
	execs := AnalyzeExecs(runcLikeExecs(10000, 1), NewExecAnalyzer(ExecDynamicThreshold, nil))
	analyzer := NewExecAnalyzer(ExecDynamicThreshold, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AnalyzeExecs(execs, analyzer)
	}
}
