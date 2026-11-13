package rules

import (
	"strconv"
	"strings"
)

type Class string

const (
	Native        Class = "native"
	Generated     Class = "generated"
	Degraded      Class = "degraded"
	NotApplicable Class = "not-applicable"
)

const catalogueNamespace = "cp"

const eventRoot = "event"

const maxDepth = 128

func rank(c Class) int {
	switch c {
	case Native:
		return 0
	case Generated:
		return 1
	case Degraded:
		return 2
	}
	return 3
}

func Worse(a, b Class) Class {
	if rank(b) > rank(a) {
		return b
	}
	return a
}

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokString
	tokNumber
	tokOp
)

type token struct {
	kind tokenKind
	text string
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || isDigit(c)
}

func stringPrefix(s string, i int) (int, bool) {
	j := i
	for j < len(s) && j-i < 2 && strings.IndexByte("rRbB", s[j]) >= 0 {
		j++
	}
	if j < len(s) && (s[j] == '\'' || s[j] == '"') {
		raw := strings.ContainsAny(s[i:j], "rR")
		return j, raw
	}
	return -1, false
}

func scanString(s string, i int, raw bool) (int, bool) {
	quote := s[i]
	triple := strings.HasPrefix(s[i:], strings.Repeat(string(quote), 3))
	width := 1
	if triple {
		width = 3
	}
	j := i + width
	for j < len(s) {
		c := s[j]
		if c == '\\' && !raw {
			j += 2
			continue
		}
		if c == '\n' && !triple {
			return j, false
		}
		if c == quote {
			if !triple {
				return j + 1, true
			}
			if strings.HasPrefix(s[j:], strings.Repeat(string(quote), 3)) {
				return j + 3, true
			}
		}
		j++
	}
	return len(s), false
}

func scanNumber(s string, i int) int {
	j := i
	if strings.HasPrefix(s[j:], "0x") || strings.HasPrefix(s[j:], "0X") {
		j += 2
		for j < len(s) && (isDigit(s[j]) || (s[j] >= 'a' && s[j] <= 'f') || (s[j] >= 'A' && s[j] <= 'F')) {
			j++
		}
	} else {
		for j < len(s) && isDigit(s[j]) {
			j++
		}
		if j+1 < len(s) && s[j] == '.' && isDigit(s[j+1]) {
			j++
			for j < len(s) && isDigit(s[j]) {
				j++
			}
		}
		if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
			k := j + 1
			if k < len(s) && (s[k] == '+' || s[k] == '-') {
				k++
			}
			if k < len(s) && isDigit(s[k]) {
				for k < len(s) && isDigit(s[k]) {
					k++
				}
				j = k
			}
		}
	}
	if j < len(s) && (s[j] == 'u' || s[j] == 'U') {
		j++
	}
	return j
}

var twoCharOps = []string{"&&", "||", "==", "!=", "<=", ">="}

func tokenize(s string) ([]token, string) {
	var out []token
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f':
			i++
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case c == '\'' || c == '"':
			end, ok := scanString(s, i, false)
			if !ok {
				return nil, "unterminated string literal"
			}
			out = append(out, token{tokString, s[i:end]})
			i = end
		case isIdentStart(c):
			if start, raw := stringPrefix(s, i); start >= 0 {
				end, ok := scanString(s, start, raw)
				if !ok {
					return nil, "unterminated string literal"
				}
				out = append(out, token{tokString, s[i:end]})
				i = end
				continue
			}
			j := i
			for {
				for j < len(s) && isIdentPart(s[j]) {
					j++
				}
				if j+1 < len(s) && s[j] == '.' && isIdentStart(s[j+1]) {
					j++
					continue
				}
				break
			}
			out = append(out, token{tokIdent, s[i:j]})
			i = j
		case isDigit(c) || (c == '.' && i+1 < len(s) && isDigit(s[i+1])):
			end := scanNumber(s, i)
			if end == i {
				end = i + 1
				for end < len(s) && isDigit(s[end]) {
					end++
				}
			}
			out = append(out, token{tokNumber, s[i:end]})
			i = end
		default:
			matched := false
			for _, op := range twoCharOps {
				if strings.HasPrefix(s[i:], op) {
					out = append(out, token{tokOp, op})
					i += 2
					matched = true
					break
				}
			}
			if matched {
				continue
			}
			if strings.IndexByte("!<>+-*/%?:,.()[]{}", c) < 0 {
				return nil, "unexpected character " + strconv.QuoteRune(rune(c))
			}
			out = append(out, token{tokOp, string(c)})
			i++
		}
	}
	return out, ""
}

type finding struct {
	class  Class
	reason string
}

type parser struct {
	toks     []token
	pos      int
	depth    int
	failed   string
	findings []finding
	bound    []string
}

func (p *parser) peek() token {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return token{}
}

func (p *parser) peekAt(n int) token {
	if p.pos+n < len(p.toks) {
		return p.toks[p.pos+n]
	}
	return token{}
}

func (p *parser) next() token {
	t := p.peek()
	if p.pos < len(p.toks) {
		p.pos++
	}
	return t
}

func (p *parser) isOp(text string) bool {
	t := p.peek()
	return t.kind == tokOp && t.text == text
}

func (p *parser) accept(text string) bool {
	if p.isOp(text) {
		p.pos++
		return true
	}
	return false
}

func (p *parser) fail(reason string) {
	if p.failed == "" {
		p.failed = reason
	}
}

func (p *parser) expect(text string) {
	if !p.accept(text) {
		p.fail("expected " + strconv.Quote(text) + " but found " + describe(p.peek()))
	}
}

func describe(t token) string {
	if t.kind == tokEOF {
		return "end of expression"
	}
	return strconv.Quote(t.text)
}

func (p *parser) note(class Class, reason string) {
	for _, f := range p.findings {
		if f.class == class && f.reason == reason {
			return
		}
	}
	p.findings = append(p.findings, finding{class, reason})
}

func (p *parser) enter() bool {
	p.depth++
	if p.depth > maxDepth {
		p.fail("expression nested too deeply")
		return false
	}
	return p.failed == ""
}

func (p *parser) leave() {
	p.depth--
}

func (p *parser) expr() {
	if !p.enter() {
		return
	}
	defer p.leave()
	p.or()
	if p.accept("?") {
		p.note(Degraded, "ternary operator")
		p.expr()
		p.expect(":")
		p.expr()
	}
}

func (p *parser) or() {
	p.and()
	for p.failed == "" && p.accept("||") {
		p.and()
	}
}

func (p *parser) and() {
	p.relation()
	for p.failed == "" && p.accept("&&") {
		p.relation()
	}
}

func (p *parser) relationOp() bool {
	t := p.peek()
	if t.kind == tokIdent && t.text == "in" {
		p.pos++
		return true
	}
	if t.kind != tokOp {
		return false
	}
	switch t.text {
	case "==", "!=", "<", ">", "<=", ">=":
		p.pos++
		return true
	}
	return false
}

func (p *parser) relation() {
	p.sum()
	for p.failed == "" && p.relationOp() {
		p.sum()
	}
}

func (p *parser) sum() {
	p.product()
	for p.failed == "" {
		t := p.peek()
		if t.kind != tokOp || (t.text != "+" && t.text != "-") {
			return
		}
		p.pos++
		p.note(Degraded, "arithmetic operator "+t.text)
		p.product()
	}
}

func (p *parser) product() {
	p.unary()
	for p.failed == "" {
		t := p.peek()
		if t.kind != tokOp || (t.text != "*" && t.text != "/" && t.text != "%") {
			return
		}
		p.pos++
		p.note(Degraded, "arithmetic operator "+t.text)
		p.unary()
	}
}

func (p *parser) unary() {
	if !p.enter() {
		return
	}
	defer p.leave()
	if p.accept("!") {
		p.unary()
		return
	}
	if p.isOp("-") {
		p.pos++
		if p.peek().kind != tokNumber {
			p.note(Degraded, "arithmetic operator -")
		}
		p.unary()
		return
	}
	p.postfix()
}

func (p *parser) postfix() {
	p.primary()
	for p.failed == "" {
		switch {
		case p.accept("."):
			t := p.next()
			if t.kind != tokIdent {
				p.fail("expected a member name but found " + describe(t))
				return
			}
			name := t.text
			if p.isOp("(") {
				p.method(name)
			} else {
				p.note(Degraded, "field selection ."+name+" on a computed value")
			}
		case p.accept("["):
			p.note(Degraded, "index operator")
			p.expr()
			p.expect("]")
		default:
			return
		}
	}
}

func (p *parser) arguments(binder bool) int {
	p.expect("(")
	if p.accept(")") {
		return 0
	}
	n := 0
	pushed := 0
	for p.failed == "" {
		if binder && n == 0 && p.peek().kind == tokIdent && !strings.Contains(p.peek().text, ".") && p.peekAt(1).kind == tokOp && p.peekAt(1).text == "," {
			p.bound = append(p.bound, p.peek().text)
			pushed++
			p.pos++
		} else {
			p.expr()
		}
		n++
		if !p.accept(",") {
			break
		}
	}
	p.expect(")")
	p.bound = p.bound[:len(p.bound)-pushed]
	return n
}

var portableMethods = map[string]bool{"startsWith": true, "endsWith": true, "contains": true}

var macros = map[string]bool{"exists": true, "exists_one": true, "all": true, "map": true, "filter": true}

var conversions = map[string]bool{"string": true, "int": true, "uint": true, "double": true, "bool": true, "bytes": true, "dyn": true, "type": true, "duration": true, "timestamp": true}

func (p *parser) method(name string) {
	switch {
	case portableMethods[name]:
		if n := p.arguments(false); n != 1 && p.failed == "" {
			p.note(Degraded, "method "+name+" called with "+strconv.Itoa(n)+" arguments, wants 1")
		}
	case macros[name]:
		p.note(Degraded, "macro "+name)
		p.arguments(true)
	default:
		p.note(Degraded, "method "+name)
		p.arguments(false)
	}
}

func (p *parser) isBound(name string) bool {
	for _, b := range p.bound {
		if b == name {
			return true
		}
	}
	return false
}

func (p *parser) global(name string) {
	switch {
	case name == "has":
		p.note(Degraded, "macro has")
	case conversions[name]:
		p.note(Degraded, "conversion "+name)
	default:
		p.note(Degraded, "function "+name)
	}
	p.arguments(false)
}

func (p *parser) call(name string) {
	root, rest, dotted := strings.Cut(name, ".")
	if !dotted {
		p.global(name)
		return
	}
	if root == catalogueNamespace {
		n := p.arguments(false)
		if p.failed != "" {
			return
		}
		f, ok := Function(name)
		switch {
		case !ok:
			p.note(NotApplicable, "unknown catalogue function "+name)
		case n != f.Arity:
			p.note(NotApplicable, name+" called with "+strconv.Itoa(n)+" arguments, wants "+strconv.Itoa(f.Arity))
		default:
			p.note(Generated, "calls "+name)
		}
		return
	}
	if root == eventRoot || p.isBound(root) {
		last := rest
		if i := strings.LastIndexByte(rest, '.'); i >= 0 {
			last = rest[i+1:]
		}
		p.method(last)
		return
	}
	p.note(Degraded, "non-catalogue function "+name)
	p.arguments(false)
}

func (p *parser) reference(name string) {
	root, _, _ := strings.Cut(name, ".")
	switch {
	case root == eventRoot || p.isBound(root):
	case root == catalogueNamespace:
		p.note(NotApplicable, "catalogue name "+name+" referenced without a call")
	default:
		p.note(Degraded, "unknown identifier "+name)
	}
}

func (p *parser) primary() {
	t := p.next()
	switch t.kind {
	case tokString, tokNumber:
	case tokIdent:
		switch {
		case t.text == "true" || t.text == "false" || t.text == "null":
		case t.text == "in":
			p.fail("unexpected \"in\"")
		case p.isOp("("):
			p.call(t.text)
		default:
			p.reference(t.text)
		}
	case tokOp:
		switch t.text {
		case "(":
			p.expr()
			p.expect(")")
		case "[":
			for p.failed == "" && !p.isOp("]") {
				p.expr()
				if !p.accept(",") {
					break
				}
			}
			p.expect("]")
		case "{":
			p.note(Degraded, "map literal")
			for p.failed == "" && !p.isOp("}") {
				p.expr()
				p.expect(":")
				p.expr()
				if !p.accept(",") {
					break
				}
			}
			p.expect("}")
		default:
			p.fail("unexpected " + describe(t))
		}
	default:
		p.fail("unexpected " + describe(t))
	}
}

func constant(toks []token) string {
	lo, hi := 0, len(toks)
	for hi-lo >= 3 && toks[lo].kind == tokOp && toks[lo].text == "(" && toks[hi-1].kind == tokOp && toks[hi-1].text == ")" {
		lo++
		hi--
	}
	if hi-lo == 1 && toks[lo].kind == tokIdent && (toks[lo].text == "true" || toks[lo].text == "false") {
		return toks[lo].text
	}
	return ""
}

func Classify(expression string) (Class, []string) {
	toks, bad := tokenize(expression)
	if bad != "" {
		return NotApplicable, []string{"parse error: " + bad}
	}
	if len(toks) == 0 {
		return NotApplicable, []string{"empty expression"}
	}
	if c := constant(toks); c != "" {
		return NotApplicable, []string{"constant expression " + c}
	}
	p := &parser{toks: toks}
	p.expr()
	if p.failed == "" && p.pos < len(p.toks) {
		p.fail("unexpected " + describe(p.peek()))
	}
	if p.failed != "" {
		return NotApplicable, []string{"parse error: " + p.failed}
	}
	class := Native
	for _, f := range p.findings {
		class = Worse(class, f.class)
	}
	var reasons []string
	for _, f := range p.findings {
		if f.class == class {
			reasons = append(reasons, f.reason)
		}
	}
	return class, reasons
}

func ClassifyRule(expressions []string) (Class, []string) {
	if len(expressions) == 0 {
		return NotApplicable, []string{"no expressions"}
	}
	class := Native
	results := make([][]string, len(expressions))
	classes := make([]Class, len(expressions))
	for i, e := range expressions {
		classes[i], results[i] = Classify(e)
		class = Worse(class, classes[i])
	}
	var reasons []string
	seen := map[string]bool{}
	for i := range expressions {
		if classes[i] != class {
			continue
		}
		for _, r := range results[i] {
			if !seen[r] {
				seen[r] = true
				reasons = append(reasons, r)
			}
		}
	}
	return class, reasons
}
