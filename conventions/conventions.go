package conventions

import "sync"

type Entry struct {
	Key     string
	Legacy  string
	Values  []string
	Meaning string
}

type Kind struct {
	Name           string
	Group          string
	Version        string
	Resource       string
	LegacyName     string
	LegacyGroup    string
	LegacyVersion  string
	LegacyResource string
	SubjectValues  []string
}

type Retired struct {
	Legacy string
	Reason string
}

var (
	indexOnce       sync.Once
	labelIndex      map[string]int
	annotationIndex map[string]int
)

func indexes() (map[string]int, map[string]int) {
	indexOnce.Do(func() {
		labelIndex = index(Labels)
		annotationIndex = index(Annotations)
	})
	return labelIndex, annotationIndex
}

func index(entries []Entry) map[string]int {
	m := make(map[string]int, 2*len(entries))
	for i, e := range entries {
		m[e.Key] = i
		if e.Legacy != "" {
			m[e.Legacy] = i
		}
	}
	return m
}

func Label(key string) (Entry, bool) {
	labels, _ := indexes()
	i, ok := labels[key]
	if !ok {
		return Entry{}, false
	}
	return Labels[i], true
}

func Annotation(key string) (Entry, bool) {
	_, annotations := indexes()
	i, ok := annotations[key]
	if !ok {
		return Entry{}, false
	}
	return Annotations[i], true
}

func lookup(key string) (Entry, bool) {
	if e, ok := Label(key); ok {
		return e, true
	}
	return Annotation(key)
}

func Current(key string) string {
	if e, ok := lookup(key); ok {
		return e.Key
	}
	return key
}

func Legacy(key string) string {
	if e, ok := lookup(key); ok && e.Legacy != "" {
		return e.Legacy
	}
	return key
}

func (e Entry) Allows(value string) bool {
	if len(e.Values) == 0 {
		return true
	}
	for _, v := range e.Values {
		if v == value {
			return true
		}
	}
	return false
}

func KindByName(name string) (Kind, bool) {
	for _, k := range Kinds {
		if k.Name == name || (k.LegacyName != "" && k.LegacyName == name) {
			return k, true
		}
	}
	return Kind{}, false
}

func IsRetired(key string) bool {
	for _, r := range RetiredKeys {
		if r.Legacy == key {
			return true
		}
	}
	return false
}
