package oracle_test

import (
	"strings"
	"testing"

	"github.com/cgalvisleon/josefina/internal/stmt"
	"github.com/cgalvisleon/josefina/internal/stmt/oracle"
)

// helper: parse one statement and convert to Oracle SQL.
func toOracleSQL(t *testing.T, input string) string {
	t.Helper()
	stmts, err := stmt.ParseText(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(stmts) == 0 {
		t.Fatal("no statements parsed")
	}
	sql, err := oracle.ToSQL(stmts[0])
	if err != nil {
		t.Fatalf("ToSQL: %v", err)
	}
	return sql
}

// ── SELECT ────────────────────────────────────────────────────────────────────

func TestOracleSelect_Star(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM apps.users`)
	if !strings.HasPrefix(sql, "SELECT * FROM apps.users") {
		t.Errorf("unexpected output: %s", sql)
	}
}

func TestOracleSelect_Columns(t *testing.T) {
	sql := toOracleSQL(t, `SELECT username, email FROM apps.users`)
	if !strings.Contains(sql, "username, email") {
		t.Errorf("expected columns in output: %s", sql)
	}
}

func TestOracleSelect_Where(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users WHERE age >= 25 AND active = TRUE`)
	if !strings.Contains(sql, "age >= 25") {
		t.Errorf("expected age condition: %s", sql)
	}
	// TRUE becomes 1 in Oracle
	if !strings.Contains(sql, "active = 1") {
		t.Errorf("expected boolean 1 for TRUE: %s", sql)
	}
}

func TestOracleSelect_ILike(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users WHERE email ILIKE '%example%'`)
	if !strings.Contains(sql, "UPPER(email) LIKE UPPER(") {
		t.Errorf("expected ILIKE emulation: %s", sql)
	}
}

func TestOracleSelect_FetchFirst(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users LIMIT 10`)
	if !strings.Contains(sql, "FETCH FIRST 10 ROWS ONLY") {
		t.Errorf("expected FETCH FIRST: %s", sql)
	}
}

func TestOracleSelect_OffsetFetch(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users LIMIT 10 OFFSET 20`)
	if !strings.Contains(sql, "OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY") {
		t.Errorf("expected OFFSET … FETCH: %s", sql)
	}
}

func TestOracleSelect_PageFetch(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users LIMIT 5 PAGE 3`)
	if !strings.Contains(sql, "OFFSET 10 ROWS FETCH NEXT 5 ROWS ONLY") {
		t.Errorf("expected page 3 offset 10: %s", sql)
	}
}

func TestOracleSelect_InnerJoin(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM apps.users INNER JOIN apps.orders ON users.username = orders.username`)
	if !strings.Contains(sql, "INNER JOIN apps.orders") {
		t.Errorf("expected INNER JOIN: %s", sql)
	}
	if !strings.Contains(sql, "users.username = orders.username") {
		t.Errorf("expected ON clause: %s", sql)
	}
}

func TestOracleSelect_LeftJoin(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users LEFT JOIN orders ON users.username = orders.username`)
	if !strings.Contains(sql, "LEFT OUTER JOIN") {
		t.Errorf("expected LEFT OUTER JOIN: %s", sql)
	}
}

func TestOracleSelect_In(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users WHERE age IN (25, 30, 35)`)
	if !strings.Contains(sql, "age IN (25, 30, 35)") {
		t.Errorf("expected IN clause: %s", sql)
	}
}

func TestOracleSelect_Between(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users WHERE age BETWEEN 20 AND 40`)
	if !strings.Contains(sql, "age BETWEEN 20 AND 40") {
		t.Errorf("expected BETWEEN: %s", sql)
	}
}

func TestOracleSelect_OrderBy(t *testing.T) {
	sql := toOracleSQL(t, `SELECT * FROM users ORDER BY age DESC`)
	if !strings.Contains(sql, "ORDER BY age DESC") {
		t.Errorf("expected ORDER BY: %s", sql)
	}
}

// ── INSERT ────────────────────────────────────────────────────────────────────

func TestOracleInsert_Single(t *testing.T) {
	sql := toOracleSQL(t, `INSERT INTO users (username, age) VALUES ('alice', 30)`)
	if !strings.HasPrefix(sql, "INSERT INTO users") {
		t.Errorf("unexpected output: %s", sql)
	}
	if !strings.Contains(sql, "'alice'") || !strings.Contains(sql, "30") {
		t.Errorf("expected values: %s", sql)
	}
}

func TestOracleInsert_MultiRow(t *testing.T) {
	sql := toOracleSQL(t, `
		INSERT INTO users (username, age) VALUES ('bob', 25), ('carol', 35)
	`)
	if !strings.HasPrefix(sql, "INSERT ALL") {
		t.Errorf("expected INSERT ALL for multi-row: %s", sql)
	}
	if !strings.Contains(sql, "SELECT 1 FROM DUAL") {
		t.Errorf("expected SELECT 1 FROM DUAL trailer: %s", sql)
	}
	if !strings.Contains(sql, "'bob'") || !strings.Contains(sql, "'carol'") {
		t.Errorf("expected both rows: %s", sql)
	}
}

func TestOracleInsert_BooleanLiteral(t *testing.T) {
	sql := toOracleSQL(t, `INSERT INTO users (username, active) VALUES ('dave', TRUE)`)
	if !strings.Contains(sql, ", 1)") {
		t.Errorf("expected TRUE→1: %s", sql)
	}
}

// ── UPSERT → MERGE ────────────────────────────────────────────────────────────

func TestOracleUpsert_MergeStructure(t *testing.T) {
	sql := toOracleSQL(t, `UPSERT INTO users (_idx, username, age) VALUES ('alice', 'alice', 31)`)
	if !strings.HasPrefix(sql, "MERGE INTO users t") {
		t.Errorf("expected MERGE INTO: %s", sql)
	}
	if !strings.Contains(sql, "ON (t._idx = s._idx)") {
		t.Errorf("expected ON clause: %s", sql)
	}
	if !strings.Contains(sql, "WHEN MATCHED THEN UPDATE SET") {
		t.Errorf("expected WHEN MATCHED: %s", sql)
	}
	if !strings.Contains(sql, "WHEN NOT MATCHED THEN INSERT") {
		t.Errorf("expected WHEN NOT MATCHED: %s", sql)
	}
}

func TestOracleUpsert_MultiRowUnionAll(t *testing.T) {
	sql := toOracleSQL(t, `
		UPSERT INTO users (_idx, username, age) VALUES
			('alice', 'alice', 31),
			('bob',   'bob',   26)
	`)
	if !strings.Contains(sql, "UNION ALL") {
		t.Errorf("expected UNION ALL for multi-row upsert: %s", sql)
	}
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

func TestOracleUpdate_Basic(t *testing.T) {
	sql := toOracleSQL(t, `UPDATE users SET active = FALSE WHERE age < 18`)
	if !strings.HasPrefix(sql, "UPDATE users SET") {
		t.Errorf("unexpected output: %s", sql)
	}
	if !strings.Contains(sql, "active = 0") {
		t.Errorf("expected FALSE→0: %s", sql)
	}
	if !strings.Contains(sql, "WHERE age < 18") {
		t.Errorf("expected WHERE clause: %s", sql)
	}
}

// ── DELETE ────────────────────────────────────────────────────────────────────

func TestOracleDelete_Basic(t *testing.T) {
	sql := toOracleSQL(t, `DELETE FROM logs WHERE active = FALSE`)
	if !strings.HasPrefix(sql, "DELETE FROM logs WHERE") {
		t.Errorf("unexpected output: %s", sql)
	}
	if !strings.Contains(sql, "active = 0") {
		t.Errorf("expected FALSE→0: %s", sql)
	}
}

// ── CREATE TABLE ──────────────────────────────────────────────────────────────

func TestOracleCreateTable_TypeMapping(t *testing.T) {
	sql := toOracleSQL(t, `
		CREATE TABLE products (
			product_id KEY     NOT NULL,
			name       TEXT    NOT NULL,
			price      NUMERIC DEFAULT 0.0,
			active     BOOLEAN DEFAULT TRUE,
			PRIMARY KEY (product_id)
		)
	`)
	if !strings.Contains(sql, "VARCHAR2(255)") {
		t.Errorf("expected KEY→VARCHAR2(255): %s", sql)
	}
	if !strings.Contains(sql, "VARCHAR2(4000)") {
		t.Errorf("expected TEXT→VARCHAR2(4000): %s", sql)
	}
	if !strings.Contains(sql, "NUMBER") {
		t.Errorf("expected NUMERIC→NUMBER: %s", sql)
	}
	if !strings.Contains(sql, "NUMBER(1)") {
		t.Errorf("expected BOOLEAN→NUMBER(1): %s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT pk_products PRIMARY KEY") {
		t.Errorf("expected PRIMARY KEY constraint: %s", sql)
	}
	if !strings.Contains(sql, "CHECK (active IN (0, 1))") {
		t.Errorf("expected CHECK constraint for BOOLEAN: %s", sql)
	}
}

func TestOracleCreateTable_IfNotExists(t *testing.T) {
	sql := toOracleSQL(t, `CREATE TABLE IF NOT EXISTS apps.logs (msg TEXT)`)
	if !strings.HasPrefix(sql, "BEGIN\n") {
		t.Errorf("expected PL/SQL block for IF NOT EXISTS: %s", sql)
	}
	if !strings.Contains(sql, "SQLCODE != -955") {
		t.Errorf("expected ORA-00955 guard: %s", sql)
	}
}

// ── DROP TABLE ────────────────────────────────────────────────────────────────

func TestOracleDropTable_Plain(t *testing.T) {
	sql := toOracleSQL(t, `DROP TABLE products`)
	if sql != "DROP TABLE products" {
		t.Errorf("unexpected output: %s", sql)
	}
}

func TestOracleDropTable_IfExists(t *testing.T) {
	sql := toOracleSQL(t, `DROP TABLE IF EXISTS products`)
	if !strings.HasPrefix(sql, "BEGIN\n") {
		t.Errorf("expected PL/SQL block for IF EXISTS: %s", sql)
	}
	if !strings.Contains(sql, "SQLCODE != -942") {
		t.Errorf("expected ORA-00942 guard: %s", sql)
	}
}

// ── Transactions ──────────────────────────────────────────────────────────────

func TestOracleTxStmt_Commit(t *testing.T) {
	stmts, err := stmt.ParseText(`
		BEGIN;
		INSERT INTO users (username, age) VALUES ('alice', 30);
		COMMIT;
	`)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := oracle.ToSQL(stmts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, "INSERT INTO users") {
		t.Errorf("expected INSERT: %s", sql)
	}
	if !strings.HasSuffix(strings.TrimSpace(sql), "COMMIT") {
		t.Errorf("expected COMMIT at end: %s", sql)
	}
	// No BEGIN in Oracle output
	if strings.Contains(sql, "BEGIN") {
		t.Errorf("unexpected BEGIN in Oracle tx output: %s", sql)
	}
}

func TestOracleTxStmt_Rollback(t *testing.T) {
	stmts, err := stmt.ParseText(`BEGIN; UPDATE users SET age = 99; ROLLBACK;`)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := oracle.ToSQL(stmts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.TrimSpace(sql), "ROLLBACK") {
		t.Errorf("expected ROLLBACK at end: %s", sql)
	}
}

func TestOracleBeginCommitStandalone(t *testing.T) {
	// CommitStmt and RollbackStmt are valid Oracle standalone statements;
	// test via direct type construction since the parser rejects them outside BEGIN.
	cases := []struct {
		st   stmt.Stmt
		want string
	}{
		{stmt.CommitStmt{}, "COMMIT"},
		{stmt.RollbackStmt{}, "ROLLBACK"},
		{stmt.BeginStmt{}, "-- BEGIN (Oracle: transactions start implicitly)"},
	}
	for _, tc := range cases {
		got, err := oracle.ToSQL(tc.st)
		if err != nil {
			t.Errorf("ToSQL(%T): %v", tc.st, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ToSQL(%T): got %q, want %q", tc.st, got, tc.want)
		}
	}
}

// ── ToSQLAll ──────────────────────────────────────────────────────────────────

func TestOracleToSQLAll(t *testing.T) {
	stmts, err := stmt.ParseText(`
		SELECT * FROM users;
		INSERT INTO logs (msg) VALUES ('hello');
	`)
	if err != nil {
		t.Fatal(err)
	}
	sqls, err := oracle.ToSQLAll(stmts)
	if err != nil {
		t.Fatal(err)
	}
	if len(sqls) != 2 {
		t.Fatalf("expected 2 SQL strings, got %d", len(sqls))
	}
	if !strings.HasPrefix(sqls[0], "SELECT") {
		t.Errorf("expected SELECT first: %s", sqls[0])
	}
	if !strings.HasPrefix(sqls[1], "INSERT") {
		t.Errorf("expected INSERT second: %s", sqls[1])
	}
}
