package jsql

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
	"github.com/cgalvisleon/josefina/internal/stmt/mysql"
	"github.com/cgalvisleon/josefina/internal/stmt/oracle"
	"github.com/cgalvisleon/josefina/internal/stmt/sqlserver"
)

/**
* ToDialect: translates a slice of parsed statements to SQL strings in the
* requested dialect. JOSEFINA and POSTGRESQL produce PostgreSQL-compatible SQL.
* MYSQL, ORACLE, and SQLSERVER use their respective dialect generators.
* @param stmts []stmt.Stmt
* @param dialect stmt.SqlDialect
* @return []string, error
**/
func ToDialect(stmts []stmt.Stmt, dialect stmt.SqlDialect) ([]string, error) {
	switch dialect {
	case stmt.DialectMySQL:
		return mysql.ToSQLAll(stmts)
	case stmt.DialectOracle:
		return oracle.ToSQLAll(stmts)
	case stmt.DialectSQLServer:
		return sqlserver.ToSQLAll(stmts)
	case stmt.DialectJosefina, stmt.DialectPostgreSQL:
		return toPostgres(stmts)
	default:
		return nil, fmt.Errorf("unknown dialect %q", dialect)
	}
}

// toPostgres renders statements as PostgreSQL-compatible SQL strings.
func toPostgres(stmts []stmt.Stmt) ([]string, error) {
	out := make([]string, 0, len(stmts))
	for _, st := range stmts {
		sql, err := stmtToPostgres(st)
		if err != nil {
			return nil, err
		}
		out = append(out, sql)
	}
	return out, nil
}

func stmtToPostgres(st stmt.Stmt) (string, error) {
	switch s := st.(type) {
	case stmt.SelectStmt:
		return pgSelect(s), nil
	case stmt.InsertStmt:
		return pgInsert(s), nil
	case stmt.UpdateStmt:
		return pgUpdate(s), nil
	case stmt.DeleteStmt:
		return pgDelete(s), nil
	case stmt.UpsertStmt:
		return pgUpsert(s), nil
	case stmt.CreateTableStmt:
		return pgCreateTable(s), nil
	case stmt.DropTableStmt:
		return pgDropTable(s), nil
	case stmt.BeginStmt:
		return "BEGIN", nil
	case stmt.CommitStmt:
		return "COMMIT", nil
	case stmt.RollbackStmt:
		return "ROLLBACK", nil
	case stmt.SetSqlStateStmt:
		return fmt.Sprintf("SET SQL STATE %s", s.Dialect), nil
	default:
		return fmt.Sprintf("-- unsupported: %T", st), nil
	}
}

func pgRef(t stmt.TableRef) string {
	if t.Schema != "" {
		return t.Schema + "." + t.Name
	}
	return t.Name
}

func pgVal(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func pgVals(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = pgVal(v)
	}
	return strings.Join(parts, ", ")
}

func pgSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return []any{v}
}

func pgConds(conds []stmt.CondExpr) string {
	var parts []string
	for i, c := range conds {
		expr := pgCond(c)
		if i == 0 {
			parts = append(parts, expr)
		} else if c.Connector == stmt.ConnOr {
			parts = append(parts, "OR "+expr)
		} else {
			parts = append(parts, "AND "+expr)
		}
	}
	return strings.Join(parts, " ")
}

func pgCond(c stmt.CondExpr) string {
	switch c.Op {
	case stmt.OpEq:
		return c.Field + " = " + pgVal(c.Value)
	case stmt.OpNeq:
		return c.Field + " != " + pgVal(c.Value)
	case stmt.OpLt:
		return c.Field + " < " + pgVal(c.Value)
	case stmt.OpLtEq:
		return c.Field + " <= " + pgVal(c.Value)
	case stmt.OpGt:
		return c.Field + " > " + pgVal(c.Value)
	case stmt.OpGtEq:
		return c.Field + " >= " + pgVal(c.Value)
	case stmt.OpLike:
		return c.Field + " LIKE " + pgVal(c.Value)
	case stmt.OpILike:
		return c.Field + " ILIKE " + pgVal(c.Value)
	case stmt.OpIn:
		return c.Field + " IN (" + pgVals(pgSlice(c.Value)) + ")"
	case stmt.OpNotIn:
		return c.Field + " NOT IN (" + pgVals(pgSlice(c.Value)) + ")"
	case stmt.OpIsNull:
		return c.Field + " IS NULL"
	case stmt.OpIsNotNull:
		return c.Field + " IS NOT NULL"
	case stmt.OpBetween:
		b := c.Value.([2]any)
		return c.Field + " BETWEEN " + pgVal(b[0]) + " AND " + pgVal(b[1])
	case stmt.OpNotBetween:
		b := c.Value.([2]any)
		return c.Field + " NOT BETWEEN " + pgVal(b[0]) + " AND " + pgVal(b[1])
	default:
		return c.Field + " = " + pgVal(c.Value)
	}
}

func pgSelect(s stmt.SelectStmt) string {
	cols := "*"
	if len(s.Columns) > 0 {
		cols = strings.Join(s.Columns, ", ")
	}
	sql := "SELECT " + cols + " FROM " + pgRef(s.From)
	if len(s.Where) > 0 {
		sql += " WHERE " + pgConds(s.Where)
	}
	if len(s.OrderBy) > 0 {
		parts := make([]string, len(s.OrderBy))
		for i, o := range s.OrderBy {
			dir := " ASC"
			if !o.Asc {
				dir = " DESC"
			}
			parts[i] = o.Column + dir
		}
		sql += " ORDER BY " + strings.Join(parts, ", ")
	}
	if s.Rows > 0 {
		sql += fmt.Sprintf(" LIMIT %d", s.Rows)
	}
	if s.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", s.Offset)
	} else if s.Page > 0 && s.Rows > 0 {
		sql += fmt.Sprintf(" OFFSET %d", (s.Page-1)*s.Rows)
	}
	return sql
}

func pgInsert(s stmt.InsertStmt) string {
	if len(s.Rows) == 0 {
		return ""
	}
	cols := s.Columns
	if len(cols) == 0 {
		for k := range s.Rows[0] {
			cols = append(cols, k)
		}
	}
	tuples := make([]string, len(s.Rows))
	for i, row := range s.Rows {
		vals := make([]string, len(cols))
		for j, c := range cols {
			vals[j] = pgVal(row[c])
		}
		tuples[i] = "(" + strings.Join(vals, ", ") + ")"
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		pgRef(s.Table), strings.Join(cols, ", "), strings.Join(tuples, ", "))
}

func pgUpsert(s stmt.UpsertStmt) string {
	if len(s.Rows) == 0 {
		return ""
	}
	cols := s.Columns
	if len(cols) == 0 {
		for k := range s.Rows[0] {
			cols = append(cols, k)
		}
	}
	tuples := make([]string, len(s.Rows))
	for i, row := range s.Rows {
		vals := make([]string, len(cols))
		for j, c := range cols {
			vals[j] = pgVal(row[c])
		}
		tuples[i] = "(" + strings.Join(vals, ", ") + ")"
	}
	var setParts []string
	for _, col := range cols {
		if col != "_idx" {
			setParts = append(setParts, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s ON CONFLICT (_idx) DO UPDATE SET %s",
		pgRef(s.Table), strings.Join(cols, ", "), strings.Join(tuples, ", "), strings.Join(setParts, ", "))
}

func pgUpdate(s stmt.UpdateStmt) string {
	parts := make([]string, len(s.Assignments))
	for i, a := range s.Assignments {
		parts[i] = a.Column + " = " + pgVal(a.Value)
	}
	sql := "UPDATE " + pgRef(s.Table) + " SET " + strings.Join(parts, ", ")
	if len(s.Where) > 0 {
		sql += " WHERE " + pgConds(s.Where)
	}
	return sql
}

func pgDelete(s stmt.DeleteStmt) string {
	sql := "DELETE FROM " + pgRef(s.Table)
	if len(s.Where) > 0 {
		sql += " WHERE " + pgConds(s.Where)
	}
	return sql
}

func pgCreateTable(s stmt.CreateTableStmt) string {
	var lines []string
	for _, col := range s.Columns {
		def := "  " + col.Name + " " + pgColType(col.Type)
		if col.Default != nil {
			def += " DEFAULT " + pgVal(col.Default)
		}
		if col.NotNull {
			def += " NOT NULL"
		}
		lines = append(lines, def)
	}
	if len(s.PrimaryKeys) > 0 {
		lines = append(lines, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(s.PrimaryKeys, ", ")))
	}
	prefix := "CREATE TABLE "
	if s.IfNotExists {
		prefix = "CREATE TABLE IF NOT EXISTS "
	}
	return prefix + pgRef(s.Table) + " (\n" + strings.Join(lines, ",\n") + "\n)"
}

func pgDropTable(s stmt.DropTableStmt) string {
	if s.IfExists {
		return "DROP TABLE IF EXISTS " + pgRef(s.Table)
	}
	return "DROP TABLE " + pgRef(s.Table)
}

func pgColType(t stmt.SqlType) string {
	switch t {
	case stmt.SqlText, stmt.SqlVarchar:
		return "TEXT"
	case stmt.SqlInt, stmt.SqlSmallInt:
		return "INTEGER"
	case stmt.SqlBigInt:
		return "BIGINT"
	case stmt.SqlFloat, stmt.SqlReal:
		return "REAL"
	case stmt.SqlNumeric:
		return "NUMERIC"
	case stmt.SqlBool:
		return "BOOLEAN"
	case stmt.SqlJson:
		return "JSON"
	case stmt.SqlJsonb:
		return "JSONB"
	case stmt.SqlTimestamp:
		return "TIMESTAMP"
	case stmt.SqlDate:
		return "DATE"
	case stmt.SqlBytes:
		return "BYTEA"
	case stmt.SqlKey:
		return "VARCHAR(255)"
	default:
		return "TEXT"
	}
}
