package stmt

import (
	"testing"
)

// ── SELECT ────────────────────────────────────────────────────────────────────

func TestParseSelect_Star(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if len(s.Columns) != 0 {
		t.Errorf("expected SELECT *, got %v", s.Columns)
	}
	if s.From.Name != "users" {
		t.Errorf("expected table users, got %q", s.From.Name)
	}
}

func TestParseSelect_Columns(t *testing.T) {
	stmts, err := ParseText(`SELECT username, email, age FROM apps.users`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if len(s.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(s.Columns))
	}
	if s.From.Schema != "apps" || s.From.Name != "users" {
		t.Errorf("unexpected table ref: %+v", s.From)
	}
}

func TestParseSelect_Where(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users WHERE age >= 25 AND active = TRUE`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if len(s.Where) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(s.Where))
	}
	c0 := s.Where[0]
	if c0.Field != "age" || c0.Op != OpGtEq || c0.Value.(int64) != 25 {
		t.Errorf("unexpected first condition: %+v", c0)
	}
	c1 := s.Where[1]
	if c1.Connector != ConnAnd || c1.Field != "active" || c1.Value.(bool) != true {
		t.Errorf("unexpected second condition: %+v", c1)
	}
}

func TestParseSelect_OrCondition(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users WHERE username = 'alice' OR username = 'bob'`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if len(s.Where) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(s.Where))
	}
	if s.Where[1].Connector != ConnOr {
		t.Errorf("expected OR connector, got %v", s.Where[1].Connector)
	}
}

func TestParseSelect_OrderLimit(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users ORDER BY age DESC LIMIT 10 OFFSET 20`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if len(s.OrderBy) != 1 || s.OrderBy[0].Column != "age" || s.OrderBy[0].Asc {
		t.Errorf("unexpected order: %+v", s.OrderBy)
	}
	if s.Rows != 10 {
		t.Errorf("expected LIMIT 10, got %d", s.Rows)
	}
	if s.Offset != 20 {
		t.Errorf("expected OFFSET 20, got %d", s.Offset)
	}
}

func TestParseSelect_InnerJoin(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users INNER JOIN orders ON users.username = orders.username`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if len(s.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(s.Joins))
	}
	j := s.Joins[0]
	if j.Kind != JoinInner || j.Table.Name != "orders" {
		t.Errorf("unexpected join: %+v", j)
	}
	if j.Keys["username"] != "username" {
		t.Errorf("unexpected join keys: %v", j.Keys)
	}
}

func TestParseSelect_LeftJoin(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users LEFT JOIN orders ON users.username = orders.username`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if s.Joins[0].Kind != JoinLeft {
		t.Errorf("expected LEFT join, got %v", s.Joins[0].Kind)
	}
}

func TestParseSelect_IsNull(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users WHERE email IS NULL`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if s.Where[0].Op != OpIsNull {
		t.Errorf("expected IS NULL, got %v", s.Where[0].Op)
	}
}

func TestParseSelect_IsNotNull(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users WHERE email IS NOT NULL`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if s.Where[0].Op != OpIsNotNull {
		t.Errorf("expected IS NOT NULL, got %v", s.Where[0].Op)
	}
}

func TestParseSelect_In(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users WHERE age IN (25, 28, 30)`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if s.Where[0].Op != OpIn {
		t.Errorf("expected IN, got %v", s.Where[0].Op)
	}
	vals := s.Where[0].Value.([]any)
	if len(vals) != 3 {
		t.Errorf("expected 3 IN values, got %d", len(vals))
	}
}

func TestParseSelect_Between(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM users WHERE age BETWEEN 20 AND 35`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if s.Where[0].Op != OpBetween {
		t.Errorf("expected BETWEEN, got %v", s.Where[0].Op)
	}
	bounds := s.Where[0].Value.([2]any)
	if bounds[0].(int64) != 20 || bounds[1].(int64) != 35 {
		t.Errorf("unexpected BETWEEN bounds: %v", bounds)
	}
}

func TestParseSelect_NegativeNumber(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM t WHERE amount > -10`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SelectStmt)
	if s.Where[0].Value.(int64) != -10 {
		t.Errorf("expected -10, got %v", s.Where[0].Value)
	}
}

func TestParseSelect_MultipleStatements(t *testing.T) {
	stmts, err := ParseText(`SELECT * FROM a; SELECT * FROM b`)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts))
	}
}

// ── INSERT ────────────────────────────────────────────────────────────────────

func TestParseInsert_Basic(t *testing.T) {
	stmts, err := ParseText(`INSERT INTO users (username, email, age) VALUES ('alice', 'alice@example.com', 30)`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(InsertStmt)
	if s.Table.Name != "users" {
		t.Errorf("unexpected table: %q", s.Table.Name)
	}
	if len(s.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(s.Rows))
	}
	if s.Rows[0]["username"] != "alice" {
		t.Errorf("unexpected username: %v", s.Rows[0]["username"])
	}
	if s.Rows[0]["age"].(int64) != 30 {
		t.Errorf("unexpected age: %v", s.Rows[0]["age"])
	}
}

func TestParseInsert_MultiRow(t *testing.T) {
	stmts, err := ParseText(`INSERT INTO users (username, age) VALUES ('bob', 25), ('carol', 35)`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(InsertStmt)
	if len(s.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(s.Rows))
	}
	if s.Rows[1]["username"] != "carol" {
		t.Errorf("unexpected second row: %v", s.Rows[1])
	}
}

func TestParseInsert_SchemaQualified(t *testing.T) {
	stmts, err := ParseText(`INSERT INTO apps.orders (product, amount) VALUES ('laptop', 1200.50)`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(InsertStmt)
	if s.Table.Schema != "apps" || s.Table.Name != "orders" {
		t.Errorf("unexpected table ref: %+v", s.Table)
	}
	if s.Rows[0]["amount"].(float64) != 1200.50 {
		t.Errorf("unexpected amount: %v", s.Rows[0]["amount"])
	}
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

func TestParseUpdate_Basic(t *testing.T) {
	stmts, err := ParseText(`UPDATE users SET active = FALSE WHERE age < 18`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(UpdateStmt)
	if s.Table.Name != "users" {
		t.Errorf("unexpected table: %q", s.Table.Name)
	}
	if len(s.Assignments) != 1 || s.Assignments[0].Column != "active" {
		t.Errorf("unexpected assignments: %v", s.Assignments)
	}
	if s.Assignments[0].Value.(bool) != false {
		t.Errorf("expected active=FALSE")
	}
	if len(s.Where) != 1 || s.Where[0].Op != OpLt {
		t.Errorf("unexpected where: %v", s.Where)
	}
}

func TestParseUpdate_MultiAssign(t *testing.T) {
	stmts, err := ParseText(`UPDATE users SET active = TRUE, age = 31 WHERE username = 'alice'`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(UpdateStmt)
	if len(s.Assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(s.Assignments))
	}
}

func TestParseUpdate_NoWhere(t *testing.T) {
	stmts, err := ParseText(`UPDATE users SET active = TRUE`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(UpdateStmt)
	if len(s.Where) != 0 {
		t.Errorf("expected no WHERE, got %v", s.Where)
	}
}

// ── DELETE ────────────────────────────────────────────────────────────────────

func TestParseDelete_Basic(t *testing.T) {
	stmts, err := ParseText(`DELETE FROM users WHERE active = FALSE`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(DeleteStmt)
	if s.Table.Name != "users" {
		t.Errorf("unexpected table: %q", s.Table.Name)
	}
	if len(s.Where) != 1 || s.Where[0].Field != "active" {
		t.Errorf("unexpected where: %v", s.Where)
	}
}

func TestParseDelete_NoWhere(t *testing.T) {
	stmts, err := ParseText(`DELETE FROM logs`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(DeleteStmt)
	if len(s.Where) != 0 {
		t.Errorf("expected no WHERE, got %v", s.Where)
	}
}

func TestParseDelete_FloatValue(t *testing.T) {
	stmts, err := ParseText(`DELETE FROM orders WHERE amount < 0.01`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(DeleteStmt)
	if s.Where[0].Value.(float64) != 0.01 {
		t.Errorf("unexpected float: %v", s.Where[0].Value)
	}
}
