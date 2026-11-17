package match

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"sync"
)

const (
	MaxExecRegexpLen = 16 << 10
	maxPooledSubject = 64 << 10
)

var (
	ErrExecRegexpTooLong  = errors.New("exec regexp too long")
	ErrExecRegexpWithArgs = errors.New("exec entry carries both args and argsRegex")
)

type ExecRegexp struct {
	expr     string
	re       *regexp.Regexp
	prefix   string
	complete bool
	err      error
}

func CompileExecRegexp(expr string) (*ExecRegexp, error) {
	if len(expr) > MaxExecRegexpLen {
		return nil, fmt.Errorf("%w: %d bytes, limit %d", ErrExecRegexpTooLong, len(expr), MaxExecRegexpLen)
	}
	parsed, err := syntax.Parse(expr, syntax.ClassNL)
	if err != nil {
		return nil, err
	}
	anchored := &syntax.Regexp{Op: syntax.OpConcat, Sub: []*syntax.Regexp{{Op: syntax.OpBeginText}, parsed, {Op: syntax.OpEndText}}}
	re, err := regexp.Compile(perlSyntax(anchored))
	if err != nil {
		return nil, err
	}
	x := &ExecRegexp{expr: expr, re: re}
	if lit, err := regexp.Compile(perlSyntax(parsed)); err == nil {
		x.prefix, x.complete = lit.LiteralPrefix()
	}
	return x, nil
}

func ExecRegexpOf(expr string) *ExecRegexp {
	if expr == "" {
		return nil
	}
	x, err := CompileExecRegexp(expr)
	if err != nil {
		return &ExecRegexp{expr: expr, err: err}
	}
	return x
}

func (x *ExecRegexp) Valid() bool {
	return x != nil && x.re != nil
}

func (x *ExecRegexp) Err() error {
	if x == nil {
		return nil
	}
	return x.err
}

func (x *ExecRegexp) String() string {
	if x == nil {
		return ""
	}
	return x.expr
}

func (x *ExecRegexp) MatchArgs(args []string) bool {
	if !x.Valid() {
		return false
	}
	sub := subject{args: args}
	ok := x.match(&sub)
	sub.release()
	return ok
}

func AppendExecSubject(dst []byte, args []string) []byte {
	for i, a := range args {
		if i > 0 {
			dst = append(dst, ' ')
		}
		dst = append(dst, a...)
	}
	return dst
}

func ExecRegexpAdmits(regexps []*ExecRegexp, args []string) bool {
	sub := subject{args: args}
	ok := false
	for _, x := range regexps {
		if x.Valid() && x.match(&sub) {
			ok = true
			break
		}
	}
	sub.release()
	return ok
}

func ExecRegexpDenies(regexps []*ExecRegexp, args []string) bool {
	for _, x := range regexps {
		if x != nil && !x.Valid() {
			return true
		}
	}
	return ExecRegexpAdmits(regexps, args)
}

func (x *ExecRegexp) match(sub *subject) bool {
	if !argsHavePrefix(sub.args, x.prefix) {
		return false
	}
	if x.complete {
		return subjectLen(sub.args) == len(x.prefix)
	}
	return x.re.Match(sub.bytes())
}

var subjectPool = sync.Pool{New: func() any {
	b := make([]byte, 0, 512)
	return &b
}}

type subject struct {
	args []string
	buf  *[]byte
	s    []byte
}

func (sub *subject) bytes() []byte {
	if sub.buf == nil {
		sub.buf = subjectPool.Get().(*[]byte)
		sub.s = AppendExecSubject((*sub.buf)[:0], sub.args)
	}
	return sub.s
}

func (sub *subject) release() {
	if sub.buf == nil {
		return
	}
	if cap(sub.s) <= maxPooledSubject {
		*sub.buf = sub.s[:0]
		subjectPool.Put(sub.buf)
	}
	sub.buf, sub.s = nil, nil
}

func argsHavePrefix(args []string, prefix string) bool {
	for i, a := range args {
		if prefix == "" {
			return true
		}
		if i > 0 {
			if prefix[0] != ' ' {
				return false
			}
			prefix = prefix[1:]
			if prefix == "" {
				return true
			}
		}
		n := min(len(a), len(prefix))
		if a[:n] != prefix[:n] {
			return false
		}
		prefix = prefix[n:]
	}
	return prefix == ""
}

func subjectLen(args []string) int {
	n := 0
	for i, a := range args {
		if i > 0 {
			n++
		}
		n += len(a)
	}
	return n
}
