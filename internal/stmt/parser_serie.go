package stmt

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
	if err := p.consumeTerm("serie value"); err != nil {
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
	if err := p.consumeTerm("serie value"); err != nil {
		return nil, err
	}
	return SetSerieStmt{Tag: tag, Value: value}, nil
}

func (p *Parser) parseGetSerie() (Stmt, error) {
	tag, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeTerm("serie tag"); err != nil {
		return nil, err
	}
	return GetSerieStmt{Tag: tag}, nil
}

func (p *Parser) parseDropSerie() (Stmt, error) {
	tag, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if err := p.consumeTerm("serie tag"); err != nil {
		return nil, err
	}
	return DropSerieStmt{Tag: tag}, nil
}
