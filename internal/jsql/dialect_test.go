package jsql_test

import (
	"strings"
	"testing"

	"github.com/cgalvisleon/josefina/internal/stmt"
	"github.com/cgalvisleon/josefina/internal/jsql"
)

func parseOne(t *testing.T, sql string) []stmt.Stmt {
	t.Helper()
	stmts, err := stmt.ParseText(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	return stmts
}

func TestToDialect_Josefina_Select(t *testing.T) {
	stmts := parseOne(t, "SELECT * FROM users WHERE id = 1;")
	out, err := jsql.ToDialect(stmts, stmt.DialectJosefina)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 output, got %d", len(out))
	}
	if !strings.HasPrefix(out[0], "SELECT") {
		t.Errorf("unexpected output: %s", out[0])
	}
}

func TestToDialect_PostgreSQL_Select(t *testing.T) {
	stmts := parseOne(t, "SELECT id, name FROM products;")
	out, err := jsql.ToDialect(stmts, stmt.DialectPostgreSQL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out[0], "SELECT id, name FROM products") {
		t.Errorf("unexpected output: %s", out[0])
	}
}

func TestToDialect_MySQL_Select(t *testing.T) {
	stmts := parseOne(t, "SELECT * FROM orders LIMIT 10;")
	out, err := jsql.ToDialect(stmts, stmt.DialectMySQL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out[0], "LIMIT 10") {
		t.Errorf("expected LIMIT 10 in MySQL output, got: %s", out[0])
	}
}

func TestToDialect_SQLServer_SelectTop(t *testing.T) {
	stmts := parseOne(t, "SELECT * FROM orders LIMIT 5;")
	out, err := jsql.ToDialect(stmts, stmt.DialectSQLServer)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out[0], "TOP (5)") {
		t.Errorf("expected TOP (5) in SQL Server output, got: %s", out[0])
	}
}

func TestToDialect_Oracle_Select(t *testing.T) {
	stmts := parseOne(t, "SELECT * FROM items LIMIT 3;")
	out, err := jsql.ToDialect(stmts, stmt.DialectOracle)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out[0], "FETCH FIRST") {
		t.Errorf("expected FETCH FIRST in Oracle output, got: %s", out[0])
	}
}

func TestToDialect_UnknownDialect(t *testing.T) {
	stmts := parseOne(t, "SELECT id FROM t;")
	_, err := jsql.ToDialect(stmts, stmt.SqlDialect("MONGODB"))
	if err == nil {
		t.Fatal("expected error for unknown dialect")
	}
}

func TestToDialect_MySQL_CreateTable(t *testing.T) {
	sql := "CREATE TABLE IF NOT EXISTS t (id INT, name TEXT);"
	stmts := parseOne(t, sql)
	out, err := jsql.ToDialect(stmts, stmt.DialectMySQL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out[0], "IF NOT EXISTS") {
		t.Errorf("expected IF NOT EXISTS in MySQL DDL, got: %s", out[0])
	}
}

func TestToDialect_SQLServer_CreateTable(t *testing.T) {
	sql := "CREATE TABLE IF NOT EXISTS t (id INT, name TEXT);"
	stmts := parseOne(t, sql)
	out, err := jsql.ToDialect(stmts, stmt.DialectSQLServer)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out[0], "OBJECT_ID") {
		t.Errorf("expected OBJECT_ID guard in SQL Server DDL, got: %s", out[0])
	}
}

func TestToDialect_Postgres_Transaction(t *testing.T) {
	sql := "BEGIN; INSERT INTO t (a) VALUES (1); COMMIT;"
	stmts := parseOne(t, sql)
	out, err := jsql.ToDialect(stmts, stmt.DialectPostgreSQL)
	if err != nil {
		t.Fatal(err)
	}
	// ParseText wraps BEGIN..COMMIT into a single TxStmt — unsupported path returns comment
	if len(out) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(out))
	}
}
