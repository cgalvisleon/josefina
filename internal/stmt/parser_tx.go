package stmt

// ── Transaction parsers ───────────────────────────────────────────────────────

/**
* parseBegin: called after the BEGIN keyword has been consumed.
* Syntax: BEGIN [TRANSACTION] [;]
* @return Stmt, error
**/
func (p *Parser) parseBegin() (Stmt, error) {
	p.eatKeyword("TRANSACTION")
	p.consumeOptionalSemi()
	return BeginStmt{}, nil
}

/**
* parseCommit: called after the COMMIT keyword has been consumed.
* Syntax: COMMIT [TRANSACTION] [;]
* @return Stmt, error
**/
func (p *Parser) parseCommit() (Stmt, error) {
	p.eatKeyword("TRANSACTION")
	p.consumeOptionalSemi()
	return CommitStmt{}, nil
}

/**
* parseRollback: called after the ROLLBACK keyword has been consumed.
* Syntax: ROLLBACK [TRANSACTION] [;]
* @return Stmt, error
**/
func (p *Parser) parseRollback() (Stmt, error) {
	p.eatKeyword("TRANSACTION")
	p.consumeOptionalSemi()
	return RollbackStmt{}, nil
}
