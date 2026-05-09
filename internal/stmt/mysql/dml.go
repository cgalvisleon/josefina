package mysql

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ── SELECT ────────────────────────────────────────────────────────────────────

/**
* genSelect: renders a SelectStmt as MySQL SQL.
* Pagination uses MySQL's LIMIT n OFFSET m syntax.
* FULL OUTER JOIN is not supported by MySQL and returns an error.
* @param s stmt.SelectStmt
* @return string, error
**/
func genSelect(s stmt.SelectStmt) (string, error) {
	var b strings.Builder

	b.WriteString("SELECT ")
	if len(s.Columns) == 0 {
		b.WriteString("*")
	} else {
		b.WriteString(strings.Join(s.Columns, ", "))
	}

	b.WriteString(" FROM ")
	b.WriteString(tableRef(s.From))

	for _, j := range s.Joins {
		if j.Kind == stmt.JoinFull {
			return "", fmt.Errorf("mysql: FULL OUTER JOIN is not supported; emulate with LEFT JOIN UNION RIGHT JOIN")
		}
		b.WriteString(" ")
		b.WriteString(joinKind(j.Kind))
		b.WriteString(" JOIN ")
		b.WriteString(tableRef(j.Table))
		b.WriteString(" ON ")
		var onParts []string
		for fromCol, toCol := range j.Keys {
			onParts = append(onParts,
				s.From.Name+"."+fromCol+" = "+j.Table.Name+"."+toCol)
		}
		b.WriteString(strings.Join(onParts, " AND "))
	}

	if len(s.Where) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(genConditions(s.Where))
	}

	if len(s.OrderBy) > 0 {
		b.WriteString(" ORDER BY ")
		parts := make([]string, len(s.OrderBy))
		for i, o := range s.OrderBy {
			dir := " ASC"
			if !o.Asc {
				dir = " DESC"
			}
			parts[i] = o.Column + dir
		}
		b.WriteString(strings.Join(parts, ", "))
	}

	// MySQL pagination: LIMIT n [OFFSET m]
	if s.Page > 0 && s.Rows > 0 {
		offset := (s.Page - 1) * s.Rows
		b.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", s.Rows, offset))
	} else if s.Offset > 0 && s.Rows > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", s.Rows, s.Offset))
	} else if s.Rows > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", s.Rows))
	}

	return b.String(), nil
}

// ── INSERT ────────────────────────────────────────────────────────────────────

/**
* genInsert: renders an InsertStmt as MySQL SQL.
* MySQL supports multi-row VALUES (…),(…) natively, so no INSERT ALL workaround
* is needed.
* @param s stmt.InsertStmt
* @return string, error
**/
func genInsert(s stmt.InsertStmt) (string, error) {
	if len(s.Rows) == 0 {
		return "", nil
	}

	cols := getColumns(s.Columns, s.Rows[0])

	valueTuples := make([]string, len(s.Rows))
	for i, row := range s.Rows {
		valueTuples[i] = "(" + rowValues(cols, row) + ")"
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableRef(s.Table),
		strings.Join(cols, ", "),
		strings.Join(valueTuples, ", ")), nil
}

func rowValues(cols []string, row map[string]any) string {
	vals := make([]string, len(cols))
	for i, col := range cols {
		vals[i] = formatValue(row[col])
	}
	return strings.Join(vals, ", ")
}

// ── UPSERT → INSERT … ON DUPLICATE KEY UPDATE ─────────────────────────────────

/**
* genUpsert: renders an UpsertStmt as MySQL INSERT … ON DUPLICATE KEY UPDATE.
* The primary key (_idx) is excluded from the UPDATE clause.
* All rows are batched into one INSERT with multiple VALUES tuples.
* @param s stmt.UpsertStmt
* @return string, error
**/
func genUpsert(s stmt.UpsertStmt) (string, error) {
	if len(s.Rows) == 0 {
		return "", nil
	}

	cols := getColumns(s.Columns, s.Rows[0])

	valueTuples := make([]string, len(s.Rows))
	for i, row := range s.Rows {
		valueTuples[i] = "(" + rowValues(cols, row) + ")"
	}

	// ON DUPLICATE KEY UPDATE: update every column except the key
	var updateParts []string
	for _, col := range cols {
		if col != "_idx" {
			updateParts = append(updateParts,
				fmt.Sprintf("%s = VALUES(%s)", col, col))
		}
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableRef(s.Table),
		strings.Join(cols, ", "),
		strings.Join(valueTuples, ", "))

	if len(updateParts) > 0 {
		sql += "\nON DUPLICATE KEY UPDATE " + strings.Join(updateParts, ", ")
	}

	return sql, nil
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

/**
* genUpdate: renders an UpdateStmt as MySQL SQL.
* @param s stmt.UpdateStmt
* @return string, error
**/
func genUpdate(s stmt.UpdateStmt) (string, error) {
	parts := make([]string, len(s.Assignments))
	for i, a := range s.Assignments {
		parts[i] = a.Column + " = " + formatValue(a.Value)
	}

	sql := "UPDATE " + tableRef(s.Table) + " SET " + strings.Join(parts, ", ")
	if len(s.Where) > 0 {
		sql += " WHERE " + genConditions(s.Where)
	}
	return sql, nil
}

// ── DELETE ────────────────────────────────────────────────────────────────────

/**
* genDelete: renders a DeleteStmt as MySQL SQL.
* @param s stmt.DeleteStmt
* @return string, error
**/
func genDelete(s stmt.DeleteStmt) (string, error) {
	sql := "DELETE FROM " + tableRef(s.Table)
	if len(s.Where) > 0 {
		sql += " WHERE " + genConditions(s.Where)
	}
	return sql, nil
}
