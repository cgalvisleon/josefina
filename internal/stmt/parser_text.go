package stmt

import (
	"fmt"
	"strconv"
	"strings"
)

type Parser struct {
	l   *lexer
	cur token
}

/**
* ParseText: parses a string into a list of statements.
* @param input: string to parse
* @return list of statements
**/
func ParseText(input string) ([]Stmt, error) {
	p := &Parser{l: newLexer(input)}
	p.cur = p.l.next()

	result := make([]Stmt, 0)
	for {
		p.skipSeps()
		if p.cur.typ == tokError {
			return nil, fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
		}
		if p.cur.typ == tokEOF {
			return result, nil
		}

		st, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		result = append(result, st)
	}
}

/**
* parseStmt: parses a statement.
* @return statement
**/
func (p *Parser) parseStmt() (Stmt, error) {
	if p.cur.typ == tokError {
		return nil, fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ == tokEOF {
		return nil, fmt.Errorf("empty input")
	}

	verb, err := p.parseKeyword()
	if err != nil {
		return nil, err
	}
	obj, err := p.parseKeyword()
	if err != nil {
		return nil, err
	}

	switch strings.ToUpper(verb) {
	case "CREATE":
		switch strings.ToUpper(obj) {
		case "USER":
			return p.parseCreateUser()
		case "DATABASE":
			return p.parseCreateDb()
		case "SERIE":
			return p.parseCreateSerie()
		default:
			return nil, p.errf("unknown CREATE target")
		}
	case "GET":
		switch strings.ToUpper(obj) {
		case "DATABASE":
			return p.parseGetDb()
		case "USER":
			return p.parseGetUser()
		case "SERIE":
			return p.parseGetSerie()
		default:
			return nil, p.errf("unknown GET target")
		}
	case "DROP":
		switch strings.ToUpper(obj) {
		case "DATABASE":
			return p.parseDropDb()
		case "USER":
			return p.parseDropUser()
		case "SERIE":
			return p.parseDropSerie()
		default:
			return nil, p.errf("unknown DROP target")
		}
	case "USE":
		switch strings.ToUpper(obj) {
		case "DATABASE":
			return p.parseUseDb()
		default:
			return nil, p.errf("unknown USE command")
		}
	case "SET":
		switch strings.ToUpper(obj) {
		case "SERIE":
			return p.parseSetSerie()
		default:
			return nil, p.errf("unknown SET target")
		}
	case "CHANGE":
		switch strings.ToUpper(obj) {
		case "PASSWORD":
			return p.parseChangePassword()
		default:
			return nil, p.errf("unknown CHANGE target")
		}
	default:
		return nil, p.errf("unknown statement")
	}
}

/**
* parseValue: parses a value.
* @return string, error
**/
func (p *Parser) parseValue() (string, error) {
	switch p.cur.typ {
	case tokString, tokIdent:
		v := p.cur.lit
		p.advance()
		if strings.TrimSpace(v) == "" {
			return "", p.errf("empty value")
		}
		return v, nil
	case tokError:
		return "", fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	case tokEOF:
		return "", p.errf("unexpected end of input")
	default:
		return "", p.errf("expected value")
	}
}

/**
* parseInt
* @return int, error
**/
func (p *Parser) parseInt() (int, error) {
	if p.cur.typ == tokError {
		return 0, fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ != tokString && p.cur.typ != tokIdent {
		return 0, p.errf("expected int")
	}

	lit := p.cur.lit
	pos := p.cur.pos
	p.advance()
	if strings.TrimSpace(lit) == "" {
		return 0, fmt.Errorf("expected int at %d", pos)
	}

	result, err := strconv.Atoi(lit)
	if err != nil {
		return 0, fmt.Errorf("expected int at %d", pos)
	}
	return result, nil
}

/**
* parseKeyword
* @return string, error
**/
func (p *Parser) parseKeyword() (string, error) {
	if p.cur.typ == tokError {
		return "", fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ != tokIdent {
		return "", p.errf("expected keyword")
	}
	kw := p.cur.lit
	p.advance()
	if strings.TrimSpace(kw) == "" {
		return "", p.errf("empty keyword")
	}
	return kw, nil
}

/**
* advance
* @return void
**/
func (p *Parser) advance() {
	p.cur = p.l.next()
}

/**
* errf
* @return error
**/
func (p *Parser) errf(msg string) error {
	return fmt.Errorf("%s at %d", msg, p.cur.pos)
}

/**
* consumeTerm
* @return error
**/
func (p *Parser) consumeTerm(after string) error {
	if p.cur.typ == tokError {
		return fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ == tokEOF {
		return nil
	}
	if p.cur.typ != tokSemicolon && p.cur.typ != tokNewline {
		return p.errf("expected ';' or newline after " + after)
	}
	for p.cur.typ == tokSemicolon || p.cur.typ == tokNewline {
		p.advance()
	}
	if p.cur.typ == tokError {
		return fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	return nil
}

/**
* skipSeps
* @return void
**/
func (p *Parser) skipSeps() {
	for p.cur.typ == tokSemicolon || p.cur.typ == tokNewline {
		p.advance()
	}
}
