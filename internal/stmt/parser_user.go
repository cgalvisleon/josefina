package stmt

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

	if err := p.consumeTerm("password"); err != nil {
		return nil, err
	}

	return CreateUserStmt{Username: username, Password: password}, nil
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
	if err := p.consumeTerm("password"); err != nil {
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
	if err := p.consumeTerm("password"); err != nil {
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
	if err := p.consumeTerm("new password"); err != nil {
		return nil, err
	}
	return ChangePasswordStmt{Username: username, OldPassword: oldPassword, NewPassword: newPassword}, nil
}
