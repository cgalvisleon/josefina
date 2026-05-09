package stmt

import (
	"fmt"
	"strings"
)

/**
* parseSetSqlState: parses SET SQL STATE <dialect>.
* Valid dialects: JOSEFINA, POSTGRESQL, MYSQL, ORACLE, SQLSERVER
* @return SetSqlStateStmt, error
**/
func (p *Parser) parseSetSqlState() (Stmt, error) {
	kw, err := p.parseKeyword()
	if err != nil {
		return nil, err
	}

	var dialect SqlDialect
	switch strings.ToUpper(kw) {
	case "JOSEFINA":
		dialect = DialectJosefina
	case "POSTGRESQL":
		dialect = DialectPostgreSQL
	case "MYSQL":
		dialect = DialectMySQL
	case "ORACLE":
		dialect = DialectOracle
	case "SQLSERVER":
		dialect = DialectSQLServer
	default:
		return nil, fmt.Errorf("unknown SQL dialect %q; valid values: JOSEFINA, POSTGRESQL, MYSQL, ORACLE, SQLSERVER", kw)
	}

	p.consumeOptionalSemi()
	return SetSqlStateStmt{Dialect: dialect}, nil
}
