package stmt

import (
	"fmt"
	"strings"
	"unicode"
)

type tokenType int

const (
	// Control
	tokEOF   tokenType = iota
	tokError           // lexer error
	// Literals
	tokIdent   // unquoted identifier or keyword
	tokQIdent  // "quoted identifier"
	tokString  // 'string literal'
	tokDollar  // $$dollar-quoted$$
	tokInteger // 42
	tokFloat   // 3.14
	tokBoolean // TRUE / FALSE (resolved from tokIdent)
	tokNull    // NULL (resolved from tokIdent)
	// Punctuation
	tokLParen    // (
	tokRParen    // )
	tokComma     // ,
	tokSemicolon // ;
	tokDot       // .
	tokColon     // :
	tokCast      // ::
	tokStar      // *
	tokPlus      // +
	tokMinus     // -
	tokSlash     // /
	tokPercent   // %
	// Comparison operators
	tokEq    // =
	tokNeq   // <> or !=
	tokLt    // <
	tokLtEq  // <=
	tokGt    // >
	tokGtEq  // >=
	tokLike  // LIKE  (resolved from tokIdent)
	tokILike // ILIKE (resolved from tokIdent)
)

// keywords maps uppercase SQL keywords to their resolved token type.
// Keywords that stay as tokIdent are handled by the parser.
var keywords = map[string]tokenType{
	"TRUE":  tokBoolean,
	"FALSE": tokBoolean,
	"NULL":  tokNull,
	"LIKE":  tokLike,
	"ILIKE": tokILike,
}

type token struct {
	typ tokenType
	lit string // raw literal text
	pos int    // byte offset in source
}

func (t token) String() string {
	switch t.typ {
	case tokEOF:
		return "EOF"
	case tokError:
		return fmt.Sprintf("ERROR(%s)", t.lit)
	default:
		return fmt.Sprintf("%q", t.lit)
	}
}

// ── Lexer ─────────────────────────────────────────────────────────────────────

type lexer struct {
	src []rune
	pos int
}

func newLexer(input string) *lexer {
	return &lexer{src: []rune(input)}
}

func (l *lexer) peek() (rune, bool) {
	if l.pos >= len(l.src) {
		return 0, false
	}
	return l.src[l.pos], true
}

func (l *lexer) peekAt(offset int) (rune, bool) {
	i := l.pos + offset
	if i >= len(l.src) {
		return 0, false
	}
	return l.src[i], true
}

func (l *lexer) consume() rune {
	r := l.src[l.pos]
	l.pos++
	return r
}

func (l *lexer) skipWhitespace() {
	for {
		r, ok := l.peek()
		if !ok || !unicode.IsSpace(r) {
			return
		}
		l.consume()
	}
}

// next returns the next token, skipping whitespace and comments.
func (l *lexer) next() token {
	for {
		l.skipWhitespace()
		start := l.pos
		r, ok := l.peek()
		if !ok {
			return token{typ: tokEOF, pos: start}
		}

		// ── Line comment: -- ──────────────────────────────────────────────────
		if r == '-' {
			if r2, ok2 := l.peekAt(1); ok2 && r2 == '-' {
				l.skipLineComment()
				continue
			}
		}

		// ── Block comment: /* */ ──────────────────────────────────────────────
		if r == '/' {
			if r2, ok2 := l.peekAt(1); ok2 && r2 == '*' {
				if err := l.skipBlockComment(); err != nil {
					return token{typ: tokError, lit: err.Error(), pos: start}
				}
				continue
			}
		}

		switch r {
		case '(':
			l.consume()
			return token{typ: tokLParen, lit: "(", pos: start}
		case ')':
			l.consume()
			return token{typ: tokRParen, lit: ")", pos: start}
		case ',':
			l.consume()
			return token{typ: tokComma, lit: ",", pos: start}
		case ';':
			l.consume()
			return token{typ: tokSemicolon, lit: ";", pos: start}
		case '.':
			l.consume()
			return token{typ: tokDot, lit: ".", pos: start}
		case '*':
			l.consume()
			return token{typ: tokStar, lit: "*", pos: start}
		case '+':
			l.consume()
			return token{typ: tokPlus, lit: "+", pos: start}
		case '-':
			l.consume()
			return token{typ: tokMinus, lit: "-", pos: start}
		case '/':
			l.consume()
			return token{typ: tokSlash, lit: "/", pos: start}
		case '%':
			l.consume()
			return token{typ: tokPercent, lit: "%", pos: start}
		case '=':
			l.consume()
			return token{typ: tokEq, lit: "=", pos: start}
		case '!':
			l.consume()
			if r2, ok2 := l.peek(); ok2 && r2 == '=' {
				l.consume()
				return token{typ: tokNeq, lit: "!=", pos: start}
			}
			return token{typ: tokError, lit: "unexpected '!'", pos: start}
		case '<':
			l.consume()
			if r2, ok2 := l.peek(); ok2 {
				if r2 == '=' {
					l.consume()
					return token{typ: tokLtEq, lit: "<=", pos: start}
				}
				if r2 == '>' {
					l.consume()
					return token{typ: tokNeq, lit: "<>", pos: start}
				}
			}
			return token{typ: tokLt, lit: "<", pos: start}
		case '>':
			l.consume()
			if r2, ok2 := l.peek(); ok2 && r2 == '=' {
				l.consume()
				return token{typ: tokGtEq, lit: ">=", pos: start}
			}
			return token{typ: tokGt, lit: ">", pos: start}
		case ':':
			l.consume()
			if r2, ok2 := l.peek(); ok2 && r2 == ':' {
				l.consume()
				return token{typ: tokCast, lit: "::", pos: start}
			}
			return token{typ: tokColon, lit: ":", pos: start}
		case '\'':
			lit, err := l.readSingleQuoted()
			if err != nil {
				return token{typ: tokError, lit: err.Error(), pos: start}
			}
			return token{typ: tokString, lit: lit, pos: start}
		case '"':
			lit, err := l.readDoubleQuoted()
			if err != nil {
				return token{typ: tokError, lit: err.Error(), pos: start}
			}
			return token{typ: tokQIdent, lit: lit, pos: start}
		case '$':
			// Dollar-quoted string: $$...$$ or $tag$...$tag$
			if tok, err := l.readDollarQuoted(); err != nil {
				return token{typ: tokError, lit: err.Error(), pos: start}
			} else {
				return token{typ: tokDollar, lit: tok, pos: start}
			}
		default:
			if unicode.IsDigit(r) {
				return l.readNumber(start)
			}
			if isIdentStart(r) {
				return l.readIdentOrKeyword(start)
			}
			l.consume()
			return token{typ: tokError, lit: fmt.Sprintf("unexpected character %q", r), pos: start}
		}
	}
}

// ── Readers ───────────────────────────────────────────────────────────────────

func (l *lexer) skipLineComment() {
	for {
		r, ok := l.peek()
		if !ok || r == '\n' {
			return
		}
		l.consume()
	}
}

func (l *lexer) skipBlockComment() error {
	l.consume() // /
	l.consume() // *
	depth := 1
	for depth > 0 {
		r, ok := l.peek()
		if !ok {
			return fmt.Errorf("unterminated block comment")
		}
		l.consume()
		if r == '/' {
			if r2, ok2 := l.peek(); ok2 && r2 == '*' {
				l.consume()
				depth++
				continue
			}
		}
		if r == '*' {
			if r2, ok2 := l.peek(); ok2 && r2 == '/' {
				l.consume()
				depth--
			}
		}
	}
	return nil
}

func (l *lexer) readSingleQuoted() (string, error) {
	l.consume() // opening '
	var b strings.Builder
	for {
		r, ok := l.peek()
		if !ok {
			return "", fmt.Errorf("unterminated string literal")
		}
		l.consume()
		if r == '\'' {
			// PostgreSQL escape: '' → '
			if r2, ok2 := l.peek(); ok2 && r2 == '\'' {
				l.consume()
				b.WriteRune('\'')
				continue
			}
			return b.String(), nil
		}
		if r == '\\' {
			r2, ok2 := l.peek()
			if !ok2 {
				return "", fmt.Errorf("unterminated escape in string")
			}
			l.consume()
			switch r2 {
			case 'n':
				b.WriteRune('\n')
			case 't':
				b.WriteRune('\t')
			case 'r':
				b.WriteRune('\r')
			default:
				b.WriteRune(r2)
			}
			continue
		}
		b.WriteRune(r)
	}
}

func (l *lexer) readDoubleQuoted() (string, error) {
	l.consume() // opening "
	var b strings.Builder
	for {
		r, ok := l.peek()
		if !ok {
			return "", fmt.Errorf("unterminated quoted identifier")
		}
		l.consume()
		if r == '"' {
			// SQL escape: "" → "
			if r2, ok2 := l.peek(); ok2 && r2 == '"' {
				l.consume()
				b.WriteRune('"')
				continue
			}
			return b.String(), nil
		}
		b.WriteRune(r)
	}
}

func (l *lexer) readDollarQuoted() (string, error) {
	l.consume() // first $
	// read optional tag until next $
	var tag strings.Builder
	for {
		r, ok := l.peek()
		if !ok {
			return "", fmt.Errorf("unterminated dollar-quote tag")
		}
		l.consume()
		if r == '$' {
			break
		}
		if !isIdentChar(r) {
			return "", fmt.Errorf("invalid character %q in dollar-quote tag", r)
		}
		tag.WriteRune(r)
	}
	delimiter := "$" + tag.String() + "$"

	var b strings.Builder
	for {
		// look for closing delimiter
		if l.pos+len([]rune(delimiter)) <= len(l.src) {
			candidate := string(l.src[l.pos : l.pos+len([]rune(delimiter))])
			if candidate == delimiter {
				l.pos += len([]rune(delimiter))
				return b.String(), nil
			}
		}
		r, ok := l.peek()
		if !ok {
			return "", fmt.Errorf("unterminated dollar-quoted string")
		}
		l.consume()
		b.WriteRune(r)
	}
}

func (l *lexer) readNumber(start int) token {
	isFloat := false
	for {
		r, ok := l.peek()
		if !ok {
			break
		}
		if unicode.IsDigit(r) {
			l.consume()
			continue
		}
		if r == '.' && !isFloat {
			// look ahead: next char must be digit to be a float (avoids schema.table confusion)
			if r2, ok2 := l.peekAt(1); ok2 && unicode.IsDigit(r2) {
				isFloat = true
				l.consume() // .
				l.consume() // first digit after .
				continue
			}
		}
		if r == 'e' || r == 'E' {
			isFloat = true
			l.consume()
			if rSign, ok2 := l.peek(); ok2 && (rSign == '+' || rSign == '-') {
				l.consume()
			}
			continue
		}
		break
	}
	lit := string(l.src[start:l.pos])
	if isFloat {
		return token{typ: tokFloat, lit: lit, pos: start}
	}
	return token{typ: tokInteger, lit: lit, pos: start}
}

func (l *lexer) readIdentOrKeyword(start int) token {
	for {
		r, ok := l.peek()
		if !ok || !isIdentChar(r) {
			break
		}
		l.consume()
	}
	lit := string(l.src[start:l.pos])
	upper := strings.ToUpper(lit)
	if tp, found := keywords[upper]; found {
		return token{typ: tp, lit: lit, pos: start}
	}
	return token{typ: tokIdent, lit: lit, pos: start}
}

// ── Character class helpers ───────────────────────────────────────────────────

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
