package main

import (
	"github.com/k8sstormcenter/sbob-go/collapse"
	"github.com/k8sstormcenter/sbob-go/conventions"
	"github.com/k8sstormcenter/sbob-go/match"
	"github.com/k8sstormcenter/sbob-go/rules"
)

func main() {
	ok := match.CompareDynamic("/etc/*", "/etc/passwd") && match.MatchIP([]string{"10.0.0.0/8"}, "10.1.2.3")
	analyzer := collapse.NewPathAnalyzerWithConfigs(collapse.OpenDynamicThreshold, nil)
	_, _ = analyzer.AnalyzePath("/tmp/x", "opens")
	if !ok {
		panic("matcher self-check failed")
	}
	if _, found := conventions.Label(conventions.LegacyLabelUserDefinedProfile); !found {
		panic("conventions self-check failed")
	}
	if class, _ := rules.Classify("!cp.was_path_opened(event.containerId, event.path)"); class != rules.Generated {
		panic("rules self-check failed")
	}
}
