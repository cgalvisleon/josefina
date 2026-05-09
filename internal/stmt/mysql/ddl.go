package mysql

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ── CREATE TABLE ──────────────────────────────────────────────────────────────

/**
* genCreateTable: renders a CreateTableStmt as MySQL SQL.
* IF NOT EXISTS is supported natively by MySQL — no PL/SQL workaround required.
* BOOLEAN columns become TINYINT(1) with a CHECK constraint (MySQL 8.0.16+).
* @param s stmt.CreateTableStmt
* @return string, error
**/
func genCreateTable(s stmt.CreateTableStmt) (string, error) {
	var lines []string

	for _, col := range s.Columns {
		mt := mysqlType(col.Type)
		def := "  `" + col.Name + "` " + mt

		if col.Default != nil {
			def += " DEFAULT " + formatValue(col.Default)
		}
		if col.NotNull || contains(s.Required, col.Name) {
			def += " NOT NULL"
		}
		if col.Type == stmt.SqlBool {
			def += " CHECK (`" + col.Name + "` IN (0, 1))"
		}
		lines = append(lines, def)
	}

	if len(s.PrimaryKeys) > 0 {
		quoted := make([]string, len(s.PrimaryKeys))
		for i, k := range s.PrimaryKeys {
			quoted[i] = "`" + k + "`"
		}
		lines = append(lines,
			fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(quoted, ", ")))
	}

	for _, u := range s.Unique {
		lines = append(lines,
			fmt.Sprintf("  UNIQUE KEY `uq_%s_%s` (`%s`)", s.Table.Name, u, u))
	}

	createKw := "CREATE TABLE"
	if s.IfNotExists {
		createKw = "CREATE TABLE IF NOT EXISTS"
	}

	return fmt.Sprintf("%s %s (\n%s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		createKw, tableRef(s.Table), strings.Join(lines, ",\n")), nil
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
* genDropTable: renders a DropTableStmt as MySQL SQL.
* IF EXISTS is supported natively by MySQL.
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
* genAlterTable: renders an AlterTableStmt as MySQL SQL.
* ADD COLUMN    → ALTER TABLE t ADD COLUMN col type
* DROP COLUMN   → ALTER TABLE t DROP COLUMN col
* ALTER COLUMN  → ALTER TABLE t MODIFY COLUMN col type
* ADD PRIMARY KEY → ALTER TABLE t ADD PRIMARY KEY (cols)
* ADD UNIQUE    → ALTER TABLE t ADD UNIQUE KEY uq_t_col (col)
* @param s stmt.AlterTableStmt
* @return string, error
**/
func genAlterTable(s stmt.AlterTableStmt) (string, error) {
	tbl := tableRef(s.Table)
	switch s.Action {
	case stmt.AlterAddColumn:
		mt := mysqlType(s.Column.Type)
		colDef := "`" + s.Column.Name + "` " + mt
		if s.Column.Default != nil {
			colDef += " DEFAULT " + formatValue(s.Column.Default)
		}
		if s.Column.NotNull {
			colDef += " NOT NULL"
		}
		return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tbl, colDef), nil

	case stmt.AlterDropColumn:
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN `%s`", tbl, s.ColName), nil

	case stmt.AlterAlterColumn:
		mt := mysqlType(s.Column.Type)
		colDef := "`" + s.Column.Name + "` " + mt
		if s.Column.Default != nil {
			colDef += " DEFAULT " + formatValue(s.Column.Default)
		}
		if s.Column.NotNull {
			colDef += " NOT NULL"
		}
		return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s", tbl, colDef), nil

	case stmt.AlterAddPrimaryKey:
		return fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (`%s`)",
			tbl, s.Column.Name), nil

	case stmt.AlterAddUnique:
		return fmt.Sprintf("ALTER TABLE %s ADD UNIQUE KEY `uq_%s_%s` (`%s`)",
			tbl, s.Table.Name, s.Column.Name, s.Column.Name), nil
	}
	return "", fmt.Errorf("mysql: unsupported ALTER action %v", s.Action)
}
