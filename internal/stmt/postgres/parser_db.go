package stmt

func (p *Parser) parseCreateDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeTerm("db name"); err != nil {
		return nil, err
	}
	return CreateDbStmt{Name: name}, nil
}

func (p *Parser) parseUseDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeTerm("db name"); err != nil {
		return nil, err
	}
	return UseDbStmt{Name: name}, nil
}

func (p *Parser) parseGetDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeTerm("db name"); err != nil {
		return nil, err
	}
	return GetDbStmt{Name: name}, nil
}

func (p *Parser) parseDropDb() (Stmt, error) {
	name, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeTerm("db name"); err != nil {
		return nil, err
	}
	return DropDbStmt{Name: name}, nil
}
