package match

import (
	"regexp/syntax"
	"strconv"
	"strings"
)

func perlSyntax(re *syntax.Regexp) string {
	var b strings.Builder
	writePerl(&b, re)
	return b.String()
}

func writePerl(b *strings.Builder, re *syntax.Regexp) {
	switch re.Op {
	case syntax.OpNoMatch:
		b.WriteString(`[^\x00-\x{10FFFF}]`)
	case syntax.OpEmptyMatch:
		b.WriteString(`(?:)`)
	case syntax.OpLiteral:
		if re.Flags&syntax.FoldCase != 0 {
			b.WriteString(`(?i:`)
		}
		for _, r := range re.Rune {
			writePerlRune(b, r)
		}
		if re.Flags&syntax.FoldCase != 0 {
			b.WriteByte(')')
		}
	case syntax.OpCharClass:
		if len(re.Rune) == 0 {
			b.WriteString(`[^\x00-\x{10FFFF}]`)
			return
		}
		b.WriteByte('[')
		for i := 0; i+1 < len(re.Rune); i += 2 {
			writePerlRune(b, re.Rune[i])
			if re.Rune[i+1] != re.Rune[i] {
				b.WriteByte('-')
				writePerlRune(b, re.Rune[i+1])
			}
		}
		b.WriteByte(']')
	case syntax.OpAnyCharNotNL:
		b.WriteString(`(?-s:.)`)
	case syntax.OpAnyChar:
		b.WriteString(`(?s:.)`)
	case syntax.OpBeginLine:
		b.WriteString(`(?m:^)`)
	case syntax.OpEndLine:
		b.WriteString(`(?m:$)`)
	case syntax.OpBeginText:
		b.WriteString(`\A`)
	case syntax.OpEndText:
		b.WriteString(`\z`)
	case syntax.OpWordBoundary:
		b.WriteString(`\b`)
	case syntax.OpNoWordBoundary:
		b.WriteString(`\B`)
	case syntax.OpCapture:
		b.WriteByte('(')
		if re.Name != "" {
			b.WriteString("?P<" + re.Name + ">")
		}
		writePerl(b, re.Sub[0])
		b.WriteByte(')')
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		b.WriteString(`(?:`)
		writePerl(b, re.Sub[0])
		b.WriteByte(')')
		switch re.Op {
		case syntax.OpStar:
			b.WriteByte('*')
		case syntax.OpPlus:
			b.WriteByte('+')
		case syntax.OpQuest:
			b.WriteByte('?')
		default:
			b.WriteByte('{')
			b.WriteString(strconv.Itoa(re.Min))
			if re.Max != re.Min {
				b.WriteByte(',')
				if re.Max >= 0 {
					b.WriteString(strconv.Itoa(re.Max))
				}
			}
			b.WriteByte('}')
		}
		if re.Flags&syntax.NonGreedy != 0 {
			b.WriteByte('?')
		}
	case syntax.OpConcat:
		b.WriteString(`(?:`)
		for _, sub := range re.Sub {
			writePerl(b, sub)
		}
		b.WriteByte(')')
	case syntax.OpAlternate:
		b.WriteString(`(?:`)
		for i, sub := range re.Sub {
			if i > 0 {
				b.WriteByte('|')
			}
			writePerl(b, sub)
		}
		b.WriteByte(')')
	default:
		b.WriteString(re.String())
	}
}

func writePerlRune(b *strings.Builder, r rune) {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		b.WriteRune(r)
		return
	}
	b.WriteString(`\x{`)
	b.WriteString(strconv.FormatInt(int64(r), 16))
	b.WriteByte('}')
}
