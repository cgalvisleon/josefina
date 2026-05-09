// Package oracle translates parsed stmt.Stmt values into Oracle SQL strings.
// It targets Oracle Database 12c Release 2 and later, where FETCH FIRST /
// OFFSET … ROWS syntax and IDENTITY columns are available.
//
// Entry points:
//
//	oracle.ToSQL(st)       — single statement → string
//	oracle.ToSQLAll(stmts) — slice of statements → []string
package oracle

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ToSQL converts one parsed statement to Oracle SQL.
// TxStmt is rendered as a semicolon-separated block ending with COMMIT or ROLLBACK.
/**
* ToSQL: converts one parsed statement to Oracle SQL.
* @param st stmt.Stmt
* @return string, error
**/
func ToSQL(st stmt.Stmt) (string, error) {
	switch s := st.(type) {
	// ── DML ───────────────────────────────────────────────────────────────────
	case stmt.SelectStmt:
		return genSelect(s)
	case stmt.InsertStmt:
		return genInsert(s)
	case stmt.UpsertStmt:
		return genUpsert(s)
	case stmt.UpdateStmt:
		return genUpdate(s)
	case stmt.DeleteStmt:
		return genDelete(s)
	// ── DDL ───────────────────────────────────────────────────────────────────
	case stmt.CreateTableStmt:
		return genCreateTable(s)
	case stmt.DropTableStmt:
		return genDropTable(s)
	case stmt.AlterTableStmt:
		return genAlterTable(s)
	// ── Transactions ──────────────────────────────────────────────────────────
	case stmt.TxStmt:
		return genTx(s)
	case stmt.BeginStmt:
		return "-- BEGIN (Oracle: transactions start implicitly)", nil
	case stmt.CommitStmt:
		return "COMMIT", nil
	case stmt.RollbackStmt:
		return "ROLLBACK", nil
	default:
		return "", fmt.Errorf("oracle: unsupported statement type %T", st)
	}
}

// ToSQLAll converts a slice of parsed statements to individual Oracle SQL strings.
/**
* ToSQLAll: converts a slice of parsed statements to Oracle SQL strings.
* @param stmts []stmt.Stmt
* @return []string, error
**/
func ToSQLAll(stmts []stmt.Stmt) ([]string, error) {
	out := make([]string, 0, len(stmts))
	for _, st := range stmts {
		sql, err := ToSQL(st)
		if err != nil {
			return nil, err
		}
		out = append(out, sql)
	}
	return out, nil
}

// ── Type mapping ──────────────────────────────────────────────────────────────

// oracleType maps a josefina SqlType to the Oracle data type string.
func oracleType(t stmt.SqlType) string {
	switch t {
	case stmt.SqlText, stmt.SqlVarchar:
		return "VARCHAR2(4000)"
	case stmt.SqlInt, stmt.SqlSmallInt:
		return "NUMBER(10)"
	case stmt.SqlBigInt:
		return "NUMBER(19)"
	case stmt.SqlFloat, stmt.SqlReal, stmt.SqlNumeric:
		return "NUMBER"
	case stmt.SqlBool:
		// Oracle has no BOOLEAN before 23c; use NUMBER(1) with a CHECK.
		return "NUMBER(1)"
	case stmt.SqlJson, stmt.SqlJsonb:
		return "CLOB"
	case stmt.SqlTimestamp, stmt.SqlDate:
		return "TIMESTAMP"
	case stmt.SqlBytes:
		return "BLOB"
	case stmt.SqlKey:
		return "VARCHAR2(255)"
	default:
		return "VARCHAR2(255)"
	}
}

// ── Identifier helpers ────────────────────────────────────────────────────────

// tableRef formats a TableRef as [schema.]name.
func tableRef(t stmt.TableRef) string {
	if t.Schema != "" {
		return t.Schema + "." + t.Name
	}
	return t.Name
}

// joinKind maps a JoinKind constant to the Oracle JOIN keyword.
func joinKind(k stmt.JoinKind) string {
	switch k {
	case stmt.JoinLeft:
		return "LEFT OUTER"
	case stmt.JoinRight:
		return "RIGHT OUTER"
	case stmt.JoinFull:
		return "FULL OUTER"
	default:
		return "INNER"
	}
}

// ── Value helpers ─────────────────────────────────────────────────────────────

// formatValue renders a Go value as an Oracle SQL literal.
// Booleans become 1/0 because Oracle has no native BOOLEAN type before 23c.
func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("'%v'", val)
	}
}

// formatValues renders a slice of values as a comma-separated Oracle literal list.
func formatValues(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = formatValue(v)
	}
	return strings.Join(parts, ", ")
}

// ── Column helpers ────────────────────────────────────────────────────────────

// getColumns returns the column list to use for a row.
// When the parser provided an explicit list it is used as-is; otherwise the
// map keys are sorted to guarantee deterministic output.
func getColumns(declared []string, row map[string]any) []string {
	if len(declared) > 0 {
		return declared
	}
	cols := make([]string, 0, len(row))
	for col := range row {
		cols = append(cols, col)
	}
	sort.Strings(cols)
	return cols
}

// ── WHERE condition generator ─────────────────────────────────────────────────

// genConditions renders a slice of CondExpr as an Oracle WHERE expression.
// ILIKE is emulated with UPPER(col) LIKE UPPER(pattern).
func genConditions(conds []stmt.CondExpr) string {
	var parts []string
	for i, c := range conds {
		expr := condExpr(c)
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

func condExpr(c stmt.CondExpr) string {
	switch c.Op {
	case stmt.OpEq:
		return c.Field + " = " + formatValue(c.Value)
	case stmt.OpNeq:
		return c.Field + " != " + formatValue(c.Value)
	case stmt.OpLt:
		return c.Field + " < " + formatValue(c.Value)
	case stmt.OpLtEq:
		return c.Field + " <= " + formatValue(c.Value)
	case stmt.OpGt:
		return c.Field + " > " + formatValue(c.Value)
	case stmt.OpGtEq:
		return c.Field + " >= " + formatValue(c.Value)
	case stmt.OpLike:
		return c.Field + " LIKE " + formatValue(c.Value)
	case stmt.OpILike:
		return "UPPER(" + c.Field + ") LIKE UPPER(" + formatValue(c.Value) + ")"
	case stmt.OpIn:
		return c.Field + " IN (" + formatValues(toSlice(c.Value)) + ")"
	case stmt.OpNotIn:
		if vals, ok := c.Value.([]any); ok {
			return c.Field + " NOT IN (" + formatValues(vals) + ")"
		}
		// NOT LIKE fallback (parser maps NOT LIKE to OpNotIn for single values)
		return c.Field + " NOT LIKE " + formatValue(c.Value)
	case stmt.OpIsNull:
		return c.Field + " IS NULL"
	case stmt.OpIsNotNull:
		return c.Field + " IS NOT NULL"
	case stmt.OpBetween:
		b := c.Value.([2]any)
		return c.Field + " BETWEEN " + formatValue(b[0]) + " AND " + formatValue(b[1])
	case stmt.OpNotBetween:
		b := c.Value.([2]any)
		return c.Field + " NOT BETWEEN " + formatValue(b[0]) + " AND " + formatValue(b[1])
	default:
		return c.Field + " = " + formatValue(c.Value)
	}
}

func toSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return []any{v}
}
