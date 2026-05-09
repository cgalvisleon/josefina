package mysql_test

import (
	"strings"
	"testing"

	"github.com/cgalvisleon/josefina/internal/stmt"
	"github.com/cgalvisleon/josefina/internal/stmt/mysql"
)

// helper: parse one statement and convert to MySQL SQL.
func toMySQLSQL(t *testing.T, input string) string {
	t.Helper()
	stmts, err := stmt.ParseText(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(stmts) == 0 {
		t.Fatal("no statements parsed")
	}
	sql, err := mysql.ToSQL(stmts[0])
	if err != nil {
		t.Fatalf("ToSQL: %v", err)
	}
	return sql
}

// ── SELECT ────────────────────────────────────────────────────────────────────

func TestMySQLSelect_Star(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM apps.users`)
	if !strings.HasPrefix(sql, "SELECT * FROM apps.users") {
		t.Errorf("unexpected output: %s", sql)
	}
}

func TestMySQLSelect_Columns(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT username, email FROM apps.users`)
	if !strings.Contains(sql, "username, email") {
		t.Errorf("expected columns in output: %s", sql)
	}
}

func TestMySQLSelect_Where(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users WHERE age >= 25 AND active = TRUE`)
	if !strings.Contains(sql, "age >= 25") {
		t.Errorf("expected age condition: %s", sql)
	}
	// MySQL preserves TRUE/FALSE
	if !strings.Contains(sql, "active = TRUE") {
		t.Errorf("expected boolean TRUE preserved: %s", sql)
	}
}

func TestMySQLSelect_ILike(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users WHERE email ILIKE '%example%'`)
	if !strings.Contains(sql, "LOWER(email) LIKE LOWER(") {
		t.Errorf("expected ILIKE emulation: %s", sql)
	}
}

func TestMySQLSelect_Limit(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users LIMIT 10`)
	if !strings.Contains(sql, "LIMIT 10") {
		t.Errorf("expected LIMIT: %s", sql)
	}
}

func TestMySQLSelect_LimitOffset(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users LIMIT 10 OFFSET 20`)
	if !strings.Contains(sql, "LIMIT 10 OFFSET 20") {
		t.Errorf("expected LIMIT … OFFSET: %s", sql)
	}
}

func TestMySQLSelect_Page(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users LIMIT 5 PAGE 3`)
	if !strings.Contains(sql, "LIMIT 5 OFFSET 10") {
		t.Errorf("expected page-3 offset 10: %s", sql)
	}
}

func TestMySQLSelect_InnerJoin(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM apps.users INNER JOIN apps.orders ON users.username = orders.username`)
	if !strings.Contains(sql, "INNER JOIN apps.orders") {
		t.Errorf("expected INNER JOIN: %s", sql)
	}
	if !strings.Contains(sql, "users.username = orders.username") {
		t.Errorf("expected ON clause: %s", sql)
	}
}

func TestMySQLSelect_LeftJoin(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users LEFT JOIN orders ON users.username = orders.username`)
	if !strings.Contains(sql, "LEFT OUTER JOIN") {
		t.Errorf("expected LEFT OUTER JOIN: %s", sql)
	}
}

func TestMySQLSelect_FullJoinError(t *testing.T) {
	stmts, err := stmt.ParseText(`SELECT * FROM users FULL JOIN orders ON users.username = orders.username`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = mysql.ToSQL(stmts[0])
	if err == nil {
		t.Fatal("expected error for FULL OUTER JOIN in MySQL")
	}
	if !strings.Contains(err.Error(), "FULL OUTER JOIN") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMySQLSelect_In(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users WHERE age IN (25, 30, 35)`)
	if !strings.Contains(sql, "age IN (25, 30, 35)") {
		t.Errorf("expected IN clause: %s", sql)
	}
}

func TestMySQLSelect_Between(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users WHERE age BETWEEN 20 AND 40`)
	if !strings.Contains(sql, "age BETWEEN 20 AND 40") {
		t.Errorf("expected BETWEEN: %s", sql)
	}
}

func TestMySQLSelect_OrderBy(t *testing.T) {
	sql := toMySQLSQL(t, `SELECT * FROM users ORDER BY age DESC`)
	if !strings.Contains(sql, "ORDER BY age DESC") {
		t.Errorf("expected ORDER BY: %s", sql)
	}
}

// ── INSERT ────────────────────────────────────────────────────────────────────

func TestMySQLInsert_Single(t *testing.T) {
	sql := toMySQLSQL(t, `INSERT INTO users (username, age) VALUES ('alice', 30)`)
	if !strings.HasPrefix(sql, "INSERT INTO users") {
		t.Errorf("unexpected output: %s", sql)
	}
	if !strings.Contains(sql, "'alice'") || !strings.Contains(sql, "30") {
		t.Errorf("expected values: %s", sql)
	}
}

func TestMySQLInsert_MultiRow(t *testing.T) {
	sql := toMySQLSQL(t, `INSERT INTO users (username, age) VALUES ('bob', 25), ('carol', 35)`)
	// MySQL supports standard multi-row VALUES — no INSERT ALL needed
	if !strings.HasPrefix(sql, "INSERT INTO users") {
		t.Errorf("unexpected prefix: %s", sql)
	}
	if !strings.Contains(sql, "('bob', 25), ('carol', 35)") {
		t.Errorf("expected both rows in VALUES: %s", sql)
	}
	if strings.Contains(sql, "INSERT ALL") {
		t.Errorf("unexpected INSERT ALL (Oracle-style): %s", sql)
	}
}

func TestMySQLInsert_BooleanLiteral(t *testing.T) {
	sql := toMySQLSQL(t, `INSERT INTO users (username, active) VALUES ('dave', TRUE)`)
	// MySQL preserves TRUE, unlike Oracle which converts to 1
	if !strings.Contains(sql, "TRUE") {
		t.Errorf("expected TRUE preserved: %s", sql)
	}
}

// ── UPSERT → INSERT … ON DUPLICATE KEY UPDATE ─────────────────────────────────

func TestMySQLUpsert_OnDuplicateKey(t *testing.T) {
	sql := toMySQLSQL(t, `UPSERT INTO users (_idx, username, age) VALUES ('alice', 'alice', 31)`)
	if !strings.HasPrefix(sql, "INSERT INTO users") {
		t.Errorf("expected INSERT INTO: %s", sql)
	}
	if !strings.Contains(sql, "ON DUPLICATE KEY UPDATE") {
		t.Errorf("expected ON DUPLICATE KEY UPDATE: %s", sql)
	}
	// _idx must not appear in UPDATE clause
	updatePart := sql[strings.Index(sql, "ON DUPLICATE KEY UPDATE"):]
	if strings.Contains(updatePart, "_idx =") {
		t.Errorf("_idx must not appear in UPDATE clause: %s", sql)
	}
}

func TestMySQLUpsert_MultiRow(t *testing.T) {
	sql := toMySQLSQL(t, `
		UPSERT INTO users (_idx, username, age) VALUES
			('alice', 'alice', 31),
			('bob',   'bob',   26)
	`)
	if !strings.Contains(sql, "ON DUPLICATE KEY UPDATE") {
		t.Errorf("expected ON DUPLICATE KEY UPDATE: %s", sql)
	}
	// Both rows must be in the VALUES section
	if !strings.Contains(sql, "'alice'") || !strings.Contains(sql, "'bob'") {
		t.Errorf("expected both rows: %s", sql)
	}
	if strings.Contains(sql, "UNION ALL") {
		t.Errorf("unexpected UNION ALL (Oracle-style): %s", sql)
	}
}

func TestMySQLUpsert_ValuesFunction(t *testing.T) {
	sql := toMySQLSQL(t, `UPSERT INTO users (_idx, username, age) VALUES ('alice', 'alice', 31)`)
	if !strings.Contains(sql, "VALUES(username)") || !strings.Contains(sql, "VALUES(age)") {
		t.Errorf("expected VALUES(col) references in UPDATE: %s", sql)
	}
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

func TestMySQLUpdate_Basic(t *testing.T) {
	sql := toMySQLSQL(t, `UPDATE users SET active = FALSE WHERE age < 18`)
	if !strings.HasPrefix(sql, "UPDATE users SET") {
		t.Errorf("unexpected output: %s", sql)
	}
	if !strings.Contains(sql, "active = FALSE") {
		t.Errorf("expected FALSE preserved: %s", sql)
	}
	if !strings.Contains(sql, "WHERE age < 18") {
		t.Errorf("expected WHERE clause: %s", sql)
	}
}

// ── DELETE ────────────────────────────────────────────────────────────────────

func TestMySQLDelete_Basic(t *testing.T) {
	sql := toMySQLSQL(t, `DELETE FROM logs WHERE active = FALSE`)
	if !strings.HasPrefix(sql, "DELETE FROM logs WHERE") {
		t.Errorf("unexpected output: %s", sql)
	}
}

// ── CREATE TABLE ──────────────────────────────────────────────────────────────

func TestMySQLCreateTable_TypeMapping(t *testing.T) {
	sql := toMySQLSQL(t, `
		CREATE TABLE products (
			product_id KEY     NOT NULL,
			name       TEXT    NOT NULL,
			price      NUMERIC DEFAULT 0.0,
			active     BOOLEAN DEFAULT TRUE,
			PRIMARY KEY (product_id)
		)
	`)
	if !strings.Contains(sql, "VARCHAR(255)") {
		t.Errorf("expected KEY→VARCHAR(255): %s", sql)
	}
	if !strings.Contains(sql, "TEXT") {
		t.Errorf("expected TEXT type: %s", sql)
	}
	if !strings.Contains(sql, "DECIMAL") {
		t.Errorf("expected NUMERIC→DECIMAL: %s", sql)
	}
	if !strings.Contains(sql, "TINYINT(1)") {
		t.Errorf("expected BOOLEAN→TINYINT(1): %s", sql)
	}
	if !strings.Contains(sql, "PRIMARY KEY") {
		t.Errorf("expected PRIMARY KEY: %s", sql)
	}
	if !strings.Contains(sql, "CHECK") {
		t.Errorf("expected CHECK constraint for BOOLEAN: %s", sql)
	}
	if !strings.Contains(sql, "ENGINE=InnoDB") {
		t.Errorf("expected ENGINE=InnoDB: %s", sql)
	}
}

func TestMySQLCreateTable_IfNotExists(t *testing.T) {
	sql := toMySQLSQL(t, `CREATE TABLE IF NOT EXISTS apps.logs (msg TEXT)`)
	if !strings.HasPrefix(sql, "CREATE TABLE IF NOT EXISTS") {
		t.Errorf("expected native IF NOT EXISTS (no PL/SQL): %s", sql)
	}
	// MySQL does it natively — no BEGIN/EXECUTE IMMEDIATE
	if strings.Contains(sql, "EXECUTE IMMEDIATE") {
		t.Errorf("unexpected PL/SQL block: %s", sql)
	}
}

// ── DROP TABLE ────────────────────────────────────────────────────────────────

func TestMySQLDropTable_Plain(t *testing.T) {
	sql := toMySQLSQL(t, `DROP TABLE products`)
	if sql != "DROP TABLE products" {
		t.Errorf("unexpected output: %s", sql)
	}
}

func TestMySQLDropTable_IfExists(t *testing.T) {
	sql := toMySQLSQL(t, `DROP TABLE IF EXISTS products`)
	if !strings.HasPrefix(sql, "DROP TABLE IF EXISTS") {
		t.Errorf("expected native IF EXISTS: %s", sql)
	}
	if strings.Contains(sql, "EXECUTE IMMEDIATE") {
		t.Errorf("unexpected PL/SQL block: %s", sql)
	}
}

// ── Transactions ──────────────────────────────────────────────────────────────

func TestMySQLTxStmt_Commit(t *testing.T) {
	stmts, err := stmt.ParseText(`
		BEGIN;
		INSERT INTO users (username, age) VALUES ('alice', 30);
		COMMIT;
	`)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := mysql.ToSQL(stmts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sql, "START TRANSACTION") {
		t.Errorf("expected START TRANSACTION: %s", sql)
	}
	if !strings.Contains(sql, "INSERT INTO users") {
		t.Errorf("expected INSERT: %s", sql)
	}
	if !strings.HasSuffix(strings.TrimSpace(sql), "COMMIT") {
		t.Errorf("expected COMMIT at end: %s", sql)
	}
}

func TestMySQLTxStmt_Rollback(t *testing.T) {
	stmts, err := stmt.ParseText(`BEGIN; UPDATE users SET age = 99; ROLLBACK;`)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := mysql.ToSQL(stmts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sql, "START TRANSACTION") {
		t.Errorf("expected START TRANSACTION: %s", sql)
	}
	if !strings.HasSuffix(strings.TrimSpace(sql), "ROLLBACK") {
		t.Errorf("expected ROLLBACK at end: %s", sql)
	}
}

func TestMySQLStandaloneStatements(t *testing.T) {
	cases := []struct {
		st   stmt.Stmt
		want string
	}{
		{stmt.BeginStmt{}, "START TRANSACTION"},
		{stmt.CommitStmt{}, "COMMIT"},
		{stmt.RollbackStmt{}, "ROLLBACK"},
	}
	for _, tc := range cases {
		got, err := mysql.ToSQL(tc.st)
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

func TestMySQLToSQLAll(t *testing.T) {
	stmts, err := stmt.ParseText(`
		SELECT * FROM users;
		INSERT INTO logs (msg) VALUES ('hello');
	`)
	if err != nil {
		t.Fatal(err)
	}
	sqls, err := mysql.ToSQLAll(stmts)
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
