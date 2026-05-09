// Package mysql translates parsed stmt.Stmt values into MySQL 8.0+ SQL strings.
//
// Notable dialect differences from standard SQL:
//   - LIMIT / OFFSET instead of FETCH FIRST / OFFSET … ROWS
//   - INSERT … ON DUPLICATE KEY UPDATE instead of MERGE / UPSERT
//   - Multi-row VALUES (…),(…) supported natively
//   - IF NOT EXISTS / IF EXISTS supported natively (no PL/SQL workaround needed)
//   - TRUE / FALSE literals preserved (TINYINT(1) storage)
//   - ILIKE emulated via LOWER(col) LIKE LOWER(pattern)
//   - FULL OUTER JOIN not supported; returns an error
//   - Transactions use START TRANSACTION / COMMIT / ROLLBACK
//
// Entry points:
//
//	mysql.ToSQL(st)       — single statement → string
//	mysql.ToSQLAll(stmts) — slice of statements → []string
package mysql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ToSQL converts one parsed statement to MySQL SQL.
/**
* ToSQL: converts one parsed statement to MySQL SQL.
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
		return "START TRANSACTION", nil
	case stmt.CommitStmt:
		return "COMMIT", nil
	case stmt.RollbackStmt:
		return "ROLLBACK", nil
	default:
		return "", fmt.Errorf("mysql: unsupported statement type %T", st)
	}
}

// ToSQLAll converts a slice of parsed statements to individual MySQL SQL strings.
/**
* ToSQLAll: converts a slice of parsed statements to MySQL SQL strings.
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

// mysqlType maps a josefina SqlType to the MySQL data type string.
func mysqlType(t stmt.SqlType) string {
	switch t {
	case stmt.SqlText, stmt.SqlVarchar:
		return "TEXT"
	case stmt.SqlInt, stmt.SqlSmallInt:
		return "INT"
	case stmt.SqlBigInt:
		return "BIGINT"
	case stmt.SqlFloat, stmt.SqlReal:
		return "FLOAT"
	case stmt.SqlNumeric:
		return "DECIMAL"
	case stmt.SqlBool:
		// MySQL stores BOOLEAN as TINYINT(1).
		return "TINYINT(1)"
	case stmt.SqlJson, stmt.SqlJsonb:
		return "JSON"
	case stmt.SqlTimestamp, stmt.SqlDate:
		return "DATETIME"
	case stmt.SqlBytes:
		return "BLOB"
	case stmt.SqlKey:
		return "VARCHAR(255)"
	default:
		return "VARCHAR(255)"
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

// joinKind maps a JoinKind to the MySQL JOIN keyword prefix.
// FULL OUTER JOIN is not supported in MySQL and must be checked by callers.
func joinKind(k stmt.JoinKind) string {
	switch k {
	case stmt.JoinLeft:
		return "LEFT OUTER"
	case stmt.JoinRight:
		return "RIGHT OUTER"
	default:
		return "INNER"
	}
}

// ── Value helpers ─────────────────────────────────────────────────────────────

// formatValue renders a Go value as a MySQL SQL literal.
// MySQL supports TRUE/FALSE natively so booleans are preserved as-is.
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
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("'%v'", val)
	}
}

// formatValues renders a slice of values as a comma-separated MySQL literal list.
func formatValues(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = formatValue(v)
	}
	return strings.Join(parts, ", ")
}

// ── Column helpers ────────────────────────────────────────────────────────────

// getColumns returns the column list for a row.
// When the parser provided an explicit list it is used as-is; otherwise map
// keys are sorted for deterministic output.
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

// genConditions renders a slice of CondExpr as a MySQL WHERE expression.
// ILIKE is emulated with LOWER(col) LIKE LOWER(pattern).
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
		// MySQL LIKE is case-insensitive for most collations, but we emit
		// an explicit LOWER() to be safe with binary/accent-sensitive collations.
		return "LOWER(" + c.Field + ") LIKE LOWER(" + formatValue(c.Value) + ")"
	case stmt.OpIn:
		return c.Field + " IN (" + formatValues(toSlice(c.Value)) + ")"
	case stmt.OpNotIn:
		if vals, ok := c.Value.([]any); ok {
			return c.Field + " NOT IN (" + formatValues(vals) + ")"
		}
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
