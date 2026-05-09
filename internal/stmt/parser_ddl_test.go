package stmt

import (
	"strings"
	"testing"
)

// ── CREATE TABLE ──────────────────────────────────────────────────────────────

func TestCreateTable_Basic(t *testing.T) {
	sql := `CREATE TABLE users (
		username  KEY        NOT NULL,
		email     TEXT       NOT NULL,
		age       INT        DEFAULT 0,
		active    BOOLEAN    DEFAULT TRUE
	)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if s.Table.Name != "users" {
		t.Errorf("unexpected table name: %q", s.Table.Name)
	}
	if len(s.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(s.Columns))
	}

	byName := map[string]ColumnDef{}
	for _, c := range s.Columns {
		byName[c.Name] = c
	}

	if byName["username"].Type != SqlKey {
		t.Errorf("username: expected KEY, got %q", byName["username"].Type)
	}
	if !byName["email"].NotNull {
		t.Error("email: expected NOT NULL")
	}
	if byName["age"].Default.(int64) != 0 {
		t.Errorf("age: expected DEFAULT 0, got %v", byName["age"].Default)
	}
	if byName["active"].Default.(bool) != true {
		t.Errorf("active: expected DEFAULT TRUE")
	}
	// NOT NULL columns collected into Required
	if len(s.Required) != 2 {
		t.Errorf("expected 2 required, got %v", s.Required)
	}
}

func TestCreateTable_SchemaQualified(t *testing.T) {
	sql := `CREATE TABLE apps.orders (product TEXT, amount FLOAT)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if s.Table.Schema != "apps" || s.Table.Name != "orders" {
		t.Errorf("unexpected table ref: %+v", s.Table)
	}
}

func TestCreateTable_IfNotExists(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS logs (msg TEXT)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if !s.IfNotExists {
		t.Error("expected IF NOT EXISTS flag")
	}
}

func TestCreateTable_InlinePrimaryKey(t *testing.T) {
	sql := `CREATE TABLE users (id KEY PRIMARY KEY, name TEXT)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if len(s.PrimaryKeys) != 1 || s.PrimaryKeys[0] != "id" {
		t.Errorf("expected PK [id], got %v", s.PrimaryKeys)
	}
}

func TestCreateTable_TableLevelPrimaryKey(t *testing.T) {
	sql := `CREATE TABLE orders (
		order_id KEY,
		user_id  KEY,
		product  TEXT,
		PRIMARY KEY (order_id, user_id)
	)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if len(s.PrimaryKeys) != 2 {
		t.Fatalf("expected 2 PKs, got %v", s.PrimaryKeys)
	}
}

func TestCreateTable_UniqueConstraint(t *testing.T) {
	sql := `CREATE TABLE users (email TEXT UNIQUE, name TEXT)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if len(s.Unique) != 1 || s.Unique[0] != "email" {
		t.Errorf("expected UNIQUE [email], got %v", s.Unique)
	}
}

func TestCreateTable_TableLevelUnique(t *testing.T) {
	sql := `CREATE TABLE users (email TEXT, UNIQUE (email))`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if len(s.Unique) != 1 || s.Unique[0] != "email" {
		t.Errorf("expected UNIQUE [email], got %v", s.Unique)
	}
}

func TestCreateTable_TypeAliases(t *testing.T) {
	sql := `CREATE TABLE t (
		a INTEGER,
		b BOOL,
		c JSONB,
		d BIGINT,
		e NUMERIC,
		f TIMESTAMP,
		g BYTEA
	)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	want := []SqlType{SqlInt, SqlBool, SqlJsonb, SqlBigInt, SqlNumeric, SqlTimestamp, SqlBytes}
	for i, col := range s.Columns {
		if col.Type != want[i] {
			t.Errorf("col[%d] %q: expected %q, got %q", i, col.Name, want[i], col.Type)
		}
	}
}

func TestCreateTable_VarcharWithLength(t *testing.T) {
	sql := `CREATE TABLE t (name VARCHAR(100) NOT NULL)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if s.Columns[0].Type != SqlText {
		t.Errorf("expected TEXT (VARCHAR alias), got %q", s.Columns[0].Type)
	}
}

func TestCreateTable_NumericPrecision(t *testing.T) {
	sql := `CREATE TABLE t (price NUMERIC(10,2) DEFAULT 0.0)`
	stmts, err := ParseText(sql)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(CreateTableStmt)
	if s.Columns[0].Type != SqlNumeric {
		t.Errorf("expected NUMERIC, got %q", s.Columns[0].Type)
	}
}

// ── DROP TABLE ────────────────────────────────────────────────────────────────

func TestDropTable_Basic(t *testing.T) {
	stmts, err := ParseText(`DROP TABLE users`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(DropTableStmt)
	if s.Table.Name != "users" {
		t.Errorf("unexpected table: %q", s.Table.Name)
	}
	if s.IfExists {
		t.Error("unexpected IfExists flag")
	}
}

func TestDropTable_IfExists(t *testing.T) {
	stmts, err := ParseText(`DROP TABLE IF EXISTS apps.logs`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(DropTableStmt)
	if !s.IfExists {
		t.Error("expected IfExists = true")
	}
	if s.Table.Schema != "apps" || s.Table.Name != "logs" {
		t.Errorf("unexpected table ref: %+v", s.Table)
	}
}

// ── ALTER TABLE ───────────────────────────────────────────────────────────────

func TestAlterTable_AddColumn(t *testing.T) {
	stmts, err := ParseText(`ALTER TABLE users ADD COLUMN phone TEXT`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(AlterTableStmt)
	if s.Action != AlterAddColumn {
		t.Errorf("expected AlterAddColumn, got %v", s.Action)
	}
	if s.Column.Name != "phone" || s.Column.Type != SqlText {
		t.Errorf("unexpected column: %+v", s.Column)
	}
}

func TestAlterTable_AddColumnNoKeyword(t *testing.T) {
	// COLUMN keyword is optional
	stmts, err := ParseText(`ALTER TABLE users ADD score INT DEFAULT 0`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(AlterTableStmt)
	if s.Action != AlterAddColumn || s.Column.Name != "score" {
		t.Errorf("unexpected: %+v", s)
	}
}

func TestAlterTable_DropColumn(t *testing.T) {
	stmts, err := ParseText(`ALTER TABLE users DROP COLUMN phone`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(AlterTableStmt)
	if s.Action != AlterDropColumn || s.ColName != "phone" {
		t.Errorf("unexpected: %+v", s)
	}
}

func TestAlterTable_SetDefault(t *testing.T) {
	stmts, err := ParseText(`ALTER TABLE users ALTER COLUMN age SET DEFAULT 18`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(AlterTableStmt)
	if s.Action != AlterAlterColumn || s.ColName != "age" {
		t.Errorf("unexpected action/col: %+v", s)
	}
	if s.Column.Default.(int64) != 18 {
		t.Errorf("expected DEFAULT 18, got %v", s.Column.Default)
	}
}

func TestAlterTable_SetNotNull(t *testing.T) {
	stmts, err := ParseText(`ALTER TABLE users ALTER COLUMN email SET NOT NULL`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(AlterTableStmt)
	if !s.Column.NotNull {
		t.Error("expected NotNull = true")
	}
}

func TestAlterTable_AddPrimaryKey(t *testing.T) {
	stmts, err := ParseText(`ALTER TABLE orders ADD PRIMARY KEY (order_id)`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(AlterTableStmt)
	if s.Action != AlterAddPrimaryKey {
		t.Errorf("expected AlterAddPrimaryKey, got %v", s.Action)
	}
	if !strings.Contains(s.Column.Name, "order_id") {
		t.Errorf("unexpected PK cols: %q", s.Column.Name)
	}
}

func TestAlterTable_SchemaQualified(t *testing.T) {
	stmts, err := ParseText(`ALTER TABLE apps.users ADD COLUMN bio TEXT`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(AlterTableStmt)
	if s.Table.Schema != "apps" || s.Table.Name != "users" {
		t.Errorf("unexpected table: %+v", s.Table)
	}
}
