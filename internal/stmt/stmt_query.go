package stmt

// ── Shared primitives ─────────────────────────────────────────────────────────

// TableRef: qualified table reference schema.name.
// Schema is optional; empty means the default schema.
type TableRef struct {
	Schema string
	Name   string
}

// ColumnRef: optional table qualifier + column name.
type ColumnRef struct {
	Table  string // empty when unqualified
	Column string
}

// ── Literal values ────────────────────────────────────────────────────────────

// LiteralKind: category of a constant value in SQL.
type LiteralKind int

const (
	LitString  LiteralKind = iota // 'text'
	LitInteger                    // 42
	LitFloat                      // 3.14
	LitBoolean                    // TRUE / FALSE
	LitNull                       // NULL
)

// Literal: a constant SQL value with its parsed Go representation.
type Literal struct {
	Kind  LiteralKind
	Raw   string // original source text
	Value any    // string | int64 | float64 | bool | nil
}

// ── WHERE conditions ──────────────────────────────────────────────────────────

// CondOp: comparison operator in a WHERE predicate.
// Maps 1-to-1 to jdb.Eq / jdb.More / jdb.Like / etc.
type CondOp string

const (
	OpEq         CondOp = "="
	OpNeq        CondOp = "!="
	OpLt         CondOp = "<"
	OpLtEq       CondOp = "<="
	OpGt         CondOp = ">"
	OpGtEq       CondOp = ">="
	OpLike       CondOp = "LIKE"
	OpILike      CondOp = "ILIKE"
	OpIn         CondOp = "IN"
	OpNotIn      CondOp = "NOT IN"
	OpIsNull     CondOp = "IS NULL"
	OpIsNotNull  CondOp = "IS NOT NULL"
	OpBetween    CondOp = "BETWEEN"
	OpNotBetween CondOp = "NOT BETWEEN"
)

// Connector: logical connector between successive WHERE conditions.
type Connector int

const (
	ConnNone Connector = iota // first condition in the list
	ConnAnd
	ConnOr
)

// CondExpr: a single predicate inside a WHERE clause.
//
//	field op value            → standard comparison
//	field IS NULL             → Op = OpIsNull,    Value = nil
//	field BETWEEN v1 AND v2   → Op = OpBetween,   Value = [2]any{v1, v2}
//	field IN (v1, v2, ...)    → Op = OpIn,        Value = []any{...}
type CondExpr struct {
	Connector Connector
	Field     string // column name (unqualified)
	Op        CondOp
	Value     any // Literal.Value, []any (IN), [2]any (BETWEEN), or nil
}

// ── ORDER BY ──────────────────────────────────────────────────────────────────

// OrderItem: one element of an ORDER BY clause.
type OrderItem struct {
	Column string
	Asc    bool // true = ASC (default), false = DESC
}

// ── JOIN ──────────────────────────────────────────────────────────────────────

// JoinKind: type of SQL join.
type JoinKind int

const (
	JoinInner JoinKind = iota
	JoinLeft
	JoinRight
	JoinFull
)

// JoinClause: JOIN table ON from_field = to_field, ...
type JoinClause struct {
	Kind  JoinKind
	Table TableRef
	Keys  map[string]string // from_col → to_col
}

// ── DML statements ────────────────────────────────────────────────────────────

// SelectStmt: SELECT [cols] FROM table [JOIN] [WHERE] [ORDER BY] [LIMIT/OFFSET]
type SelectStmt struct {
	Columns []string     // empty slice means SELECT *
	From    TableRef
	Joins   []JoinClause
	Where   []CondExpr
	OrderBy []OrderItem
	Page    int // >0 enables page-based pagination together with Rows
	Rows    int // max rows per page (0 = no limit)
	Offset  int // raw offset when Page == 0
}

func (SelectStmt) stmt() {}

// InsertStmt: INSERT INTO table [(cols)] VALUES (vals), ...
// Columns is empty when the parser uses the positional form.
// Each element of Rows corresponds to one VALUES tuple.
type InsertStmt struct {
	Table   TableRef
	Columns []string // may be empty (executor uses field names from Rows)
	Rows    []map[string]any
}

func (InsertStmt) stmt() {}

// UpsertStmt: UPSERT INTO table [(cols)] VALUES (vals), ...
// Identical shape to InsertStmt; the executor calls model.Upsert instead of model.Insert.
type UpsertStmt struct {
	Table   TableRef
	Columns []string
	Rows    []map[string]any
}

func (UpsertStmt) stmt() {}

// Assignment: one col = val pair inside a SET clause.
type Assignment struct {
	Column string
	Value  any // Literal.Value
}

// UpdateStmt: UPDATE table SET col=val,... [WHERE ...]
type UpdateStmt struct {
	Table       TableRef
	Assignments []Assignment
	Where       []CondExpr
}

func (UpdateStmt) stmt() {}

// DeleteStmt: DELETE FROM table [WHERE ...]
type DeleteStmt struct {
	Table TableRef
	Where []CondExpr
}

func (DeleteStmt) stmt() {}
