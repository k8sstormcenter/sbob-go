package match

import "strings"

type runtimeKind int

const (
	notRuntime runtimeKind = iota
	rtShell
	rtPython
	rtPerl
	rtRuby
	rtNode
	rtPHP
	rtLua
	rtAwk
	rtJava
)

var shellNames = map[string]bool{"sh": true, "bash": true, "dash": true, "ash": true, "zsh": true, "ksh": true, "mksh": true}

var awkNames = map[string]bool{"awk": true, "gawk": true, "mawk": true, "nawk": true}

var wrapperNames = map[string]bool{"env": true, "nohup": true, "nice": true, "timeout": true, "stdbuf": true, "xargs": true}

func baseName(p string) string {
	p = strings.TrimRight(p, "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func versioned(name, base string) bool {
	if !strings.HasPrefix(name, base) {
		return false
	}
	for _, r := range name[len(base):] {
		if !(r >= '0' && r <= '9' || r == '.') {
			return false
		}
	}
	return true
}

func kindOfName(name string) runtimeKind {
	switch {
	case shellNames[name]:
		return rtShell
	case awkNames[name]:
		return rtAwk
	case name == "node" || name == "nodejs":
		return rtNode
	case name == "java":
		return rtJava
	case versioned(name, "python"):
		return rtPython
	case versioned(name, "perl"):
		return rtPerl
	case versioned(name, "ruby"):
		return rtRuby
	case versioned(name, "php"):
		return rtPHP
	case versioned(name, "lua"):
		return rtLua
	}
	return notRuntime
}

func usableName(name string) bool {
	return name != "" && !strings.Contains(name, DynamicIdentifier) && !strings.Contains(name, WildcardIdentifier)
}

func IsInterpreterPath(path string) bool {
	name := baseName(path)
	if !usableName(name) {
		return false
	}
	return name == "busybox" || kindOfName(name) != notRuntime
}

func appletKind(name string) runtimeKind {
	if k := kindOfName(name); k == rtShell || k == rtAwk {
		return k
	}
	return notRuntime
}

func resolveDirect(path string, args []string, depth int) (runtimeKind, int) {
	if len(args) == 0 || depth > 4 {
		return notRuntime, 0
	}
	first := baseName(args[0])
	pathName := baseName(path)
	if pathName == "" || !usableName(pathName) {
		pathName = first
	}
	if pathName == "busybox" || first == "busybox" {
		if first == "busybox" {
			if len(args) > 1 {
				if k := appletKind(baseName(args[1])); k != notRuntime {
					return k, 1
				}
			}
			return notRuntime, 0
		}
		if usableName(first) {
			return appletKind(first), 0
		}
		return notRuntime, 0
	}
	if wrapperNames[pathName] {
		if usableName(first) && first != pathName {
			return notRuntime, 0
		}
		start := wrappedStart(pathName, args)
		if start <= 0 || start >= len(args) {
			return notRuntime, 0
		}
		k, off := resolveDirect("", args[start:], depth+1)
		if k == notRuntime {
			return notRuntime, 0
		}
		return k, start + off
	}
	k := kindOfName(pathName)
	if k == notRuntime {
		return notRuntime, 0
	}
	if usableName(first) && kindOfName(first) != k {
		return notRuntime, 0
	}
	return k, 0
}

func wrappedStart(wrapper string, args []string) int {
	valued := map[string]map[string]bool{
		"env":     {"-u": true, "--unset": true, "-C": true, "--chdir": true, "-S": true, "--split-string": true},
		"nice":    {"-n": true, "--adjustment": true},
		"timeout": {"-s": true, "--signal": true, "-k": true, "--kill-after": true},
		"stdbuf":  {"-i": true, "-o": true, "-e": true, "--input": true, "--output": true, "--error": true},
		"xargs":   {"-I": true, "-n": true, "-P": true, "-d": true, "-L": true, "-s": true, "-E": true, "-a": true, "--arg-file": true, "--delimiter": true, "--max-args": true, "--max-procs": true, "--max-lines": true, "--max-chars": true, "--eof": true, "--replace": true},
		"nohup":   {},
	}[wrapper]
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if wrapper == "env" && !strings.HasPrefix(a, "-") && strings.Contains(a, "=") {
			i++
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			if valued[a] {
				i += 2
				continue
			}
			i++
			continue
		}
		break
	}
	if wrapper == "timeout" && i < len(args) {
		i++
	}
	return i
}

type parsed struct {
	main int
	code []int
}

func parseRuntime(kind runtimeKind, args []string) parsed {
	switch kind {
	case rtShell:
		return parseShell(args)
	case rtPython:
		return parsePython(args)
	case rtPerl:
		return parseCluster(args, "eE", "IMmxiCdD", "I", "l0", false)
	case rtRuby:
		return parseCluster(args, "e", "xTW", "rICEF", "0", false)
	case rtNode:
		return parseNode(args)
	case rtPHP:
		return parseCluster(args, "rRBE", "", "cdztf", "", true)
	case rtLua:
		return parseCluster(args, "e", "", "l", "", false)
	case rtAwk:
		return parseAwk(args)
	case rtJava:
		return parseJava(args)
	}
	return parsed{main: len(args)}
}

func parseShell(args []string) parsed {
	code := false
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if a == "-o" || a == "+o" || a == "-O" || a == "+O" || a == "--rcfile" || a == "--init-file" {
			i += 2
			continue
		}
		if strings.HasPrefix(a, "--") {
			i++
			continue
		}
		if (strings.HasPrefix(a, "-") || strings.HasPrefix(a, "+")) && len(a) > 1 {
			if strings.HasPrefix(a, "-") && strings.ContainsRune(a[1:], 'c') {
				code = true
			}
			i++
			continue
		}
		break
	}
	if i >= len(args) {
		return parsed{main: len(args)}
	}
	if code {
		return parsed{main: i, code: []int{i}}
	}
	return parsed{main: i}
}

func parsePython(args []string) parsed {
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "-" {
			return parsed{main: i}
		}
		if a == "--" {
			if i+1 < len(args) {
				return parsed{main: i + 1}
			}
			return parsed{main: len(args)}
		}
		if !strings.HasPrefix(a, "-") {
			return parsed{main: i}
		}
		if strings.HasPrefix(a, "--") {
			if a == "--check-hash-based-pycs" {
				i += 2
				continue
			}
			i++
			continue
		}
		next := i + 1
		for j := 1; j < len(a); j++ {
			switch a[j] {
			case 'c', 'm':
				if j+1 < len(a) {
					if a[j] == 'c' {
						return parsed{main: i, code: []int{i}}
					}
					return parsed{main: i}
				}
				if i+1 < len(args) {
					if a[j] == 'c' {
						return parsed{main: i + 1, code: []int{i + 1}}
					}
					return parsed{main: i + 1}
				}
				return parsed{main: len(args)}
			case 'W', 'X', 'Q':
				if j+1 == len(a) {
					next = i + 2
				}
				j = len(a)
			}
		}
		i = next
	}
	return parsed{main: len(args)}
}

func parseCluster(args []string, codeLetters, restValues, spacedValues, digitValues string, fileIsMain bool) parsed {
	var code []int
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if a == "-" || !strings.HasPrefix(a, "-") {
			break
		}
		if strings.HasPrefix(a, "--") {
			i++
			continue
		}
		next := i + 1
	cluster:
		for j := 1; j < len(a); j++ {
			c := a[j]
			switch {
			case strings.IndexByte(codeLetters, c) >= 0:
				if j+1 < len(a) {
					code = append(code, i)
				} else if i+1 < len(args) {
					code = append(code, i+1)
					next = i + 2
				}
				break cluster
			case fileIsMain && c == 'f':
				if j+1 < len(a) {
					return parsed{main: i, code: code}
				}
				if i+1 < len(args) {
					return parsed{main: i + 1, code: code}
				}
				return parsed{main: len(args), code: code}
			case strings.IndexByte(digitValues, c) >= 0:
				for j+1 < len(a) && a[j+1] >= '0' && a[j+1] <= '9' {
					j++
				}
			case strings.IndexByte(spacedValues, c) >= 0:
				if j+1 == len(a) {
					next = i + 2
				}
				break cluster
			case strings.IndexByte(restValues, c) >= 0:
				break cluster
			}
		}
		i = next
	}
	if len(code) > 0 {
		return parsed{main: code[len(code)-1], code: code}
	}
	if i >= len(args) {
		return parsed{main: len(args)}
	}
	return parsed{main: i}
}

func parseNode(args []string) parsed {
	var code []int
	valued := map[string]bool{"-r": true, "--require": true, "--import": true, "--loader": true, "--experimental-loader": true, "-C": true, "--conditions": true, "--input-type": true, "--title": true}
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if a == "-" || !strings.HasPrefix(a, "-") {
			break
		}
		switch {
		case a == "-e" || a == "--eval" || a == "-p" || a == "--print" || a == "-pe" || a == "-ep":
			if i+1 < len(args) {
				code = append(code, i+1)
			}
			i += 2
			continue
		case strings.HasPrefix(a, "--eval=") || strings.HasPrefix(a, "--print="):
			code = append(code, i)
		case valued[a]:
			i += 2
			continue
		}
		i++
	}
	if len(code) > 0 {
		return parsed{main: code[len(code)-1], code: code}
	}
	if i >= len(args) {
		return parsed{main: len(args)}
	}
	return parsed{main: i}
}

func parseAwk(args []string) parsed {
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if a == "-f" || a == "--file" {
			if i+1 < len(args) {
				return parsed{main: i + 1}
			}
			return parsed{main: len(args)}
		}
		if strings.HasPrefix(a, "-f") && len(a) > 2 {
			return parsed{main: i}
		}
		if a == "-v" || a == "-F" || a == "--assign" || a == "--field-separator" {
			i += 2
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			i++
			continue
		}
		break
	}
	if i < len(args) {
		return parsed{main: i, code: []int{i}}
	}
	return parsed{main: len(args)}
}

func parseJava(args []string) parsed {
	valued := map[string]bool{"-cp": true, "-classpath": true, "--class-path": true, "-p": true, "--module-path": true, "--upgrade-module-path": true, "--add-modules": true, "--limit-modules": true, "--add-exports": true, "--add-reads": true, "--add-opens": true, "--patch-module": true, "--source": true, "--enable-native-access": true}
	i := 1
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-jar" || a == "-m" || a == "--module":
			if i+1 < len(args) {
				return parsed{main: i + 1}
			}
			return parsed{main: len(args)}
		case strings.HasPrefix(a, "--module="):
			return parsed{main: i}
		case valued[a]:
			i += 2
			continue
		case strings.HasPrefix(a, "-") || strings.HasPrefix(a, "@"):
			i++
			continue
		}
		return parsed{main: i}
	}
	return parsed{main: len(args)}
}

func CodeOperands(path string, args []string) []int {
	kind, off := resolveDirect(path, args, 0)
	if kind == notRuntime {
		return nil
	}
	p := parseRuntime(kind, args[off:])
	if len(p.code) == 0 {
		return nil
	}
	out := make([]int, len(p.code))
	for i, c := range p.code {
		out[i] = c + off
	}
	return out
}

func LiteralPrefix(path string, args []string) int {
	kind, off := resolveDirect(path, args, 0)
	if kind == notRuntime {
		return 0
	}
	p := parseRuntime(kind, args[off:])
	if p.main >= len(args)-off {
		return len(args)
	}
	return off + p.main + 1
}

func DirectInterpreterInvocation(path string, args []string) bool {
	return LiteralPrefix(path, args) > 0
}

func GeneralizedCodeSelection(path string, args []string) bool {
	n := LiteralPrefix(path, args)
	for i := 0; i < n && i < len(args); i++ {
		a := args[i]
		if !strings.Contains(a, DynamicIdentifier) && a != ExecArgsWildcard {
			continue
		}
		if i == 0 && usableName(baseName(a)) {
			continue
		}
		return true
	}
	return false
}
