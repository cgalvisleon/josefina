package stmt_test

import (
	"testing"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

func TestSetSqlState_Josefina(t *testing.T) {
	stmts, err := stmt.ParseText("SET SQL STATE JOSEFINA;")
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(stmts))
	}
	s, ok := stmts[0].(stmt.SetSqlStateStmt)
	if !ok {
		t.Fatalf("expected SetSqlStateStmt, got %T", stmts[0])
	}
	if s.Dialect != stmt.DialectJosefina {
		t.Errorf("expected JOSEFINA, got %q", s.Dialect)
	}
}

func TestSetSqlState_PostgreSQL(t *testing.T) {
	stmts, err := stmt.ParseText("SET SQL STATE POSTGRESQL;")
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(stmt.SetSqlStateStmt)
	if s.Dialect != stmt.DialectPostgreSQL {
		t.Errorf("expected POSTGRESQL, got %q", s.Dialect)
	}
}

func TestSetSqlState_MySQL(t *testing.T) {
	stmts, err := stmt.ParseText("SET SQL STATE MYSQL;")
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(stmt.SetSqlStateStmt)
	if s.Dialect != stmt.DialectMySQL {
		t.Errorf("expected MYSQL, got %q", s.Dialect)
	}
}

func TestSetSqlState_Oracle(t *testing.T) {
	stmts, err := stmt.ParseText("SET SQL STATE ORACLE;")
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(stmt.SetSqlStateStmt)
	if s.Dialect != stmt.DialectOracle {
		t.Errorf("expected ORACLE, got %q", s.Dialect)
	}
}

func TestSetSqlState_SQLServer(t *testing.T) {
	stmts, err := stmt.ParseText("SET SQL STATE SQLSERVER;")
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(stmt.SetSqlStateStmt)
	if s.Dialect != stmt.DialectSQLServer {
		t.Errorf("expected SQLSERVER, got %q", s.Dialect)
	}
}

func TestSetSqlState_InvalidDialect(t *testing.T) {
	_, err := stmt.ParseText("SET SQL STATE MONGODB;")
	if err == nil {
		t.Fatal("expected error for unknown dialect, got nil")
	}
}

func TestSetSqlState_MissingDialect(t *testing.T) {
	_, err := stmt.ParseText("SET SQL STATE;")
	if err == nil {
		t.Fatal("expected error for missing dialect, got nil")
	}
}

func TestSetSqlState_CaseInsensitive(t *testing.T) {
	stmts, err := stmt.ParseText("set sql state mysql;")
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(stmt.SetSqlStateStmt)
	if s.Dialect != stmt.DialectMySQL {
		t.Errorf("expected MYSQL, got %q", s.Dialect)
	}
}
