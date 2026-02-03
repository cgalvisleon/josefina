package stmt

import (
	"fmt"
	"strings"
)

type Parser struct {
	l   *lexer
	cur token
}

func ParseText(input string) (Stmt, error) {
	p := &Parser{l: newLexer(input)}
	p.cur = p.l.next()
	return p.parseStmt()
}

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
		case "DB":
			return p.parseCreateDb()
		default:
			return nil, p.errf("unknown CREATE target")
		}
	case "GET":
		switch strings.ToUpper(obj) {
		case "DB":
			return p.parseGetDb()
		default:
			return nil, p.errf("unknown GET target")
		}
	case "DROP":
		switch strings.ToUpper(obj) {
		case "DB":
			return p.parseDropDb()
		default:
			return nil, p.errf("unknown DROP target")
		}
	default:
		return nil, p.errf("unknown statement")
	}
}

func (p *Parser) parseCreateUser() (Stmt, error) {
	username, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if p.cur.typ != tokComma {
		return nil, p.errf("expected ',' after username")
	}
	p.advance()

	password, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	if p.cur.typ == tokSemicolon {
		p.advance()
	}
	if p.cur.typ == tokError {
		return nil, fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ != tokEOF {
		return nil, p.errf("unexpected token after password")
	}

	return CreateUserStmt{Username: username, Password: password}, nil
}

func (p *Parser) parseCreateDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	if p.cur.typ == tokSemicolon {
		p.advance()
	}
	if p.cur.typ == tokError {
		return nil, fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ != tokEOF {
		return nil, p.errf("unexpected token after db name")
	}

	return CreateDbStmt{Name: name}, nil
}

func (p *Parser) parseGetDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	if p.cur.typ == tokSemicolon {
		p.advance()
	}
	if p.cur.typ == tokError {
		return nil, fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ != tokEOF {
		return nil, p.errf("unexpected token after db name")
	}

	return GetDbStmt{Name: name}, nil
}

func (p *Parser) parseDropDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	if p.cur.typ == tokSemicolon {
		p.advance()
	}
	if p.cur.typ == tokError {
		return nil, fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ != tokEOF {
		return nil, p.errf("unexpected token after db name")
	}

	return DropDbStmt{Name: name}, nil
}

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

func (p *Parser) advance() {
	p.cur = p.l.next()
}

func (p *Parser) isKeyword(want string) bool {
	return p.cur.typ == tokIdent && strings.EqualFold(p.cur.lit, want)
}

func (p *Parser) errf(msg string) error {
	return fmt.Errorf("%s at %d", msg, p.cur.pos)
}
