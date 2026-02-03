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
		case "SERIE":
			return p.parseCreateSerie()
		default:
			return nil, p.errf("unknown CREATE target")
		}
	case "GET":
		switch strings.ToUpper(obj) {
		case "DB":
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
		case "DB":
			return p.parseDropDb()
		case "USER":
			return p.parseDropUser()
		case "SERIE":
			return p.parseDropSerie()
		default:
			return nil, p.errf("unknown DROP target")
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

	if err := p.consumeEnd("password"); err != nil {
		return nil, err
	}

	return CreateUserStmt{Username: username, Password: password}, nil
}

func (p *Parser) parseCreateDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("db name"); err != nil {
		return nil, err
	}
	return CreateDbStmt{Name: name}, nil
}

func (p *Parser) parseGetDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("db name"); err != nil {
		return nil, err
	}
	return GetDbStmt{Name: name}, nil
}

func (p *Parser) parseDropDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("db name"); err != nil {
		return nil, err
	}
	return DropDbStmt{Name: name}, nil
}

func (p *Parser) parseGetUser() (Stmt, error) {
	username, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	password, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("password"); err != nil {
		return nil, err
	}
	return GetUserStmt{Username: username, Password: password}, nil
}

func (p *Parser) parseDropUser() (Stmt, error) {
	username, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	password, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("password"); err != nil {
		return nil, err
	}
	return DropUserStmt{Username: username, Password: password}, nil
}

func (p *Parser) parseChangePassword() (Stmt, error) {
	username, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	oldPassword, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	newPassword, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("new password"); err != nil {
		return nil, err
	}
	return ChangePasswordStmt{Username: username, OldPassword: oldPassword, NewPassword: newPassword}, nil
}

func (p *Parser) parseCreateSerie() (Stmt, error) {
	tag, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	format, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	value, err := p.parseInt()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("serie value"); err != nil {
		return nil, err
	}
	return CreateSerieStmt{Tag: tag, Format: format, Value: value}, nil
}

func (p *Parser) parseSetSerie() (Stmt, error) {
	tag, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	value, err := p.parseInt()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("serie value"); err != nil {
		return nil, err
	}
	return SetSerieStmt{Tag: tag, Value: value}, nil
}

func (p *Parser) parseGetSerie() (Stmt, error) {
	tag, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("serie tag"); err != nil {
		return nil, err
	}
	return GetSerieStmt{Tag: tag}, nil
}

func (p *Parser) parseDropSerie() (Stmt, error) {
	tag, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeEnd("serie tag"); err != nil {
		return nil, err
	}
	return DropSerieStmt{Tag: tag}, nil
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

func (p *Parser) consumeEnd(after string) error {
	if p.cur.typ == tokSemicolon {
		p.advance()
	}
	if p.cur.typ == tokError {
		return fmt.Errorf("%s at %d", p.cur.lit, p.cur.pos)
	}
	if p.cur.typ != tokEOF {
		return p.errf("unexpected token after " + after)
	}
	return nil
}
