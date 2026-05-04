package stmt

import (
	"fmt"
	"strconv"
	"strings"
)

// ── Keyword helpers ───────────────────────────────────────────────────────────

// kw returns the uppercase literal of the current token when it is a plain
// identifier; otherwise returns "". Does NOT advance the parser.
func (p *Parser) kw() string {
	if p.cur.typ == tokIdent {
		return strings.ToUpper(p.cur.lit)
	}
	return ""
}

// eatKeyword advances past the current token when its uppercase literal
// matches kw and it is a plain identifier. Returns true on match.
func (p *Parser) eatKeyword(kw string) bool {
	if p.cur.typ == tokIdent && strings.ToUpper(p.cur.lit) == kw {
		p.advance()
		return true
	}
	return false
}

// expectKeyword consumes the current token if it matches kw; returns an error
// otherwise.
func (p *Parser) expectKeyword(kw string) error {
	if !p.eatKeyword(kw) {
		return p.errf(fmt.Sprintf("expected keyword %s", kw))
	}
	return nil
}

// eatNull consumes the current token if it is a NULL literal (tokNull) or
// the identifier "NULL". Returns true on match.
func (p *Parser) eatNull() bool {
	if p.cur.typ == tokNull {
		p.advance()
		return true
	}
	if p.cur.typ == tokIdent && strings.ToUpper(p.cur.lit) == "NULL" {
		p.advance()
		return true
	}
	return false
}

// expectNull consumes a NULL token; returns an error otherwise.
func (p *Parser) expectNull() error {
	if p.eatNull() {
		return nil
	}
	return p.errf("expected NULL")
}

// consumeOptionalSemi discards trailing semicolons.
func (p *Parser) consumeOptionalSemi() {
	for p.cur.typ == tokSemicolon {
		p.advance()
	}
}

// isClauseKeyword returns true for keywords that terminate a SELECT/UPDATE/DELETE
// clause so the parser knows when to stop reading column lists or SET assignments.
func isClauseKeyword(kw string) bool {
	switch kw {
	case "WHERE", "ORDER", "GROUP", "HAVING", "LIMIT", "OFFSET", "PAGE",
		"JOIN", "INNER", "LEFT", "RIGHT", "FULL", "OUTER",
		"AND", "OR", "ON", "SET", "FROM", "VALUES", "":
		return true
	}
	return false
}

// ── Table reference ───────────────────────────────────────────────────────────

// parseTableRef parses an optional schema qualifier followed by a table name:
//
//	[schema.]name
func (p *Parser) parseTableRef() (TableRef, error) {
	if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
		return TableRef{}, p.errf("expected table name")
	}
	first := p.cur.lit
	p.advance()
	if p.cur.typ == tokDot {
		p.advance()
		if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
			return TableRef{}, p.errf("expected table name after '.'")
		}
		name := p.cur.lit
		p.advance()
		return TableRef{Schema: first, Name: name}, nil
	}
	return TableRef{Name: first}, nil
}

// ── SELECT ────────────────────────────────────────────────────────────────────

// parseSelect: called immediately after the SELECT keyword has been consumed.
//
//	SELECT * | col [, col ...]
//	  FROM [schema.]table
//	  [INNER|LEFT|RIGHT|FULL JOIN [schema.]table ON col = col [AND ...]]
//	  [WHERE conditions]
//	  [ORDER BY col [ASC|DESC] [, ...]]
//	  [LIMIT n] [OFFSET n | PAGE n]
func (p *Parser) parseSelect() (Stmt, error) {
	cols, err := p.parseSelectColumns()
	if err != nil {
		return nil, err
	}
	if err := p.expectKeyword("FROM"); err != nil {
		return nil, err
	}
	tbl, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}

	st := SelectStmt{Columns: cols, From: tbl}

	for {
		j, ok, err := p.tryParseJoin()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		st.Joins = append(st.Joins, j)
	}

	if p.eatKeyword("WHERE") {
		st.Where, err = p.parseConditions()
		if err != nil {
			return nil, err
		}
	}

	if p.eatKeyword("ORDER") {
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		st.OrderBy, err = p.parseOrderBy()
		if err != nil {
			return nil, err
		}
	}

	if p.eatKeyword("LIMIT") {
		st.Rows, err = p.parseIntLiteral()
		if err != nil {
			return nil, err
		}
	}
	if p.eatKeyword("OFFSET") {
		st.Offset, err = p.parseIntLiteral()
		if err != nil {
			return nil, err
		}
	}
	if p.eatKeyword("PAGE") {
		st.Page, err = p.parseIntLiteral()
		if err != nil {
			return nil, err
		}
	}

	p.consumeOptionalSemi()
	return st, nil
}

// parseSelectColumns: * or a comma-separated list of [table.]column names.
// Returns an empty slice to mean SELECT *.
func (p *Parser) parseSelectColumns() ([]string, error) {
	if p.cur.typ == tokStar {
		p.advance()
		return []string{}, nil
	}
	var cols []string
	for {
		if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
			return nil, p.errf("expected column name in SELECT list")
		}
		col := p.cur.lit
		p.advance()
		if p.cur.typ == tokDot {
			p.advance()
			if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
				return nil, p.errf("expected column name after '.'")
			}
			col = p.cur.lit
			p.advance()
		}
		cols = append(cols, col)
		if p.cur.typ != tokComma {
			break
		}
		p.advance()
		// a star after a comma is invalid but stop cleanly
		if p.cur.typ == tokStar {
			break
		}
	}
	return cols, nil
}

// ── INSERT ────────────────────────────────────────────────────────────────────

// parseInsert: called after INSERT INTO has been consumed.
//
//	[schema.]table [(col, ...)] VALUES (val, ...) [, (val, ...) ...]
func (p *Parser) parseInsert() (Stmt, error) {
	tbl, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}

	var cols []string
	if p.cur.typ == tokLParen {
		cols, err = p.parseIdentList()
		if err != nil {
			return nil, err
		}
	}

	if err := p.expectKeyword("VALUES"); err != nil {
		return nil, err
	}

	var rows []map[string]any
	for {
		vals, err := p.parseValueTuple()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(vals))
		for i, v := range vals {
			key := fmt.Sprintf("col%d", i+1)
			if i < len(cols) {
				key = cols[i]
			}
			row[key] = v
		}
		rows = append(rows, row)
		if p.cur.typ != tokComma {
			break
		}
		p.advance()
	}

	p.consumeOptionalSemi()
	return InsertStmt{Table: tbl, Columns: cols, Rows: rows}, nil
}

// ── UPSERT ────────────────────────────────────────────────────────────────────

// parseUpsert: called after UPSERT INTO has been consumed.
// Same syntax as INSERT; the executor calls model.Upsert instead.
func (p *Parser) parseUpsert() (Stmt, error) {
	tbl, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}

	var cols []string
	if p.cur.typ == tokLParen {
		cols, err = p.parseIdentList()
		if err != nil {
			return nil, err
		}
	}

	if err := p.expectKeyword("VALUES"); err != nil {
		return nil, err
	}

	var rows []map[string]any
	for {
		vals, err := p.parseValueTuple()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(vals))
		for i, v := range vals {
			key := fmt.Sprintf("col%d", i+1)
			if i < len(cols) {
				key = cols[i]
			}
			row[key] = v
		}
		rows = append(rows, row)
		if p.cur.typ != tokComma {
			break
		}
		p.advance()
	}

	p.consumeOptionalSemi()
	return UpsertStmt{Table: tbl, Columns: cols, Rows: rows}, nil
}

// parseIdentList: ( ident [, ident ...] )
func (p *Parser) parseIdentList() ([]string, error) {
	if p.cur.typ != tokLParen {
		return nil, p.errf("expected '('")
	}
	p.advance()
	var idents []string
	for p.cur.typ != tokRParen {
		if p.cur.typ == tokEOF {
			return nil, p.errf("unterminated identifier list")
		}
		if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
			return nil, p.errf("expected identifier in list")
		}
		idents = append(idents, p.cur.lit)
		p.advance()
		if p.cur.typ == tokComma {
			p.advance()
		}
	}
	p.advance() // consume )
	return idents, nil
}

// parseValueTuple: ( val [, val ...] )
func (p *Parser) parseValueTuple() ([]any, error) {
	if p.cur.typ != tokLParen {
		return nil, p.errf("expected '(' in VALUES")
	}
	p.advance()
	var vals []any
	for p.cur.typ != tokRParen {
		if p.cur.typ == tokEOF {
			return nil, p.errf("unterminated VALUES tuple")
		}
		lit, err := p.parseLiteralValue()
		if err != nil {
			return nil, err
		}
		vals = append(vals, lit.Value)
		if p.cur.typ == tokComma {
			p.advance()
		}
	}
	p.advance() // consume )
	return vals, nil
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

// parseUpdate: called after UPDATE has been consumed.
//
//	[schema.]table SET col=val [, col=val ...] [WHERE conditions]
func (p *Parser) parseUpdate() (Stmt, error) {
	tbl, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}
	if err := p.expectKeyword("SET"); err != nil {
		return nil, err
	}
	assignments, err := p.parseSetClause()
	if err != nil {
		return nil, err
	}
	var conds []CondExpr
	if p.eatKeyword("WHERE") {
		conds, err = p.parseConditions()
		if err != nil {
			return nil, err
		}
	}
	p.consumeOptionalSemi()
	return UpdateStmt{Table: tbl, Assignments: assignments, Where: conds}, nil
}

// parseSetClause: col=val [, col=val ...]
func (p *Parser) parseSetClause() ([]Assignment, error) {
	var assignments []Assignment
	for {
		if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
			return nil, p.errf("expected column name in SET")
		}
		if isClauseKeyword(p.kw()) {
			break
		}
		col := p.cur.lit
		p.advance()
		if p.cur.typ != tokEq {
			return nil, p.errf("expected '=' in SET")
		}
		p.advance()
		lit, err := p.parseLiteralValue()
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, Assignment{Column: col, Value: lit.Value})
		if p.cur.typ != tokComma {
			break
		}
		p.advance()
	}
	return assignments, nil
}

// ── DELETE ────────────────────────────────────────────────────────────────────

// parseDelete: called after DELETE FROM has been consumed.
//
//	[schema.]table [WHERE conditions]
func (p *Parser) parseDelete() (Stmt, error) {
	tbl, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}
	var conds []CondExpr
	if p.eatKeyword("WHERE") {
		var err error
		conds, err = p.parseConditions()
		if err != nil {
			return nil, err
		}
	}
	p.consumeOptionalSemi()
	return DeleteStmt{Table: tbl, Where: conds}, nil
}

// ── JOIN ──────────────────────────────────────────────────────────────────────

// tryParseJoin attempts to parse one JOIN clause. Returns (clause, true, nil)
// on success, (zero, false, nil) when the current token is not a join keyword,
// or (zero, false, err) on a parse error.
func (p *Parser) tryParseJoin() (JoinClause, bool, error) {
	var kind JoinKind
	switch p.kw() {
	case "INNER":
		p.advance()
		if err := p.expectKeyword("JOIN"); err != nil {
			return JoinClause{}, false, err
		}
		kind = JoinInner
	case "LEFT":
		p.advance()
		p.eatKeyword("OUTER")
		if err := p.expectKeyword("JOIN"); err != nil {
			return JoinClause{}, false, err
		}
		kind = JoinLeft
	case "RIGHT":
		p.advance()
		p.eatKeyword("OUTER")
		if err := p.expectKeyword("JOIN"); err != nil {
			return JoinClause{}, false, err
		}
		kind = JoinRight
	case "FULL":
		p.advance()
		p.eatKeyword("OUTER")
		if err := p.expectKeyword("JOIN"); err != nil {
			return JoinClause{}, false, err
		}
		kind = JoinFull
	case "JOIN":
		p.advance()
		kind = JoinInner
	default:
		return JoinClause{}, false, nil
	}

	tbl, err := p.parseTableRef()
	if err != nil {
		return JoinClause{}, false, err
	}
	if err := p.expectKeyword("ON"); err != nil {
		return JoinClause{}, false, err
	}
	keys, err := p.parseJoinOn()
	if err != nil {
		return JoinClause{}, false, err
	}
	return JoinClause{Kind: kind, Table: tbl, Keys: keys}, true, nil
}

// parseJoinOn: col = col [AND col = col ...]
// Strips any table qualifier from both sides; the executor resolves from/to
// by matching the key names against the two model schemas.
func (p *Parser) parseJoinOn() (map[string]string, error) {
	keys := make(map[string]string)
	for {
		left, err := p.parseQualifiedIdent()
		if err != nil {
			return nil, err
		}
		if p.cur.typ != tokEq {
			return nil, p.errf("expected '=' in JOIN ON")
		}
		p.advance()
		right, err := p.parseQualifiedIdent()
		if err != nil {
			return nil, err
		}
		keys[left] = right
		if !p.eatKeyword("AND") {
			break
		}
	}
	return keys, nil
}

// parseQualifiedIdent: [qualifier.]name — returns only the trailing name part.
func (p *Parser) parseQualifiedIdent() (string, error) {
	if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
		return "", p.errf("expected identifier")
	}
	name := p.cur.lit
	p.advance()
	if p.cur.typ == tokDot {
		p.advance()
		if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
			return "", p.errf("expected name after '.'")
		}
		name = p.cur.lit
		p.advance()
	}
	return name, nil
}

// ── ORDER BY ──────────────────────────────────────────────────────────────────

// parseOrderBy: col [ASC|DESC] [, col [ASC|DESC] ...]
func (p *Parser) parseOrderBy() ([]OrderItem, error) {
	var items []OrderItem
	for {
		if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
			break
		}
		if isClauseKeyword(p.kw()) {
			break
		}
		col := p.cur.lit
		p.advance()
		asc := true
		if p.eatKeyword("DESC") {
			asc = false
		} else {
			p.eatKeyword("ASC")
		}
		items = append(items, OrderItem{Column: col, Asc: asc})
		if p.cur.typ != tokComma {
			break
		}
		p.advance()
	}
	return items, nil
}

// ── WHERE conditions ──────────────────────────────────────────────────────────

// parseConditions: one or more predicates joined by AND/OR.
func (p *Parser) parseConditions() ([]CondExpr, error) {
	var conds []CondExpr
	conn := ConnNone
	for {
		cond, err := p.parseOneCond(conn)
		if err != nil {
			return nil, err
		}
		conds = append(conds, cond)
		switch p.kw() {
		case "AND":
			p.advance()
			conn = ConnAnd
		case "OR":
			p.advance()
			conn = ConnOr
		default:
			return conds, nil
		}
	}
}

// parseOneCond: parses a single WHERE predicate with its leading connector.
//
//	field op value
//	field IS [NOT] NULL
//	field [NOT] IN (...)
//	field [NOT] BETWEEN val AND val
//	field [NOT] LIKE pattern
func (p *Parser) parseOneCond(conn Connector) (CondExpr, error) {
	field, err := p.parseQualifiedIdent()
	if err != nil {
		return CondExpr{}, err
	}

	switch p.kw() {
	case "IS":
		p.advance()
		notNull := p.eatKeyword("NOT")
		if err := p.expectNull(); err != nil {
			return CondExpr{}, p.errf("expected NULL after IS [NOT]")
		}
		op := OpIsNull
		if notNull {
			op = OpIsNotNull
		}
		return CondExpr{Connector: conn, Field: field, Op: op}, nil

	case "NOT":
		p.advance()
		switch p.kw() {
		case "IN":
			p.advance()
			vals, err := p.parseInList()
			if err != nil {
				return CondExpr{}, err
			}
			return CondExpr{Connector: conn, Field: field, Op: OpNotIn, Value: vals}, nil
		case "BETWEEN":
			p.advance()
			lo, hi, err := p.parseBetweenBounds()
			if err != nil {
				return CondExpr{}, err
			}
			return CondExpr{Connector: conn, Field: field, Op: OpNotBetween, Value: [2]any{lo, hi}}, nil
		case "LIKE":
			p.advance()
			val, err := p.parseLiteralValue()
			if err != nil {
				return CondExpr{}, err
			}
			return CondExpr{Connector: conn, Field: field, Op: OpNotIn, Value: val.Value}, nil
		default:
			return CondExpr{}, p.errf("expected IN, BETWEEN, or LIKE after NOT")
		}

	case "IN":
		p.advance()
		vals, err := p.parseInList()
		if err != nil {
			return CondExpr{}, err
		}
		return CondExpr{Connector: conn, Field: field, Op: OpIn, Value: vals}, nil

	case "BETWEEN":
		p.advance()
		lo, hi, err := p.parseBetweenBounds()
		if err != nil {
			return CondExpr{}, err
		}
		return CondExpr{Connector: conn, Field: field, Op: OpBetween, Value: [2]any{lo, hi}}, nil
	}

	// Standard comparison operator
	op, err := p.parseCondOp()
	if err != nil {
		return CondExpr{}, err
	}
	val, err := p.parseLiteralValue()
	if err != nil {
		return CondExpr{}, err
	}
	return CondExpr{Connector: conn, Field: field, Op: op, Value: val.Value}, nil
}

// parseBetweenBounds: val AND val
func (p *Parser) parseBetweenBounds() (any, any, error) {
	lo, err := p.parseLiteralValue()
	if err != nil {
		return nil, nil, err
	}
	if err := p.expectKeyword("AND"); err != nil {
		return nil, nil, err
	}
	hi, err := p.parseLiteralValue()
	if err != nil {
		return nil, nil, err
	}
	return lo.Value, hi.Value, nil
}

// parseCondOp: maps the current operator token to a CondOp constant.
func (p *Parser) parseCondOp() (CondOp, error) {
	switch p.cur.typ {
	case tokEq:
		p.advance()
		return OpEq, nil
	case tokNeq:
		p.advance()
		return OpNeq, nil
	case tokLt:
		p.advance()
		return OpLt, nil
	case tokLtEq:
		p.advance()
		return OpLtEq, nil
	case tokGt:
		p.advance()
		return OpGt, nil
	case tokGtEq:
		p.advance()
		return OpGtEq, nil
	case tokLike:
		p.advance()
		return OpLike, nil
	case tokILike:
		p.advance()
		return OpILike, nil
	}
	return "", p.errf("expected comparison operator (=, !=, <, <=, >, >=, LIKE, ILIKE)")
}

// parseInList: ( val [, val ...] )
func (p *Parser) parseInList() ([]any, error) {
	if p.cur.typ != tokLParen {
		return nil, p.errf("expected '(' after IN")
	}
	p.advance()
	var vals []any
	for p.cur.typ != tokRParen {
		if p.cur.typ == tokEOF {
			return nil, p.errf("unterminated IN list")
		}
		lit, err := p.parseLiteralValue()
		if err != nil {
			return nil, err
		}
		vals = append(vals, lit.Value)
		if p.cur.typ == tokComma {
			p.advance()
		}
	}
	p.advance() // consume )
	return vals, nil
}

// ── Literal values ────────────────────────────────────────────────────────────

// parseLiteralValue: 'string' | integer | float | TRUE | FALSE | NULL | -number
func (p *Parser) parseLiteralValue() (Literal, error) {
	switch p.cur.typ {
	case tokString, tokDollar:
		lit := Literal{Kind: LitString, Raw: p.cur.lit, Value: p.cur.lit}
		p.advance()
		return lit, nil
	case tokInteger:
		raw := p.cur.lit
		p.advance()
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Literal{}, fmt.Errorf("invalid integer %q", raw)
		}
		return Literal{Kind: LitInteger, Raw: raw, Value: n}, nil
	case tokFloat:
		raw := p.cur.lit
		p.advance()
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return Literal{}, fmt.Errorf("invalid float %q", raw)
		}
		return Literal{Kind: LitFloat, Raw: raw, Value: f}, nil
	case tokBoolean:
		raw := p.cur.lit
		p.advance()
		return Literal{Kind: LitBoolean, Raw: raw, Value: strings.ToUpper(raw) == "TRUE"}, nil
	case tokNull:
		p.advance()
		return Literal{Kind: LitNull, Raw: "NULL", Value: nil}, nil
	case tokMinus:
		p.advance()
		inner, err := p.parseLiteralValue()
		if err != nil {
			return Literal{}, err
		}
		switch v := inner.Value.(type) {
		case int64:
			return Literal{Kind: LitInteger, Raw: "-" + inner.Raw, Value: -v}, nil
		case float64:
			return Literal{Kind: LitFloat, Raw: "-" + inner.Raw, Value: -v}, nil
		}
		return Literal{}, p.errf("'-' must be followed by a number")
	}
	return Literal{}, p.errf("expected literal value (string, number, TRUE, FALSE, NULL)")
}

// parseIntLiteral: wraps parseLiteralValue expecting an integer result.
func (p *Parser) parseIntLiteral() (int, error) {
	lit, err := p.parseLiteralValue()
	if err != nil {
		return 0, err
	}
	v, ok := lit.Value.(int64)
	if !ok {
		return 0, p.errf("expected integer literal")
	}
	return int(v), nil
}
