package dynamicpathdetector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/kubescape/storage/pkg/config"
)

const contractDocPath = "../../../../docs/sbob-matcher-contract.md"

const corpusJSONPath = "matchercorpus/corpus.json"

var contractCitation = regexp.MustCompile(`\{\{(paths|args|subtract)\|([^}]+)\}\}`)

type contractCorpus struct {
	Paths    []struct{ Name string } `json:"paths"`
	Args     []struct{ Name string } `json:"args"`
	Subtract []struct{ Name string } `json:"subtract"`
}

func loadContractCorpus(t *testing.T) map[string]map[string]bool {
	t.Helper()
	b, err := os.ReadFile(corpusJSONPath)
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var c contractCorpus
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatalf("parse corpus: %v", err)
	}
	named := map[string]map[string]bool{
		"paths":    {},
		"args":     {},
		"subtract": {},
	}
	for _, x := range c.Paths {
		if x.Name != "" {
			named["paths"][x.Name] = true
		}
	}
	for _, x := range c.Args {
		if x.Name != "" {
			named["args"][x.Name] = true
		}
	}
	for _, x := range c.Subtract {
		if x.Name != "" {
			named["subtract"][x.Name] = true
		}
	}
	return named
}

func loadContractDoc(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(contractDocPath))
	if err != nil {
		t.Fatalf("read contract doc: %v", err)
	}
	return string(b)
}

func TestContract_EveryCitationResolvesToACase(t *testing.T) {
	named := loadContractCorpus(t)
	doc := loadContractDoc(t)
	matches := contractCitation.FindAllStringSubmatch(doc, -1)
	if len(matches) == 0 {
		t.Fatal("the contract document cites no conformance case; a rule nobody can check is not a contract")
	}
	var unresolved []string
	for _, m := range matches {
		kind, name := m[1], m[2]
		if !named[kind][name] {
			unresolved = append(unresolved, kind+": "+name)
		}
	}
	sort.Strings(unresolved)
	t.Logf("citations %d, named cases %d/%d/%d (paths/args/subtract)",
		len(matches), len(named["paths"]), len(named["args"]), len(named["subtract"]))
	if len(unresolved) > 0 {
		t.Fatalf("the contract states rules no conformance case decides:\n  %s", strings.Join(unresolved, "\n  "))
	}
}

func TestContract_EveryNamedCaseIsCitedByARule(t *testing.T) {
	named := loadContractCorpus(t)
	doc := loadContractDoc(t)
	cited := map[string]map[string]bool{"paths": {}, "args": {}, "subtract": {}}
	for _, m := range contractCitation.FindAllStringSubmatch(doc, -1) {
		cited[m[1]][m[2]] = true
	}
	var orphans []string
	for kind, cases := range named {
		for name := range cases {
			if !cited[kind][name] {
				orphans = append(orphans, kind+": "+name)
			}
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Fatalf("named conformance cases no rule in the contract claims, so the document no longer describes what is enforced:\n  %s", strings.Join(orphans, "\n  "))
	}
}

func TestContract_TheLimitsTableMatchesTheOwnedConstants(t *testing.T) {
	doc := loadContractDoc(t)
	cfg, err := config.LoadConfig("../../../../configuration")
	if err != nil {
		t.Fatalf("load storage config defaults: %v", err)
	}
	row := regexp.MustCompile(`\|\s*profile size before a profile is marked too large\s*\|\s*storage\s*\|\s*` + "`maxContainerProfileSize`" + `\s*\|\s*(\d+)\s*\|`)
	m := row.FindStringSubmatch(doc)
	if m == nil {
		t.Fatal("the limits table has no row for the storage profile size cap")
	}
	want := itoaContract(cfg.MaxContainerProfileSize)
	if m[1] != want {
		t.Fatalf("the contract states a profile size cap of %s, storage's own default is %s", m[1], want)
	}
}

func itoaContract(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}
