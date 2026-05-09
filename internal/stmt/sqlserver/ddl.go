package sqlserver

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ── CREATE TABLE ──────────────────────────────────────────────────────────────

/**
* genCreateTable: renders a CreateTableStmt as T-SQL.
* IF NOT EXISTS is emulated with IF OBJECT_ID(N'…', N'U') IS NULL BEGIN … END
* because SQL Server does not support CREATE TABLE IF NOT EXISTS syntax.
* BOOLEAN columns become BIT; no CHECK constraint needed (BIT only stores 0/1).
* @param s stmt.CreateTableStmt
* @return string, error
**/
func genCreateTable(s stmt.CreateTableStmt) (string, error) {
	inner := buildCreateTable(s)
	if !s.IfNotExists {
		return inner, nil
	}
	tblRef := tableRef(s.Table)
	indented := "  " + strings.ReplaceAll(inner, "\n", "\n  ")
	return fmt.Sprintf(
		"IF OBJECT_ID(N'%s', N'U') IS NULL\nBEGIN\n%s\nEND",
		tblRef, indented), nil
}

func buildCreateTable(s stmt.CreateTableStmt) string {
	var lines []string

	for _, col := range s.Columns {
		st := sqlserverType(col.Type)
		def := "  " + col.Name + " " + st

		if col.Default != nil {
			def += " DEFAULT " + formatValue(col.Default)
		}
		if col.NotNull || contains(s.Required, col.Name) {
			def += " NOT NULL"
		}
		lines = append(lines, def)
	}

	if len(s.PrimaryKeys) > 0 {
		lines = append(lines,
			fmt.Sprintf("  CONSTRAINT pk_%s PRIMARY KEY (%s)",
				s.Table.Name, strings.Join(s.PrimaryKeys, ", ")))
	}

	for _, u := range s.Unique {
		lines = append(lines,
			fmt.Sprintf("  CONSTRAINT uq_%s_%s UNIQUE (%s)", s.Table.Name, u, u))
	}

	return fmt.Sprintf("CREATE TABLE %s (\n%s\n)",
		tableRef(s.Table), strings.Join(lines, ",\n"))
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ── DROP TABLE ────────────────────────────────────────────────────────────────

/**
* genDropTable: renders a DropTableStmt as T-SQL.
* DROP TABLE IF EXISTS is supported natively in SQL Server 2016+.
* @param s stmt.DropTableStmt
* @return string, error
**/
func genDropTable(s stmt.DropTableStmt) (string, error) {
	if s.IfExists {
		return "DROP TABLE IF EXISTS " + tableRef(s.Table), nil
	}
	return "DROP TABLE " + tableRef(s.Table), nil
}

// ── ALTER TABLE ───────────────────────────────────────────────────────────────

/**
* genAlterTable: renders an AlterTableStmt as T-SQL.
* ADD COLUMN    → ALTER TABLE t ADD col type
* DROP COLUMN   → ALTER TABLE t DROP COLUMN col
* ALTER COLUMN  → ALTER TABLE t ALTER COLUMN col type
* ADD PRIMARY KEY → ALTER TABLE t ADD CONSTRAINT pk_t PRIMARY KEY (col)
* ADD UNIQUE    → ALTER TABLE t ADD CONSTRAINT uq_t_col UNIQUE (col)
* @param s stmt.AlterTableStmt
* @return string, error
**/
func genAlterTable(s stmt.AlterTableStmt) (string, error) {
	tbl := tableRef(s.Table)
	switch s.Action {
	case stmt.AlterAddColumn:
		st := sqlserverType(s.Column.Type)
		colDef := s.Column.Name + " " + st
		if s.Column.Default != nil {
			colDef += " DEFAULT " + formatValue(s.Column.Default)
		}
		if s.Column.NotNull {
			colDef += " NOT NULL"
		}
		return fmt.Sprintf("ALTER TABLE %s ADD %s", tbl, colDef), nil

	case stmt.AlterDropColumn:
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tbl, s.ColName), nil

	case stmt.AlterAlterColumn:
		// SQL Server uses ALTER COLUMN (not MODIFY)
		st := sqlserverType(s.Column.Type)
		colDef := s.Column.Name + " " + st
		if s.Column.NotNull {
			colDef += " NOT NULL"
		}
		return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s", tbl, colDef), nil

	case stmt.AlterAddPrimaryKey:
		return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT pk_%s PRIMARY KEY (%s)",
			tbl, s.Table.Name, s.Column.Name), nil

	case stmt.AlterAddUnique:
		return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT uq_%s_%s UNIQUE (%s)",
			tbl, s.Table.Name, s.Column.Name, s.Column.Name), nil
	}
	return "", fmt.Errorf("sqlserver: unsupported ALTER action %v", s.Action)
}
