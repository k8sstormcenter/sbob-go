package rules

type Rule struct {
	Name      string
	Section   string
	EngineID  string
	EngineIDs []string
	Events    []string
	Proposed  bool
}

type ProfileFunction struct {
	Name  string
	Arity int
}

func ByName(name string) (Rule, bool) {
	for _, r := range Catalogue {
		if r.Name == name {
			return r, true
		}
	}
	return Rule{}, false
}

func ByEngineID(id string) (Rule, bool) {
	if id == "" {
		return Rule{}, false
	}
	for _, r := range Catalogue {
		if r.EngineID == id {
			return r, true
		}
		for _, extra := range r.EngineIDs {
			if extra == id {
				return r, true
			}
		}
	}
	return Rule{}, false
}

func Function(name string) (ProfileFunction, bool) {
	for _, f := range Functions {
		if f.Name == name {
			return f, true
		}
	}
	return ProfileFunction{}, false
}
