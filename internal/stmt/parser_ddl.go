package stmt

import "strings"

// ── SQL type resolution ───────────────────────────────────────────────────────

// sqlTypeFrom maps an uppercase SQL type keyword to a SqlType constant.
// Aliases (INTEGER→INT, BOOL→BOOLEAN, etc.) are normalised here so the
// executor only has to handle the canonical set.
func sqlTypeFrom(kw string) (SqlType, bool) {
	switch strings.ToUpper(kw) {
	case "TEXT", "VARCHAR", "CHAR", "CHARACTER", "VARYING":
		return SqlText, true
	case "INT", "INTEGER":
		return SqlInt, true
	case "BIGINT":
		return SqlBigInt, true
	case "SMALLINT":
		return SqlSmallInt, true
	case "FLOAT", "DOUBLE":
		return SqlFloat, true
	case "REAL":
		return SqlReal, true
	case "NUMERIC", "DECIMAL":
		return SqlNumeric, true
	case "BOOL", "BOOLEAN":
		return SqlBool, true
	case "JSON":
		return SqlJson, true
	case "JSONB":
		return SqlJsonb, true
	case "TIMESTAMP", "TIMESTAMPTZ":
		return SqlTimestamp, true
	case "DATE":
		return SqlDate, true
	case "BYTEA":
		return SqlBytes, true
	case "KEY": // josefina-specific natural key
		return SqlKey, true
	}
	return "", false
}

// ── CREATE TABLE ──────────────────────────────────────────────────────────────

// parseCreateTable: called after CREATE TABLE has been consumed.
//
//	CREATE TABLE [IF NOT EXISTS] [schema.]name (
//	    col_name type [DEFAULT val] [NOT NULL] [PRIMARY KEY] [UNIQUE]
//	    [, ...]
//	    [, PRIMARY KEY (col, ...)]
//	    [, UNIQUE (col)]
//	)
func (p *Parser) parseCreateTable() (Stmt, error) {
	ifNotExists := false
	if p.eatKeyword("IF") {
		if err := p.expectKeyword("NOT"); err != nil {
			return nil, err
		}
		if err := p.expectKeyword("EXISTS"); err != nil {
			return nil, err
		}
		ifNotExists = true
	}

	tbl, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}

	if p.cur.typ != tokLParen {
		return nil, p.errf("expected '(' after table name in CREATE TABLE")
	}
	p.advance()

	st := CreateTableStmt{Table: tbl, IfNotExists: ifNotExists}

	for p.cur.typ != tokRParen {
		if p.cur.typ == tokEOF {
			return nil, p.errf("unterminated CREATE TABLE definition")
		}

		// Table-level constraints: PRIMARY KEY (...) or UNIQUE (...)
		if p.kw() == "PRIMARY" {
			p.advance()
			if err := p.expectKeyword("KEY"); err != nil {
				return nil, err
			}
			cols, err := p.parseIdentList()
			if err != nil {
				return nil, err
			}
			st.PrimaryKeys = append(st.PrimaryKeys, cols...)
			p.eatCommaOrRParen(&st)
			continue
		}
		if p.kw() == "UNIQUE" {
			p.advance()
			cols, err := p.parseIdentList()
			if err != nil {
				return nil, err
			}
			st.Unique = append(st.Unique, cols...)
			p.eatCommaOrRParen(&st)
			continue
		}

		// Column definition
		col, err := p.parseColumnDef()
		if err != nil {
			return nil, err
		}

		// Collect inline constraints into the table-level slices
		if col.PrimaryKey {
			st.PrimaryKeys = append(st.PrimaryKeys, col.Name)
		}
		if col.Unique {
			st.Unique = append(st.Unique, col.Name)
		}
		if col.NotNull {
			st.Required = append(st.Required, col.Name)
		}
		st.Columns = append(st.Columns, col)

		if p.cur.typ == tokComma {
			p.advance()
		}
	}
	p.advance() // consume )
	p.consumeOptionalSemi()
	return st, nil
}

// eatCommaOrRParen skips a comma after a table constraint.
// Does nothing when ) is next (last item).
func (p *Parser) eatCommaOrRParen(_ *CreateTableStmt) {
	if p.cur.typ == tokComma {
		p.advance()
	}
}

// parseColumnDef: col_name type [DEFAULT val] [NOT NULL | NULL] [PRIMARY KEY] [UNIQUE]
func (p *Parser) parseColumnDef() (ColumnDef, error) {
	if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
		return ColumnDef{}, p.errf("expected column name")
	}
	name := p.cur.lit
	p.advance()

	// SQL type — may span two tokens: CHARACTER VARYING, DOUBLE PRECISION, etc.
	if p.cur.typ != tokIdent {
		return ColumnDef{}, p.errf("expected type name for column " + name)
	}
	typeKw := strings.ToUpper(p.cur.lit)
	p.advance()

	// Absorb optional length/precision: VARCHAR(255), NUMERIC(10,2) — ignore the values
	if p.cur.typ == tokLParen {
		if err := p.skipParenGroup(); err != nil {
			return ColumnDef{}, err
		}
	}

	// Two-word types
	if typeKw == "CHARACTER" && p.kw() == "VARYING" {
		p.advance()
		typeKw = "VARCHAR"
	} else if typeKw == "DOUBLE" && p.kw() == "PRECISION" {
		p.advance()
		typeKw = "FLOAT"
	} else if typeKw == "TIMESTAMP" && (p.kw() == "WITH" || p.kw() == "WITHOUT") {
		p.advance() // WITH/WITHOUT
		p.advance() // TIME
		p.advance() // ZONE
		typeKw = "TIMESTAMP"
	}

	sqlType, ok := sqlTypeFrom(typeKw)
	if !ok {
		return ColumnDef{}, p.errf("unknown SQL type: " + typeKw)
	}

	col := ColumnDef{Name: name, Type: sqlType}

	// Optional column constraints (order-independent)
	for {
		// NULL token (not a tokIdent) means explicit nullable — skip it
		if p.cur.typ == tokNull {
			p.advance()
			continue
		}
		switch p.kw() {
		case "DEFAULT":
			p.advance()
			lit, err := p.parseLiteralValue()
			if err != nil {
				return ColumnDef{}, err
			}
			col.Default = lit.Value
			continue
		case "NOT":
			p.advance()
			if err := p.expectNull(); err != nil {
				return ColumnDef{}, err
			}
			col.NotNull = true
			continue
		case "NULL": // explicit NULL — no-op, column is nullable by default
			p.advance()
			continue
		case "PRIMARY":
			p.advance()
			if err := p.expectKeyword("KEY"); err != nil {
				return ColumnDef{}, err
			}
			col.PrimaryKey = true
			col.NotNull = true
			continue
		case "UNIQUE":
			p.advance()
			col.Unique = true
			continue
		case "REFERENCES": // FK inline hint — skip for now
			p.advance()
			_, err := p.parseTableRef()
			if err != nil {
				return ColumnDef{}, err
			}
			if p.cur.typ == tokLParen {
				if err := p.skipParenGroup(); err != nil {
					return ColumnDef{}, err
				}
			}
			continue
		}
		break
	}

	return col, nil
}

// skipParenGroup: consumes ( ... ) including nested parens.
func (p *Parser) skipParenGroup() error {
	if p.cur.typ != tokLParen {
		return p.errf("expected '('")
	}
	depth := 1
	p.advance()
	for depth > 0 {
		if p.cur.typ == tokEOF {
			return p.errf("unterminated parenthesis group")
		}
		if p.cur.typ == tokLParen {
			depth++
		} else if p.cur.typ == tokRParen {
			depth--
		}
		p.advance()
	}
	return nil
}

// ── DROP TABLE ────────────────────────────────────────────────────────────────

// parseDropTable: called after DROP TABLE has been consumed.
//
//	DROP TABLE [IF EXISTS] [schema.]name
func (p *Parser) parseDropTable() (Stmt, error) {
	ifExists := false
	if p.eatKeyword("IF") {
		if err := p.expectKeyword("EXISTS"); err != nil {
			return nil, err
		}
		ifExists = true
	}
	tbl, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}
	p.consumeOptionalSemi()
	return DropTableStmt{Table: tbl, IfExists: ifExists}, nil
}

// ── ALTER TABLE ───────────────────────────────────────────────────────────────

// parseAlterTable: called after ALTER TABLE has been consumed.
//
//	ALTER TABLE [schema.]name ADD [COLUMN] col_def
//	ALTER TABLE [schema.]name DROP [COLUMN] col_name
//	ALTER TABLE [schema.]name ALTER [COLUMN] col_name SET DEFAULT val
//	ALTER TABLE [schema.]name ALTER [COLUMN] col_name DROP DEFAULT
//	ALTER TABLE [schema.]name ALTER [COLUMN] col_name SET NOT NULL
//	ALTER TABLE [schema.]name ALTER [COLUMN] col_name DROP NOT NULL
//	ALTER TABLE [schema.]name ADD PRIMARY KEY (cols)
//	ALTER TABLE [schema.]name ADD UNIQUE (col)
func (p *Parser) parseAlterTable() (Stmt, error) {
	tbl, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}

	action, err := p.parseKeyword()
	if err != nil {
		return nil, err
	}

	st := AlterTableStmt{Table: tbl}

	switch strings.ToUpper(action) {
	case "ADD":
		// ADD [COLUMN] col_def  |  ADD PRIMARY KEY (...)  |  ADD UNIQUE (...)
		if p.kw() == "PRIMARY" {
			p.advance()
			if err := p.expectKeyword("KEY"); err != nil {
				return nil, err
			}
			cols, err := p.parseIdentList()
			if err != nil {
				return nil, err
			}
			st.Action = AlterAddPrimaryKey
			st.Column = ColumnDef{Name: strings.Join(cols, ",")}
		} else if p.kw() == "UNIQUE" {
			p.advance()
			cols, err := p.parseIdentList()
			if err != nil {
				return nil, err
			}
			st.Action = AlterAddUnique
			st.Column = ColumnDef{Name: strings.Join(cols, ",")}
		} else {
			p.eatKeyword("COLUMN")
			col, err := p.parseColumnDef()
			if err != nil {
				return nil, err
			}
			st.Action = AlterAddColumn
			st.Column = col
		}

	case "DROP":
		p.eatKeyword("COLUMN")
		if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
			return nil, p.errf("expected column name after DROP COLUMN")
		}
		st.Action = AlterDropColumn
		st.ColName = p.cur.lit
		p.advance()

	case "ALTER":
		p.eatKeyword("COLUMN")
		if p.cur.typ != tokIdent && p.cur.typ != tokQIdent {
			return nil, p.errf("expected column name after ALTER COLUMN")
		}
		colName := p.cur.lit
		p.advance()

		subAction, err := p.parseKeyword()
		if err != nil {
			return nil, err
		}
		st.Action = AlterAlterColumn
		st.ColName = colName

		switch strings.ToUpper(subAction) {
		case "SET":
			switch p.kw() {
			case "DEFAULT":
				p.advance()
				lit, err := p.parseLiteralValue()
				if err != nil {
					return nil, err
				}
				st.Column = ColumnDef{Name: colName, Default: lit.Value}
			case "NOT":
				p.advance()
				if err := p.expectNull(); err != nil {
					return nil, err
				}
				st.Column = ColumnDef{Name: colName, NotNull: true}
			default:
				return nil, p.errf("expected DEFAULT or NOT NULL after SET")
			}
		case "DROP":
			switch p.kw() {
			case "DEFAULT":
				p.advance()
				st.Column = ColumnDef{Name: colName}
			case "NOT":
				p.advance()
				if err := p.expectNull(); err != nil {
					return nil, err
				}
				st.Column = ColumnDef{Name: colName, NotNull: false}
			default:
				return nil, p.errf("expected DEFAULT or NOT NULL after DROP")
			}
		case "TYPE":
			if p.cur.typ != tokIdent {
				return nil, p.errf("expected type name after TYPE")
			}
			sqlType, ok := sqlTypeFrom(p.cur.lit)
			if !ok {
				return nil, p.errf("unknown SQL type: " + p.cur.lit)
			}
			p.advance()
			st.Column = ColumnDef{Name: colName, Type: sqlType}
		default:
			return nil, p.errf("unknown ALTER COLUMN action: " + subAction)
		}

	default:
		return nil, p.errf("expected ADD, DROP, or ALTER in ALTER TABLE")
	}

	p.consumeOptionalSemi()
	return st, nil
}
