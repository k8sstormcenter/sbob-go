package matchercorpus

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed corpus.json
var raw []byte

type PathCase struct {
	Name    string `json:"name,omitempty"`
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Want    bool   `json:"want"`
}

type ArgCase struct {
	Name    string   `json:"name,omitempty"`
	Profile []string `json:"profile"`
	Runtime []string `json:"runtime"`
	Want    bool     `json:"want"`
}

type Entry struct {
	Path string   `json:"path,omitempty"`
	Args []string `json:"args,omitempty"`
	Name string   `json:"name,omitempty"`
}

type SubtractCase struct {
	Name     string  `json:"name,omitempty"`
	Surface  string  `json:"surface"`
	Allow    []Entry `json:"allow"`
	Subtract []Entry `json:"subtract"`
	Runtime  Entry   `json:"runtime"`
	Want     bool    `json:"want"`
}

type ExecNameCase struct {
	Name     string `json:"name,omitempty"`
	Recorded string `json:"recorded"`
	Invoked  string `json:"invoked"`
	Want     bool   `json:"want"`
}

type SentinelCase struct {
	Name     string   `json:"name,omitempty"`
	Surface  string   `json:"surface"`
	Profile  []string `json:"profile"`
	Observed string   `json:"observed"`
	Want     bool     `json:"want"`
}

type Corpus struct {
	Paths     []PathCase     `json:"paths"`
	Args      []ArgCase      `json:"args"`
	Subtract  []SubtractCase `json:"subtract,omitempty"`
	ExecNames []ExecNameCase `json:"execNames,omitempty"`
	Sentinels []SentinelCase `json:"sentinels,omitempty"`
}

var (
	once   sync.Once
	corpus Corpus
	err    error
)

func load() {
	once.Do(func() {
		err = json.Unmarshal(raw, &corpus)
		if err != nil {
			err = fmt.Errorf("matchercorpus: corpus.json: %w", err)
		}
	})
}

func Load() (Corpus, error) {
	load()
	return corpus, err
}

func Paths() []PathCase {
	load()
	if err != nil {
		panic(err)
	}
	return corpus.Paths
}

func Args() []ArgCase {
	load()
	if err != nil {
		panic(err)
	}
	return corpus.Args
}

func Subtracts() []SubtractCase {
	load()
	if err != nil {
		panic(err)
	}
	return corpus.Subtract
}

func Version() string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func ExecNames() []ExecNameCase {
	load()
	if err != nil {
		panic(err)
	}
	return corpus.ExecNames
}

func Sentinels() []SentinelCase {
	load()
	if err != nil {
		panic(err)
	}
	return corpus.Sentinels
}
