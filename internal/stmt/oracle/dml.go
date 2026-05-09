package oracle

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ── SELECT ────────────────────────────────────────────────────────────────────

/**
* genSelect: renders a SelectStmt as Oracle SQL.
* Pagination uses Oracle 12c+ OFFSET … ROWS FETCH NEXT … ROWS ONLY syntax.
* ILIKE is emulated via UPPER(col) LIKE UPPER(pattern).
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

	// Oracle 12c+ row-limiting clause
	if s.Page > 0 && s.Rows > 0 {
		offset := (s.Page - 1) * s.Rows
		b.WriteString(fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, s.Rows))
	} else if s.Offset > 0 && s.Rows > 0 {
		b.WriteString(fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", s.Offset, s.Rows))
	} else if s.Rows > 0 {
		b.WriteString(fmt.Sprintf(" FETCH FIRST %d ROWS ONLY", s.Rows))
	}

	return b.String(), nil
}

// ── INSERT ────────────────────────────────────────────────────────────────────

/**
* genInsert: renders an InsertStmt as Oracle SQL.
* Single-row inserts use the standard INSERT INTO … VALUES form.
* Multi-row inserts use INSERT ALL … SELECT 1 FROM DUAL, which is compatible
* with all Oracle 12c+ editions.
* @param s stmt.InsertStmt
* @return string, error
**/
func genInsert(s stmt.InsertStmt) (string, error) {
	if len(s.Rows) == 0 {
		return "", nil
	}

	cols := getColumns(s.Columns, s.Rows[0])

	if len(s.Rows) == 1 {
		return singleInsert(tableRef(s.Table), cols, s.Rows[0]), nil
	}

	// Multi-row: INSERT ALL … SELECT 1 FROM DUAL
	var b strings.Builder
	b.WriteString("INSERT ALL\n")
	for _, row := range s.Rows {
		b.WriteString("  INTO ")
		b.WriteString(tableRef(s.Table))
		b.WriteString(" (")
		b.WriteString(strings.Join(cols, ", "))
		b.WriteString(") VALUES (")
		b.WriteString(rowValues(cols, row))
		b.WriteString(")\n")
	}
	b.WriteString("SELECT 1 FROM DUAL")
	return b.String(), nil
}

func singleInsert(tbl string, cols []string, row map[string]any) string {
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tbl, strings.Join(cols, ", "), rowValues(cols, row))
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
* genUpsert: renders an UpsertStmt as an Oracle MERGE statement.
* The merge key is _idx (josefina's universal primary key).
* Multiple rows are handled via UNION ALL inside the USING subquery.
* @param s stmt.UpsertStmt
* @return string, error
**/
func genUpsert(s stmt.UpsertStmt) (string, error) {
	if len(s.Rows) == 0 {
		return "", nil
	}

	cols := getColumns(s.Columns, s.Rows[0])
	tbl := tableRef(s.Table)

	// USING subquery: one row per UNION ALL branch
	sourceParts := make([]string, len(s.Rows))
	for i, row := range s.Rows {
		selParts := make([]string, len(cols))
		for j, col := range cols {
			selParts[j] = fmt.Sprintf("%s AS %s", formatValue(row[col]), col)
		}
		sourceParts[i] = "SELECT " + strings.Join(selParts, ", ") + " FROM DUAL"
	}
	using := strings.Join(sourceParts, "\n  UNION ALL\n  ")

	// SET clause: all columns except the merge key
	var setParts []string
	for _, col := range cols {
		if col != "_idx" {
			setParts = append(setParts, fmt.Sprintf("t.%s = s.%s", col, col))
		}
	}

	// INSERT column list
	insertCols := make([]string, len(cols))
	insertVals := make([]string, len(cols))
	for i, col := range cols {
		insertCols[i] = col
		insertVals[i] = "s." + col
	}

	var b strings.Builder
	b.WriteString("MERGE INTO ")
	b.WriteString(tbl)
	b.WriteString(" t\n")
	b.WriteString("USING (\n  ")
	b.WriteString(using)
	b.WriteString("\n) s\n")
	b.WriteString("ON (t._idx = s._idx)\n")
	if len(setParts) > 0 {
		b.WriteString("WHEN MATCHED THEN UPDATE SET ")
		b.WriteString(strings.Join(setParts, ", "))
		b.WriteString("\n")
	}
	b.WriteString("WHEN NOT MATCHED THEN INSERT (")
	b.WriteString(strings.Join(insertCols, ", "))
	b.WriteString(") VALUES (")
	b.WriteString(strings.Join(insertVals, ", "))
	b.WriteString(")")

	return b.String(), nil
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

/**
* genUpdate: renders an UpdateStmt as Oracle SQL.
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
* genDelete: renders a DeleteStmt as Oracle SQL.
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
