package sqlserver_test

import (
	"strings"
	"testing"

	"github.com/cgalvisleon/josefina/internal/stmt"
	"github.com/cgalvisleon/josefina/internal/stmt/sqlserver"
)

// helper: parse one statement and convert to T-SQL.
func toTSQL(t *testing.T, input string) string {
	t.Helper()
	stmts, err := stmt.ParseText(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(stmts) == 0 {
		t.Fatal("no statements parsed")
	}
	sql, err := sqlserver.ToSQL(stmts[0])
	if err != nil {
		t.Fatalf("ToSQL: %v", err)
	}
	return sql
}

// ── SELECT ────────────────────────────────────────────────────────────────────

func TestSQLServerSelect_Star(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM apps.users`)
	if !strings.HasPrefix(sql, "SELECT * FROM apps.users") {
		t.Errorf("unexpected output: %s", sql)
	}
}

func TestSQLServerSelect_Columns(t *testing.T) {
	sql := toTSQL(t, `SELECT username, email FROM apps.users`)
	if !strings.Contains(sql, "username, email") {
		t.Errorf("expected columns in output: %s", sql)
	}
}

func TestSQLServerSelect_Where(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM users WHERE age >= 25 AND active = TRUE`)
	if !strings.Contains(sql, "age >= 25") {
		t.Errorf("expected age condition: %s", sql)
	}
	// BIT type: TRUE → 1, FALSE → 0
	if !strings.Contains(sql, "active = 1") {
		t.Errorf("expected TRUE→1: %s", sql)
	}
}

func TestSQLServerSelect_ILike(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM users WHERE email ILIKE '%example%'`)
	if !strings.Contains(sql, "LOWER(email) LIKE LOWER(") {
		t.Errorf("expected ILIKE emulation: %s", sql)
	}
}

func TestSQLServerSelect_TopOnly(t *testing.T) {
	// LIMIT without OFFSET → TOP (n)
	sql := toTSQL(t, `SELECT * FROM users LIMIT 10`)
	if !strings.Contains(sql, "TOP (10)") {
		t.Errorf("expected TOP (10): %s", sql)
	}
	if strings.Contains(sql, "FETCH") {
		t.Errorf("unexpected FETCH when using TOP: %s", sql)
	}
}

func TestSQLServerSelect_OffsetFetch(t *testing.T) {
	// LIMIT + OFFSET → OFFSET … ROWS FETCH NEXT … ROWS ONLY
	sql := toTSQL(t, `SELECT * FROM users ORDER BY age LIMIT 10 OFFSET 20`)
	if strings.Contains(sql, "TOP") {
		t.Errorf("unexpected TOP when OFFSET present: %s", sql)
	}
	if !strings.Contains(sql, "OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY") {
		t.Errorf("expected OFFSET/FETCH: %s", sql)
	}
}

func TestSQLServerSelect_OffsetFetchNoOrderBy(t *testing.T) {
	// When OFFSET is given without ORDER BY, a dummy ORDER BY must be added
	sql := toTSQL(t, `SELECT * FROM users LIMIT 10 OFFSET 5`)
	if !strings.Contains(sql, "ORDER BY (SELECT NULL)") {
		t.Errorf("expected dummy ORDER BY for OFFSET/FETCH: %s", sql)
	}
	if !strings.Contains(sql, "OFFSET 5 ROWS FETCH NEXT 10 ROWS ONLY") {
		t.Errorf("expected OFFSET/FETCH: %s", sql)
	}
}

func TestSQLServerSelect_Page(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM users LIMIT 5 PAGE 3`)
	if !strings.Contains(sql, "OFFSET 10 ROWS FETCH NEXT 5 ROWS ONLY") {
		t.Errorf("expected page-3 offset 10: %s", sql)
	}
}

func TestSQLServerSelect_InnerJoin(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM apps.users INNER JOIN apps.orders ON users.username = orders.username`)
	if !strings.Contains(sql, "INNER JOIN apps.orders") {
		t.Errorf("expected INNER JOIN: %s", sql)
	}
	if !strings.Contains(sql, "users.username = orders.username") {
		t.Errorf("expected ON clause: %s", sql)
	}
}

func TestSQLServerSelect_LeftJoin(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM users LEFT JOIN orders ON users.username = orders.username`)
	if !strings.Contains(sql, "LEFT OUTER JOIN") {
		t.Errorf("expected LEFT OUTER JOIN: %s", sql)
	}
}

func TestSQLServerSelect_FullJoin(t *testing.T) {
	// SQL Server supports FULL OUTER JOIN natively (unlike MySQL)
	sql := toTSQL(t, `SELECT * FROM users FULL JOIN orders ON users.username = orders.username`)
	if !strings.Contains(sql, "FULL OUTER JOIN") {
		t.Errorf("expected FULL OUTER JOIN: %s", sql)
	}
}

func TestSQLServerSelect_In(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM users WHERE age IN (25, 30, 35)`)
	if !strings.Contains(sql, "age IN (25, 30, 35)") {
		t.Errorf("expected IN clause: %s", sql)
	}
}

func TestSQLServerSelect_Between(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM users WHERE age BETWEEN 20 AND 40`)
	if !strings.Contains(sql, "age BETWEEN 20 AND 40") {
		t.Errorf("expected BETWEEN: %s", sql)
	}
}

func TestSQLServerSelect_OrderBy(t *testing.T) {
	sql := toTSQL(t, `SELECT * FROM users ORDER BY age DESC`)
	if !strings.Contains(sql, "ORDER BY age DESC") {
		t.Errorf("expected ORDER BY: %s", sql)
	}
}

// ── INSERT ────────────────────────────────────────────────────────────────────

func TestSQLServerInsert_Single(t *testing.T) {
	sql := toTSQL(t, `INSERT INTO users (username, age) VALUES ('alice', 30)`)
	if !strings.HasPrefix(sql, "INSERT INTO users") {
		t.Errorf("unexpected output: %s", sql)
	}
	// Strings prefixed with N for Unicode
	if !strings.Contains(sql, "N'alice'") {
		t.Errorf("expected N-prefixed string: %s", sql)
	}
}

func TestSQLServerInsert_MultiRow(t *testing.T) {
	sql := toTSQL(t, `INSERT INTO users (username, age) VALUES ('bob', 25), ('carol', 35)`)
	if !strings.HasPrefix(sql, "INSERT INTO users") {
		t.Errorf("unexpected prefix: %s", sql)
	}
	if !strings.Contains(sql, "N'bob'") || !strings.Contains(sql, "N'carol'") {
		t.Errorf("expected both N-prefixed rows: %s", sql)
	}
	if strings.Contains(sql, "INSERT ALL") {
		t.Errorf("unexpected INSERT ALL: %s", sql)
	}
}

func TestSQLServerInsert_BooleanLiteral(t *testing.T) {
	sql := toTSQL(t, `INSERT INTO users (username, active) VALUES ('dave', TRUE)`)
	// BIT: TRUE → 1
	if !strings.Contains(sql, ", 1)") {
		t.Errorf("expected TRUE→1: %s", sql)
	}
}

// ── UPSERT → MERGE ────────────────────────────────────────────────────────────

func TestSQLServerUpsert_MergeStructure(t *testing.T) {
	sql := toTSQL(t, `UPSERT INTO users (_idx, username, age) VALUES ('alice', 'alice', 31)`)
	if !strings.HasPrefix(sql, "MERGE INTO users AS t") {
		t.Errorf("expected MERGE INTO: %s", sql)
	}
	if !strings.Contains(sql, "USING (VALUES") {
		t.Errorf("expected VALUES-based USING: %s", sql)
	}
	if strings.Contains(sql, "FROM DUAL") {
		t.Errorf("unexpected FROM DUAL (Oracle-style): %s", sql)
	}
	if !strings.Contains(sql, "ON t._idx = s._idx") {
		t.Errorf("expected ON clause: %s", sql)
	}
	if !strings.Contains(sql, "WHEN MATCHED THEN") {
		t.Errorf("expected WHEN MATCHED: %s", sql)
	}
	if !strings.Contains(sql, "WHEN NOT MATCHED THEN") {
		t.Errorf("expected WHEN NOT MATCHED: %s", sql)
	}
	// T-SQL requires semicolon at end of MERGE
	if !strings.HasSuffix(strings.TrimSpace(sql), ";") {
		t.Errorf("expected trailing semicolon on MERGE: %s", sql)
	}
}

func TestSQLServerUpsert_MultiRow(t *testing.T) {
	sql := toTSQL(t, `
		UPSERT INTO users (_idx, username, age) VALUES
			('alice', 'alice', 31),
			('bob',   'bob',   26)
	`)
	if !strings.Contains(sql, "USING (VALUES") {
		t.Errorf("expected VALUES-based USING: %s", sql)
	}
	if strings.Contains(sql, "UNION ALL") {
		t.Errorf("unexpected UNION ALL (Oracle-style): %s", sql)
	}
	// Both rows must appear in the VALUES section
	if !strings.Contains(sql, "N'alice'") || !strings.Contains(sql, "N'bob'") {
		t.Errorf("expected both rows: %s", sql)
	}
}

func TestSQLServerUpsert_KeyExcludedFromUpdate(t *testing.T) {
	sql := toTSQL(t, `UPSERT INTO users (_idx, username, age) VALUES ('alice', 'alice', 31)`)
	updatePart := sql[strings.Index(sql, "UPDATE SET"):]
	if strings.Contains(updatePart, "t._idx = s._idx") {
		t.Errorf("_idx must not appear in UPDATE SET: %s", sql)
	}
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

func TestSQLServerUpdate_Basic(t *testing.T) {
	sql := toTSQL(t, `UPDATE users SET active = FALSE WHERE age < 18`)
	if !strings.HasPrefix(sql, "UPDATE users SET") {
		t.Errorf("unexpected output: %s", sql)
	}
	// BIT: FALSE → 0
	if !strings.Contains(sql, "active = 0") {
		t.Errorf("expected FALSE→0: %s", sql)
	}
	if !strings.Contains(sql, "WHERE age < 18") {
		t.Errorf("expected WHERE clause: %s", sql)
	}
}

// ── DELETE ────────────────────────────────────────────────────────────────────

func TestSQLServerDelete_Basic(t *testing.T) {
	sql := toTSQL(t, `DELETE FROM logs WHERE active = FALSE`)
	if !strings.HasPrefix(sql, "DELETE FROM logs WHERE") {
		t.Errorf("unexpected output: %s", sql)
	}
	if !strings.Contains(sql, "active = 0") {
		t.Errorf("expected FALSE→0: %s", sql)
	}
}

// ── CREATE TABLE ──────────────────────────────────────────────────────────────

func TestSQLServerCreateTable_TypeMapping(t *testing.T) {
	sql := toTSQL(t, `
		CREATE TABLE products (
			product_id KEY     NOT NULL,
			name       TEXT    NOT NULL,
			price      NUMERIC DEFAULT 0.0,
			active     BOOLEAN DEFAULT TRUE,
			data       JSON,
			payload    BYTEA,
			PRIMARY KEY (product_id)
		)
	`)
	if !strings.Contains(sql, "NVARCHAR(255)") {
		t.Errorf("expected KEY→NVARCHAR(255): %s", sql)
	}
	if !strings.Contains(sql, "NVARCHAR(MAX)") {
		t.Errorf("expected TEXT→NVARCHAR(MAX): %s", sql)
	}
	if !strings.Contains(sql, "DECIMAL") {
		t.Errorf("expected NUMERIC→DECIMAL: %s", sql)
	}
	if !strings.Contains(sql, "BIT") {
		t.Errorf("expected BOOLEAN→BIT: %s", sql)
	}
	if !strings.Contains(sql, "VARBINARY(MAX)") {
		t.Errorf("expected BYTEA→VARBINARY(MAX): %s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT pk_products PRIMARY KEY") {
		t.Errorf("expected PRIMARY KEY constraint: %s", sql)
	}
}

func TestSQLServerCreateTable_IfNotExists(t *testing.T) {
	sql := toTSQL(t, `CREATE TABLE IF NOT EXISTS apps.logs (msg TEXT)`)
	if !strings.HasPrefix(sql, "IF OBJECT_ID(") {
		t.Errorf("expected OBJECT_ID guard: %s", sql)
	}
	if !strings.Contains(sql, "N'U'") {
		t.Errorf("expected N'U' type parameter: %s", sql)
	}
	if !strings.Contains(sql, "IS NULL") {
		t.Errorf("expected IS NULL check: %s", sql)
	}
	if !strings.Contains(sql, "BEGIN") || !strings.Contains(sql, "END") {
		t.Errorf("expected BEGIN/END block: %s", sql)
	}
	// No PL/SQL EXECUTE IMMEDIATE
	if strings.Contains(sql, "EXECUTE IMMEDIATE") {
		t.Errorf("unexpected Oracle PL/SQL: %s", sql)
	}
}

// ── DROP TABLE ────────────────────────────────────────────────────────────────

func TestSQLServerDropTable_Plain(t *testing.T) {
	sql := toTSQL(t, `DROP TABLE products`)
	if sql != "DROP TABLE products" {
		t.Errorf("unexpected output: %s", sql)
	}
}

func TestSQLServerDropTable_IfExists(t *testing.T) {
	sql := toTSQL(t, `DROP TABLE IF EXISTS products`)
	if !strings.HasPrefix(sql, "DROP TABLE IF EXISTS") {
		t.Errorf("expected native DROP TABLE IF EXISTS: %s", sql)
	}
}

// ── Transactions ──────────────────────────────────────────────────────────────

func TestSQLServerTxStmt_Commit(t *testing.T) {
	stmts, err := stmt.ParseText(`
		BEGIN;
		INSERT INTO users (username, age) VALUES ('alice', 30);
		COMMIT;
	`)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := sqlserver.ToSQL(stmts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sql, "BEGIN TRANSACTION") {
		t.Errorf("expected BEGIN TRANSACTION: %s", sql)
	}
	if !strings.Contains(sql, "INSERT INTO users") {
		t.Errorf("expected INSERT: %s", sql)
	}
	if !strings.HasSuffix(strings.TrimSpace(sql), "COMMIT") {
		t.Errorf("expected COMMIT at end: %s", sql)
	}
}

func TestSQLServerTxStmt_Rollback(t *testing.T) {
	stmts, err := stmt.ParseText(`BEGIN; UPDATE users SET age = 99; ROLLBACK;`)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := sqlserver.ToSQL(stmts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sql, "BEGIN TRANSACTION") {
		t.Errorf("expected BEGIN TRANSACTION: %s", sql)
	}
	if !strings.HasSuffix(strings.TrimSpace(sql), "ROLLBACK") {
		t.Errorf("expected ROLLBACK at end: %s", sql)
	}
}

func TestSQLServerStandaloneStatements(t *testing.T) {
	cases := []struct {
		st   stmt.Stmt
		want string
	}{
		{stmt.BeginStmt{}, "BEGIN TRANSACTION"},
		{stmt.CommitStmt{}, "COMMIT"},
		{stmt.RollbackStmt{}, "ROLLBACK"},
	}
	for _, tc := range cases {
		got, err := sqlserver.ToSQL(tc.st)
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

func TestSQLServerToSQLAll(t *testing.T) {
	stmts, err := stmt.ParseText(`
		SELECT * FROM users;
		INSERT INTO logs (msg) VALUES ('hello');
	`)
	if err != nil {
		t.Fatal(err)
	}
	sqls, err := sqlserver.ToSQLAll(stmts)
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
