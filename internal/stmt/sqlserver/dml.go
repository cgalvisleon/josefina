package sqlserver

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ── SELECT ────────────────────────────────────────────────────────────────────

/**
* genSelect: renders a SelectStmt as T-SQL.
* Pagination strategy:
*   LIMIT only           → SELECT TOP (n) …
*   LIMIT + OFFSET/PAGE  → … ORDER BY … OFFSET m ROWS FETCH NEXT n ROWS ONLY
*                          (ORDER BY (SELECT NULL) is appended when no ORDER BY is present)
* @param s stmt.SelectStmt
* @return string, error
**/
func genSelect(s stmt.SelectStmt) (string, error) {
	var b strings.Builder

	b.WriteString("SELECT ")

	// TOP is used only when LIMIT is given without any offset/page.
	useTop := s.Rows > 0 && s.Offset == 0 && s.Page == 0
	useOffsetFetch := s.Rows > 0 && (s.Offset > 0 || s.Page > 0)

	if useTop {
		b.WriteString(fmt.Sprintf("TOP (%d) ", s.Rows))
	}

	if len(s.Columns) == 0 {
		b.WriteString("*")
	} else {
		b.WriteString(strings.Join(s.Columns, ", "))
	}

	b.WriteString(" FROM ")
	b.WriteString(tableRef(s.From))

	for _, j := range s.Joins {
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

	hasOrderBy := len(s.OrderBy) > 0
	if hasOrderBy {
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

	if useOffsetFetch {
		// OFFSET/FETCH requires ORDER BY in SQL Server.
		if !hasOrderBy {
			b.WriteString(" ORDER BY (SELECT NULL)")
		}
		if s.Page > 0 && s.Rows > 0 {
			offset := (s.Page - 1) * s.Rows
			b.WriteString(fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, s.Rows))
		} else {
			b.WriteString(fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", s.Offset, s.Rows))
		}
	}

	return b.String(), nil
}

// ── INSERT ────────────────────────────────────────────────────────────────────

/**
* genInsert: renders an InsertStmt as T-SQL.
* SQL Server supports multi-row VALUES (…),(…) natively.
* @param s stmt.InsertStmt
* @return string, error
**/
func genInsert(s stmt.InsertStmt) (string, error) {
	if len(s.Rows) == 0 {
		return "", nil
	}

	cols := getColumns(s.Columns, s.Rows[0])

	tuples := make([]string, len(s.Rows))
	for i, row := range s.Rows {
		tuples[i] = "(" + rowValues(cols, row) + ")"
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableRef(s.Table),
		strings.Join(cols, ", "),
		strings.Join(tuples, ", ")), nil
}

func rowValues(cols []string, row map[string]any) string {
	vals := make([]string, len(cols))
	for i, col := range cols {
		vals[i] = formatValue(row[col])
	}
	return strings.Join(vals, ", ")
}

// ── UPSERT → MERGE ────────────────────────────────────────────────────────────

/**
* genUpsert: renders an UpsertStmt as a SQL Server MERGE statement.
* SQL Server MERGE uses a VALUES-based USING clause — no FROM DUAL or UNION ALL needed.
* The merge key is _idx (josefina's universal primary key).
* Note: T-SQL requires a semicolon to terminate MERGE statements.
* @param s stmt.UpsertStmt
* @return string, error
**/
func genUpsert(s stmt.UpsertStmt) (string, error) {
	if len(s.Rows) == 0 {
		return "", nil
	}

	cols := getColumns(s.Columns, s.Rows[0])
	tbl := tableRef(s.Table)

	// USING (VALUES (…),(…)) AS s (col1, col2, …)
	tuples := make([]string, len(s.Rows))
	for i, row := range s.Rows {
		tuples[i] = "(" + rowValues(cols, row) + ")"
	}
	sourceAlias := strings.Join(cols, ", ")

	// SET clause: all columns except the merge key
	var setParts []string
	for _, col := range cols {
		if col != "_idx" {
			setParts = append(setParts, fmt.Sprintf("t.%s = s.%s", col, col))
		}
	}

	// INSERT column + value lists
	insertCols := make([]string, len(cols))
	insertVals := make([]string, len(cols))
	for i, col := range cols {
		insertCols[i] = col
		insertVals[i] = "s." + col
	}

	var b strings.Builder
	b.WriteString("MERGE INTO ")
	b.WriteString(tbl)
	b.WriteString(" AS t\n")
	b.WriteString("USING (VALUES\n  ")
	b.WriteString(strings.Join(tuples, ",\n  "))
	b.WriteString("\n) AS s (")
	b.WriteString(sourceAlias)
	b.WriteString(")\n")
	b.WriteString("ON t._idx = s._idx\n")
	if len(setParts) > 0 {
		b.WriteString("WHEN MATCHED THEN\n")
		b.WriteString("  UPDATE SET ")
		b.WriteString(strings.Join(setParts, ", "))
		b.WriteString("\n")
	}
	b.WriteString("WHEN NOT MATCHED THEN\n")
	b.WriteString("  INSERT (")
	b.WriteString(strings.Join(insertCols, ", "))
	b.WriteString(") VALUES (")
	b.WriteString(strings.Join(insertVals, ", "))
	b.WriteString(");") // T-SQL requires semicolon after MERGE

	return b.String(), nil
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

/**
* genUpdate: renders an UpdateStmt as T-SQL.
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
* genDelete: renders a DeleteStmt as T-SQL.
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
