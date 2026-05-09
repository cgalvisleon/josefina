package oracle

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ── CREATE TABLE ──────────────────────────────────────────────────────────────

/**
* genCreateTable: renders a CreateTableStmt as Oracle SQL.
* IF NOT EXISTS is emulated with a PL/SQL EXECUTE IMMEDIATE block that swallows
* ORA-00955 (name already used by an existing object).
* BOOLEAN columns become NUMBER(1); a CHECK constraint is appended automatically.
* @param s stmt.CreateTableStmt
* @return string, error
**/
func genCreateTable(s stmt.CreateTableStmt) (string, error) {
	inner := buildCreateTable(s)
	if !s.IfNotExists {
		return inner, nil
	}
	// Escape single quotes inside the EXECUTE IMMEDIATE string
	escaped := strings.ReplaceAll(inner, "'", "''")
	var b strings.Builder
	b.WriteString("BEGIN\n")
	b.WriteString("  EXECUTE IMMEDIATE '")
	b.WriteString(escaped)
	b.WriteString("';\n")
	b.WriteString("EXCEPTION\n")
	b.WriteString("  WHEN OTHERS THEN\n")
	b.WriteString("    IF SQLCODE != -955 THEN RAISE; END IF;\n")
	b.WriteString("END")
	return b.String(), nil
}

func buildCreateTable(s stmt.CreateTableStmt) string {
	var lines []string
	var checks []string

	for _, col := range s.Columns {
		ot := oracleType(col.Type)
		def := "  " + col.Name + " " + ot

		if col.Default != nil {
			def += " DEFAULT " + formatValue(col.Default)
		}
		if col.NotNull || contains(s.Required, col.Name) {
			def += " NOT NULL"
		}
		lines = append(lines, def)

		// BOOLEAN → NUMBER(1) check constraint
		if col.Type == stmt.SqlBool {
			checks = append(checks,
				fmt.Sprintf("  CONSTRAINT chk_%s_%s CHECK (%s IN (0, 1))",
					s.Table.Name, col.Name, col.Name))
		}
	}

	// Table-level PRIMARY KEY
	if len(s.PrimaryKeys) > 0 {
		lines = append(lines,
			fmt.Sprintf("  CONSTRAINT pk_%s PRIMARY KEY (%s)",
				s.Table.Name, strings.Join(s.PrimaryKeys, ", ")))
	}

	// Table-level UNIQUE
	for _, u := range s.Unique {
		lines = append(lines,
			fmt.Sprintf("  CONSTRAINT uq_%s_%s UNIQUE (%s)", s.Table.Name, u, u))
	}

	lines = append(lines, checks...)

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
* genDropTable: renders a DropTableStmt as Oracle SQL.
* IF EXISTS is emulated with a PL/SQL block that swallows ORA-00942
* (table or view does not exist).
* @param s stmt.DropTableStmt
* @return string, error
**/
func genDropTable(s stmt.DropTableStmt) (string, error) {
	sql := "DROP TABLE " + tableRef(s.Table)
	if !s.IfExists {
		return sql, nil
	}
	var b strings.Builder
	b.WriteString("BEGIN\n")
	b.WriteString("  EXECUTE IMMEDIATE '")
	b.WriteString(sql)
	b.WriteString("';\n")
	b.WriteString("EXCEPTION\n")
	b.WriteString("  WHEN OTHERS THEN\n")
	b.WriteString("    IF SQLCODE != -942 THEN RAISE; END IF;\n")
	b.WriteString("END")
	return b.String(), nil
}

// ── ALTER TABLE ───────────────────────────────────────────────────────────────

/**
* genAlterTable: renders an AlterTableStmt as Oracle SQL.
* ADD COLUMN  → ALTER TABLE t ADD (col type)
* DROP COLUMN → ALTER TABLE t DROP COLUMN col
* ALTER COLUMN → ALTER TABLE t MODIFY (col type)
* ADD PRIMARY KEY → ALTER TABLE t ADD CONSTRAINT pk_t PRIMARY KEY (cols)
* ADD UNIQUE → ALTER TABLE t ADD CONSTRAINT uq_t_col UNIQUE (col)
* @param s stmt.AlterTableStmt
* @return string, error
**/
func genAlterTable(s stmt.AlterTableStmt) (string, error) {
	tbl := tableRef(s.Table)
	switch s.Action {
	case stmt.AlterAddColumn:
		ot := oracleType(s.Column.Type)
		colDef := s.Column.Name + " " + ot
		if s.Column.Default != nil {
			colDef += " DEFAULT " + formatValue(s.Column.Default)
		}
		if s.Column.NotNull {
			colDef += " NOT NULL"
		}
		return fmt.Sprintf("ALTER TABLE %s ADD (%s)", tbl, colDef), nil

	case stmt.AlterDropColumn:
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tbl, s.ColName), nil

	case stmt.AlterAlterColumn:
		ot := oracleType(s.Column.Type)
		colDef := s.Column.Name + " " + ot
		if s.Column.Default != nil {
			colDef += " DEFAULT " + formatValue(s.Column.Default)
		}
		if s.Column.NotNull {
			colDef += " NOT NULL"
		}
		return fmt.Sprintf("ALTER TABLE %s MODIFY (%s)", tbl, colDef), nil

	case stmt.AlterAddPrimaryKey:
		return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT pk_%s PRIMARY KEY (%s)",
			tbl, s.Table.Name, s.Column.Name), nil

	case stmt.AlterAddUnique:
		return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT uq_%s_%s UNIQUE (%s)",
			tbl, s.Table.Name, s.Column.Name, s.Column.Name), nil
	}
	return "", fmt.Errorf("oracle: unsupported ALTER action %v", s.Action)
}
