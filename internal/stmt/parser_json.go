package stmt

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
)

// ParseJson converts a JSON query document into a list of statements.
//
// Supported shapes:
//
//	SELECT  {"op":"select","schema":"s","table":"t","select":["a","b"],
//	         "where":[{"field":"f","op":"eq","value":1}],
//	         "order":[{"column":"f","asc":true}],"rows":10,"offset":0}
//
//	INSERT  {"op":"insert","schema":"s","table":"t",
//	         "rows":[{"col":val,...}]}
//
//	UPDATE  {"op":"update","schema":"s","table":"t",
//	         "set":[{"column":"f","value":v}],
//	         "where":[...]}
//
//	DELETE  {"op":"delete","schema":"s","table":"t","where":[...]}
/**
* ParseJson: parses a JSON query document into statements.
* @param input et.Json
* @return []Stmt, error
**/
func ParseJson(input et.Json) ([]Stmt, error) {
	op := strings.ToUpper(input.Str("op"))
	if op == "" {
		return nil, fmt.Errorf("json query: missing 'op' field")
	}

	tbl := TableRef{
		Schema: input.Str("schema"),
		Name:   input.Str("table"),
	}
	if tbl.Name == "" {
		return nil, fmt.Errorf("json query: missing 'table' field")
	}

	switch op {
	case "SELECT":
		st, err := jsonParseSelect(tbl, input)
		if err != nil {
			return nil, err
		}
		return []Stmt{st}, nil
	case "INSERT":
		st, err := jsonParseInsert(tbl, input)
		if err != nil {
			return nil, err
		}
		return []Stmt{st}, nil
	case "UPSERT":
		st, err := jsonParseUpsert(tbl, input)
		if err != nil {
			return nil, err
		}
		return []Stmt{st}, nil
	case "UPDATE":
		st, err := jsonParseUpdate(tbl, input)
		if err != nil {
			return nil, err
		}
		return []Stmt{st}, nil
	case "DELETE":
		st, err := jsonParseDelete(tbl, input)
		if err != nil {
			return nil, err
		}
		return []Stmt{st}, nil
	default:
		return nil, fmt.Errorf("json query: unknown op %q", op)
	}
}

func jsonParseSelect(tbl TableRef, q et.Json) (SelectStmt, error) {
	st := SelectStmt{From: tbl}

	if cols, ok := q["select"].([]any); ok {
		for _, c := range cols {
			if s, ok := c.(string); ok {
				st.Columns = append(st.Columns, s)
			}
		}
	}

	where, err := jsonParseWhere(q)
	if err != nil {
		return st, err
	}
	st.Where = where

	if order, ok := q["order"].([]any); ok {
		for _, o := range order {
			if m, ok := o.(map[string]any); ok {
				col, _ := m["column"].(string)
				asc := true
				if a, ok := m["asc"].(bool); ok {
					asc = a
				}
				if col != "" {
					st.OrderBy = append(st.OrderBy, OrderItem{Column: col, Asc: asc})
				}
			}
		}
	}

	if rows, ok := q["rows"].(float64); ok {
		st.Rows = int(rows)
	}
	if offset, ok := q["offset"].(float64); ok {
		st.Offset = int(offset)
	}
	if page, ok := q["page"].(float64); ok {
		st.Page = int(page)
	}

	return st, nil
}

func jsonParseInsert(tbl TableRef, q et.Json) (InsertStmt, error) {
	rows, err := jsonParseRows(q)
	if err != nil {
		return InsertStmt{}, fmt.Errorf("json insert: %w", err)
	}
	return InsertStmt{Table: tbl, Rows: rows}, nil
}

func jsonParseUpsert(tbl TableRef, q et.Json) (UpsertStmt, error) {
	rows, err := jsonParseRows(q)
	if err != nil {
		return UpsertStmt{}, fmt.Errorf("json upsert: %w", err)
	}
	return UpsertStmt{Table: tbl, Rows: rows}, nil
}

func jsonParseRows(q et.Json) ([]map[string]any, error) {
	rawRows, ok := q["rows"].([]any)
	if !ok {
		return nil, fmt.Errorf("'rows' must be an array")
	}
	rows := make([]map[string]any, 0, len(rawRows))
	for _, r := range rawRows {
		if m, ok := r.(map[string]any); ok {
			row := make(map[string]any, len(m))
			for k, v := range m {
				row[k] = v
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func jsonParseUpdate(tbl TableRef, q et.Json) (UpdateStmt, error) {
	st := UpdateStmt{Table: tbl}

	if set, ok := q["set"].([]any); ok {
		for _, a := range set {
			if m, ok := a.(map[string]any); ok {
				col, _ := m["column"].(string)
				val := m["value"]
				if col != "" {
					st.Assignments = append(st.Assignments, Assignment{Column: col, Value: val})
				}
			}
		}
	}

	where, err := jsonParseWhere(q)
	if err != nil {
		return st, err
	}
	st.Where = where

	return st, nil
}

func jsonParseDelete(tbl TableRef, q et.Json) (DeleteStmt, error) {
	st := DeleteStmt{Table: tbl}

	where, err := jsonParseWhere(q)
	if err != nil {
		return st, err
	}
	st.Where = where

	return st, nil
}

// jsonParseWhere converts the "where" array in a JSON query into []CondExpr.
// Each element: {"field":"f","op":"eq","value":v,"connector":"and"}
func jsonParseWhere(q et.Json) ([]CondExpr, error) {
	raw, ok := q["where"].([]any)
	if !ok {
		return nil, nil
	}

	conds := make([]CondExpr, 0, len(raw))
	for i, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("json where[%d]: expected object", i)
		}

		field, _ := m["field"].(string)
		if field == "" {
			return nil, fmt.Errorf("json where[%d]: missing 'field'", i)
		}

		opStr, _ := m["op"].(string)
		op, err := jsonCondOp(opStr)
		if err != nil {
			return nil, fmt.Errorf("json where[%d]: %w", i, err)
		}

		connStr, _ := m["connector"].(string)
		var conn Connector
		switch strings.ToLower(connStr) {
		case "and":
			conn = ConnAnd
		case "or":
			conn = ConnOr
		default:
			conn = ConnNone
		}

		conds = append(conds, CondExpr{
			Connector: conn,
			Field:     field,
			Op:        op,
			Value:     m["value"],
		})
	}
	return conds, nil
}

// jsonCondOp maps JSON operator strings to CondOp constants.
// Accepts both josefina internal names (eq, less, more_eq, …) and
// SQL-style names (=, <, >=, like, is_null, …).
func jsonCondOp(s string) (CondOp, error) {
	switch strings.ToLower(s) {
	case "eq", "=":
		return OpEq, nil
	case "neg", "neq", "!=", "<>":
		return OpNeq, nil
	case "less", "<":
		return OpLt, nil
	case "less_eq", "<=":
		return OpLtEq, nil
	case "more", ">":
		return OpGt, nil
	case "more_eq", ">=":
		return OpGtEq, nil
	case "like":
		return OpLike, nil
	case "ilike":
		return OpILike, nil
	case "in":
		return OpIn, nil
	case "not_in":
		return OpNotIn, nil
	case "is_null", "null":
		return OpIsNull, nil
	case "not_null", "is_not_null":
		return OpIsNotNull, nil
	case "between":
		return OpBetween, nil
	case "not_between":
		return OpNotBetween, nil
	default:
		return "", fmt.Errorf("unknown op %q", s)
	}
}
