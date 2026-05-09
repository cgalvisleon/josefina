package stmt

// ── CACHE commands ────────────────────────────────────────────────────────────
//
//	SET   CACHE key value [duration_secs]
//	GET   CACHE key
//	DEL   CACHE key
//	EXIST CACHE key

func (p *Parser) parseSetCache() (Stmt, error) {
	key, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	var duration float64
	if p.cur.typ == tokFloat || p.cur.typ == tokInteger {
		lit, err := p.parseLiteralValue()
		if err != nil {
			return nil, err
		}
		switch v := lit.Value.(type) {
		case float64:
			duration = v
		case int64:
			duration = float64(v)
		}
	}

	p.consumeOptionalSemi()
	return SetCacheStmt{Key: key, Value: value, Duration: duration}, nil
}

func (p *Parser) parseGetCache() (Stmt, error) {
	key, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.consumeOptionalSemi()
	return GetCacheStmt{Key: key}, nil
}

func (p *Parser) parseDelCache() (Stmt, error) {
	key, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.consumeOptionalSemi()
	return DelCacheStmt{Key: key}, nil
}

func (p *Parser) parseExistCache() (Stmt, error) {
	key, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.consumeOptionalSemi()
	return ExistCacheStmt{Key: key}, nil
}
