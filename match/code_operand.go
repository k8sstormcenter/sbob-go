package match

import "strings"

type interpreterKind int

const (
	notInterpreter interpreterKind = iota
	kindShell
	kindPython
	kindPerl
	kindRuby
	kindNode
	kindPHP
	kindLua
	kindAwk
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

func kindOfName(name string) interpreterKind {
	switch {
	case shellNames[name]:
		return kindShell
	case awkNames[name]:
		return kindAwk
	case name == "node" || name == "nodejs":
		return kindNode
	case strings.HasPrefix(name, "python"):
		return kindPython
	case strings.HasPrefix(name, "perl"):
		return kindPerl
	case strings.HasPrefix(name, "ruby"):
		return kindRuby
	case strings.HasPrefix(name, "php"):
		return kindPHP
	case strings.HasPrefix(name, "lua"):
		return kindLua
	}
	return notInterpreter
}

func usableName(name string) bool {
	return name != "" && !strings.Contains(name, DynamicIdentifier) && !strings.Contains(name, WildcardIdentifier)
}

func resolveInterpreter(path string, args []string, depth int) (interpreterKind, int) {
	if len(args) == 0 || depth > 4 {
		return notInterpreter, 0
	}
	argv0 := baseName(args[0])
	pathName := baseName(path)
	if argv0 == "busybox" && len(args) > 1 {
		if k := kindOfName(baseName(args[1])); k == kindShell || k == kindAwk {
			return k, 1
		}
		return notInterpreter, 0
	}
	if usableName(argv0) {
		if k := kindOfName(argv0); k != notInterpreter {
			return k, 0
		}
		if wrapperNames[argv0] {
			return resolveWrapped(argv0, args, depth)
		}
	}
	if pathName != "busybox" && usableName(pathName) {
		if k := kindOfName(pathName); k != notInterpreter {
			return k, 0
		}
		if wrapperNames[pathName] {
			return resolveWrapped(pathName, args, depth)
		}
	}
	return notInterpreter, 0
}

func resolveWrapped(wrapper string, args []string, depth int) (interpreterKind, int) {
	start := wrappedStart(wrapper, args)
	if start <= 0 || start >= len(args) {
		return notInterpreter, 0
	}
	k, off := resolveInterpreter("", args[start:], depth+1)
	if k == notInterpreter {
		return notInterpreter, 0
	}
	return k, start + off
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

func CodeOperands(path string, args []string) []int {
	kind, off := resolveInterpreter(path, args, 0)
	if kind == notInterpreter {
		return nil
	}
	rest := args[off:]
	var idx []int
	switch kind {
	case kindShell:
		idx = shellOperands(rest)
	case kindPython:
		idx = pythonOperands(rest)
	case kindPerl:
		idx = clusterOperands(rest, "eE", "IMmxiCdD", "", "l0")
	case kindRuby:
		idx = clusterOperands(rest, "e", "xTW", "rICEF", "0")
	case kindNode:
		idx = nodeOperands(rest)
	case kindPHP:
		idx = clusterOperands(rest, "rRBE", "", "cdzt", "")
	case kindLua:
		idx = clusterOperands(rest, "e", "", "l", "")
	case kindAwk:
		idx = awkOperands(rest)
	}
	for i := range idx {
		idx[i] += off
	}
	return idx
}

func IsInterpreterPath(path string) bool {
	name := baseName(path)
	if !usableName(name) {
		return false
	}
	return name == "busybox" || kindOfName(name) != notInterpreter
}

func DirectInterpreterInvocation(path string, args []string) bool {
	if len(args) == 0 {
		return false
	}
	if len(CodeOperands(path, args)) > 0 {
		return true
	}
	argv0 := baseName(args[0])
	if argv0 == "busybox" {
		return len(args) > 1 && kindOfName(baseName(args[1])) != notInterpreter
	}
	if usableName(argv0) {
		if kindOfName(argv0) != notInterpreter {
			return true
		}
		if wrapperNames[argv0] {
			k, _ := resolveWrapped(argv0, args, 0)
			return k != notInterpreter
		}
		return false
	}
	return IsInterpreterPath(path)
}

func shellOperands(args []string) []int {
	code := false
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if (a == "-o" || a == "+o" || a == "-O" || a == "+O") && i+1 < len(args) {
			i += 2
			continue
		}
		if a == "--rcfile" || a == "--init-file" {
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
	if code && i < len(args) {
		return []int{i}
	}
	return nil
}

func pythonOperands(args []string) []int {
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "-" || a == "--" || !strings.HasPrefix(a, "-") {
			return nil
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
			case 'c':
				if j+1 < len(a) {
					return []int{i}
				}
				if i+1 < len(args) {
					return []int{i + 1}
				}
				return nil
			case 'm':
				return nil
			case 'W', 'X', 'Q':
				if j+1 == len(a) {
					next = i + 2
				}
				j = len(a)
			}
		}
		i = next
	}
	return nil
}

func clusterOperands(args []string, codeLetters, restValues, spacedValues, digitValues string) []int {
	var idx []int
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" || a == "-" || !strings.HasPrefix(a, "-") {
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
					idx = append(idx, i)
				} else if i+1 < len(args) {
					idx = append(idx, i+1)
					next = i + 2
				}
				break cluster
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
	return idx
}

func nodeOperands(args []string) []int {
	var idx []int
	valued := map[string]bool{"-r": true, "--require": true, "--import": true, "--loader": true, "--experimental-loader": true, "-C": true, "--conditions": true, "--input-type": true, "--title": true}
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" || a == "-" || !strings.HasPrefix(a, "-") {
			break
		}
		switch {
		case a == "-e" || a == "--eval" || a == "-p" || a == "--print" || a == "-pe" || a == "-ep":
			if i+1 < len(args) {
				idx = append(idx, i+1)
			}
			i += 2
			continue
		case strings.HasPrefix(a, "--eval=") || strings.HasPrefix(a, "--print="):
			idx = append(idx, i)
		case valued[a]:
			i += 2
			continue
		}
		i++
	}
	return idx
}

func awkOperands(args []string) []int {
	i := 1
	for i < len(args) {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if a == "-f" || a == "--file" || strings.HasPrefix(a, "-f") && len(a) > 2 {
			return nil
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
		return []int{i}
	}
	return nil
}
