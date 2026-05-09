package stmt

import (
	"testing"
)

func TestLexer_Operators(t *testing.T) {
	input := `= != <> < <= > >= :: :`
	want := []tokenType{tokEq, tokNeq, tokNeq, tokLt, tokLtEq, tokGt, tokGtEq, tokCast, tokColon}
	l := newLexer(input)
	for i, exp := range want {
		tok := l.next()
		if tok.typ != exp {
			t.Errorf("token[%d]: got %d (%q), want %d", i, tok.typ, tok.lit, exp)
		}
	}
}

func TestLexer_Literals(t *testing.T) {
	input := `42 3.14 'hello''world' "quoted" TRUE FALSE NULL`
	l := newLexer(input)
	cases := []struct {
		tp  tokenType
		lit string
	}{
		{tokInteger, "42"},
		{tokFloat, "3.14"},
		{tokString, "hello'world"},
		{tokQIdent, "quoted"},
		{tokBoolean, "TRUE"},
		{tokBoolean, "FALSE"},
		{tokNull, "NULL"},
	}
	for i, c := range cases {
		tok := l.next()
		if tok.typ != c.tp || tok.lit != c.lit {
			t.Errorf("token[%d]: got typ=%d lit=%q, want typ=%d lit=%q", i, tok.typ, tok.lit, c.tp, c.lit)
		}
	}
}

func TestLexer_Comments(t *testing.T) {
	input := `-- line comment
42 /* block
comment */ 99`
	l := newLexer(input)
	tok := l.next()
	if tok.typ != tokInteger || tok.lit != "42" {
		t.Errorf("expected 42, got %q (type %d)", tok.lit, tok.typ)
	}
	tok = l.next()
	if tok.typ != tokInteger || tok.lit != "99" {
		t.Errorf("expected 99, got %q (type %d)", tok.lit, tok.typ)
	}
}

func TestLexer_DollarQuoted(t *testing.T) {
	input := `$$hello world$$`
	l := newLexer(input)
	tok := l.next()
	if tok.typ != tokDollar || tok.lit != "hello world" {
		t.Errorf("got typ=%d lit=%q", tok.typ, tok.lit)
	}
}

func TestLexer_SelectStatement(t *testing.T) {
	input := `SELECT * FROM public.users WHERE age >= 25 AND active = TRUE`
	l := newLexer(input)
	var types []tokenType
	for {
		tok := l.next()
		if tok.typ == tokEOF {
			break
		}
		if tok.typ == tokError {
			t.Fatalf("lexer error: %s", tok.lit)
		}
		types = append(types, tok.typ)
	}
	// SELECT * FROM public . users WHERE age >= 25 AND active = TRUE
	expected := []tokenType{
		tokIdent,   // SELECT
		tokStar,    // *
		tokIdent,   // FROM
		tokIdent,   // public
		tokDot,     // .
		tokIdent,   // users
		tokIdent,   // WHERE
		tokIdent,   // age
		tokGtEq,    // >=
		tokInteger, // 25
		tokIdent,   // AND
		tokIdent,   // active
		tokEq,      // =
		tokBoolean, // TRUE
	}
	if len(types) != len(expected) {
		t.Fatalf("token count: got %d want %d", len(types), len(expected))
	}
	for i, exp := range expected {
		if types[i] != exp {
			t.Errorf("token[%d]: got %d want %d", i, types[i], exp)
		}
	}
}
