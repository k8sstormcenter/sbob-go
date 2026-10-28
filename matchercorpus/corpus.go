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

type Corpus struct {
	Paths []PathCase `json:"paths"`
	Args  []ArgCase  `json:"args"`
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

func Version() string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
