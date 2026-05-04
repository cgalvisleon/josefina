package stmt

import (
	"testing"

	"github.com/cgalvisleon/et/et"
)

func TestParseJson_Select(t *testing.T) {
	q := et.Json{
		"op":     "select",
		"schema": "apps",
		"table":  "users",
		"select": []any{"username", "email"},
		"where": []any{
			map[string]any{"field": "age", "op": "more_eq", "value": float64(25)},
			map[string]any{"field": "active", "op": "eq", "value": true, "connector": "and"},
		},
		"order":  []any{map[string]any{"column": "age", "asc": true}},
		"rows":   float64(10),
		"offset": float64(0),
	}

	stmts, err := ParseJson(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(stmts))
	}
	s := stmts[0].(SelectStmt)

	if s.From.Schema != "apps" || s.From.Name != "users" {
		t.Errorf("unexpected table ref: %+v", s.From)
	}
	if len(s.Columns) != 2 {
		t.Errorf("expected 2 select columns, got %d", len(s.Columns))
	}
	if len(s.Where) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(s.Where))
	}
	if s.Where[0].Op != OpGtEq || s.Where[0].Field != "age" {
		t.Errorf("unexpected first cond: %+v", s.Where[0])
	}
	if s.Where[1].Connector != ConnAnd {
		t.Errorf("expected AND connector")
	}
	if len(s.OrderBy) != 1 || !s.OrderBy[0].Asc {
		t.Errorf("unexpected order: %+v", s.OrderBy)
	}
	if s.Rows != 10 {
		t.Errorf("expected rows=10, got %d", s.Rows)
	}
}

func TestParseJson_Insert(t *testing.T) {
	q := et.Json{
		"op":    "insert",
		"table": "users",
		"rows": []any{
			map[string]any{"username": "alice", "age": float64(30)},
		},
	}
	stmts, err := ParseJson(q)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(InsertStmt)
	if len(s.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(s.Rows))
	}
	if s.Rows[0]["username"] != "alice" {
		t.Errorf("unexpected username: %v", s.Rows[0]["username"])
	}
}

func TestParseJson_Update(t *testing.T) {
	q := et.Json{
		"op":    "update",
		"table": "users",
		"set":   []any{map[string]any{"column": "active", "value": false}},
		"where": []any{map[string]any{"field": "age", "op": "less", "value": float64(18)}},
	}
	stmts, err := ParseJson(q)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(UpdateStmt)
	if len(s.Assignments) != 1 || s.Assignments[0].Column != "active" {
		t.Errorf("unexpected assignments: %v", s.Assignments)
	}
	if len(s.Where) != 1 || s.Where[0].Op != OpLt {
		t.Errorf("unexpected where: %v", s.Where)
	}
}

func TestParseJson_Delete(t *testing.T) {
	q := et.Json{
		"op":    "delete",
		"table": "logs",
		"where": []any{map[string]any{"field": "active", "op": "eq", "value": false}},
	}
	stmts, err := ParseJson(q)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(DeleteStmt)
	if s.Table.Name != "logs" {
		t.Errorf("unexpected table: %q", s.Table.Name)
	}
	if len(s.Where) != 1 {
		t.Errorf("expected 1 condition, got %d", len(s.Where))
	}
}

func TestParseJson_MissingOp(t *testing.T) {
	_, err := ParseJson(et.Json{"table": "users"})
	if err == nil {
		t.Error("expected error for missing op")
	}
}

func TestParseJson_UnknownOp(t *testing.T) {
	_, err := ParseJson(et.Json{"op": "upsert", "table": "users"})
	if err == nil {
		t.Error("expected error for unknown op")
	}
}
