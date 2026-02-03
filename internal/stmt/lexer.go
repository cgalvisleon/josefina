package stmt

import (
	"fmt"
	"strings"
	"unicode"
)

type tokenType int

const (
	tokEOF tokenType = iota
	tokError
	tokIdent
	tokString
	tokComma
	tokSemicolon
	tokNewline
)

type token struct {
	typ tokenType
	lit string
	pos int
}

type lexer struct {
	src []rune
	i   int
}

func newLexer(input string) *lexer {
	return &lexer{src: []rune(input)}
}

func (l *lexer) next() token {
	l.skipSpaces()
	start := l.i
	if l.i >= len(l.src) {
		return token{typ: tokEOF, pos: start}
	}

	r := l.src[l.i]
	switch r {
	case ',':
		l.i++
		return token{typ: tokComma, lit: ",", pos: start}
	case ';':
		l.i++
		return token{typ: tokSemicolon, lit: ";", pos: start}
	case '\n':
		l.i++
		return token{typ: tokNewline, lit: "\n", pos: start}
	case '\'', '"':
		lit, err := l.readQuoted(r)
		if err != nil {
			return token{typ: tokError, lit: err.Error(), pos: start}
		}
		return token{typ: tokString, lit: lit, pos: start}
	default:
		if isIdentStart(r) {
			lit := l.readIdent()
			return token{typ: tokIdent, lit: lit, pos: start}
		}
		l.i++
		return token{typ: tokError, lit: fmt.Sprintf("unexpected character %q", r), pos: start}
	}
}

func (l *lexer) skipSpaces() {
	for l.i < len(l.src) {
		r := l.src[l.i]
		if r == '\n' {
			return
		}
		if !unicode.IsSpace(r) {
			return
		}
		l.i++
	}
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.'
}

func (l *lexer) readIdent() string {
	start := l.i
	for l.i < len(l.src) {
		r := l.src[l.i]
		if r == '\n' || unicode.IsSpace(r) || r == ',' || r == ';' {
			break
		}
		l.i++
	}
	return string(l.src[start:l.i])
}

func (l *lexer) readQuoted(quote rune) (string, error) {
	l.i++
	var b strings.Builder
	for l.i < len(l.src) {
		r := l.src[l.i]
		if r == quote {
			l.i++
			return b.String(), nil
		}
		if r == '\\' {
			if l.i+1 >= len(l.src) {
				return "", fmt.Errorf("unterminated escape")
			}
			n := l.src[l.i+1]
			switch n {
			case '\\', '\'', '"':
				b.WriteRune(n)
				l.i += 2
				continue
			case 'n':
				b.WriteRune('\n')
				l.i += 2
				continue
			case 't':
				b.WriteRune('\t')
				l.i += 2
				continue
			default:
				b.WriteRune(n)
				l.i += 2
				continue
			}
		}
		b.WriteRune(r)
		l.i++
	}
	return "", fmt.Errorf("unterminated string")
}
