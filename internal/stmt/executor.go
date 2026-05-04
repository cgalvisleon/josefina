package stmt

import (
	"fmt"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/jdb"
)

// ── Entry point ───────────────────────────────────────────────────────────────

/**
* ExecSQL: parses a SQL string and executes all statements against the given database.
* @param db *jdb.DB
* @param sql string
* @return et.Items, error
**/
func ExecSQL(db *jdb.DB, sql string) (et.Items, error) {
	stmts, err := ParseText(sql)
	if err != nil {
		return et.Items{}, err
	}
	return Exec(db, stmts)
}

/**
* Exec: executes a list of parsed statements against the given database.
* DML and SELECT statements accumulate results; DDL and cache commands return
* an empty Items on success.
* @param db *jdb.DB
* @param stmts []Stmt
* @return et.Items, error
**/
func Exec(db *jdb.DB, stmts []Stmt) (et.Items, error) {
	result := et.Items{}
	for _, st := range stmts {
		items, err := execOne(db, st)
		if err != nil {
			return et.Items{}, err
		}
		result.AddMany(items.Result)
	}
	return result, nil
}

func execOne(db *jdb.DB, st Stmt) (et.Items, error) {
	switch s := st.(type) {
	// ── DML ───────────────────────────────────────────────────────────────────
	case SelectStmt:
		return execSelect(db, s)
	case InsertStmt:
		return execInsert(db, s)
	case UpsertStmt:
		return execUpsert(db, s)
	case UpdateStmt:
		return execUpdate(db, s)
	case DeleteStmt:
		return execDelete(db, s)
	// ── DDL ───────────────────────────────────────────────────────────────────
	case CreateTableStmt:
		return execCreateTable(db, s)
	case DropTableStmt:
		return execDropTable(db, s)
	case AlterTableStmt:
		return execAlterTable(db, s)
	// ── Cache ─────────────────────────────────────────────────────────────────
	case SetCacheStmt:
		return execSetCache(db, s)
	case GetCacheStmt:
		return execGetCache(db, s)
	case DelCacheStmt:
		return execDelCache(db, s)
	case ExistCacheStmt:
		return execExistCache(db, s)
	default:
		return et.Items{}, fmt.Errorf("executor: unsupported statement type %T", st)
	}
}

// ── DML: SELECT ───────────────────────────────────────────────────────────────

func execSelect(db *jdb.DB, s SelectStmt) (et.Items, error) {
	model, err := db.GetModel(s.From.Schema, s.From.Name)
	if err != nil {
		return et.Items{}, err
	}

	w := jdb.From(model)

	if len(s.Columns) > 0 {
		w.Selects(s.Columns...)
	}

	w = applyWhere(w, s.Where)

	for _, j := range s.Joins {
		joinModel, err := db.GetModel(j.Table.Schema, j.Table.Name)
		if err != nil {
			return et.Items{}, err
		}
		switch j.Kind {
		case JoinInner:
			w.InnerJoin(joinModel, j.Keys)
		case JoinLeft:
			w.LeftJoin(joinModel, j.Keys)
		case JoinRight:
			w.RightJoin(joinModel, j.Keys)
		case JoinFull:
			w.FullJoin(joinModel, j.Keys)
		}
	}

	for _, ob := range s.OrderBy {
		if ob.Asc {
			w.Asc(ob.Column)
		} else {
			w.Desc(ob.Column)
		}
	}

	if s.Page > 0 && s.Rows > 0 {
		w.Limit(s.Page, s.Rows)
	} else if s.Rows > 0 && s.Offset > 0 {
		w.SetOffset(s.Offset, s.Rows)
	} else if s.Rows > 0 {
		w.Limit(1, s.Rows)
	}

	return w.All()
}

// ── DML: INSERT ───────────────────────────────────────────────────────────────

func execInsert(db *jdb.DB, s InsertStmt) (et.Items, error) {
	model, err := db.GetModel(s.Table.Schema, s.Table.Name)
	if err != nil {
		return et.Items{}, err
	}
	result := et.Items{}
	for _, row := range s.Rows {
		items, err := model.Insert(et.Json(row)).Exec()
		if err != nil {
			return et.Items{}, err
		}
		result.AddMany(items.Result)
	}
	return result, nil
}

// ── DML: UPSERT ───────────────────────────────────────────────────────────────

func execUpsert(db *jdb.DB, s UpsertStmt) (et.Items, error) {
	model, err := db.GetModel(s.Table.Schema, s.Table.Name)
	if err != nil {
		return et.Items{}, err
	}
	result := et.Items{}
	for _, row := range s.Rows {
		items, err := model.Upsert(et.Json(row)).Exec()
		if err != nil {
			return et.Items{}, err
		}
		result.AddMany(items.Result)
	}
	return result, nil
}

// ── DML: UPDATE ───────────────────────────────────────────────────────────────

func execUpdate(db *jdb.DB, s UpdateStmt) (et.Items, error) {
	model, err := db.GetModel(s.Table.Schema, s.Table.Name)
	if err != nil {
		return et.Items{}, err
	}
	data := make(et.Json, len(s.Assignments))
	for _, a := range s.Assignments {
		data[a.Column] = a.Value
	}
	cmd := model.Update(data)
	applyCommandWhere(cmd, s.Where)
	return cmd.Exec()
}

// ── DML: DELETE ───────────────────────────────────────────────────────────────

func execDelete(db *jdb.DB, s DeleteStmt) (et.Items, error) {
	model, err := db.GetModel(s.Table.Schema, s.Table.Name)
	if err != nil {
		return et.Items{}, err
	}
	cmd := model.Delete()
	applyCommandWhere(cmd, s.Where)
	return cmd.Exec()
}

// ── DDL: CREATE TABLE ─────────────────────────────────────────────────────────

func execCreateTable(db *jdb.DB, s CreateTableStmt) (et.Items, error) {
	fields := make(map[string]jdb.DField, len(s.Columns))
	for _, col := range s.Columns {
		fields[col.Name] = jdb.DField{
			Type:    sqlTypeToTypeData(col.Type),
			Default: col.Default,
		}
	}

	required := make([]jdb.DIndex, 0, len(s.Required))
	for _, r := range s.Required {
		required = append(required, jdb.DIndex{Name: r})
	}
	unique := make([]jdb.DIndex, 0, len(s.Unique))
	for _, u := range s.Unique {
		unique = append(unique, jdb.DIndex{Name: u})
	}

	model, err := db.Define(jdb.DModel{
		Schema:      s.Table.Schema,
		Name:        s.Table.Name,
		Fields:      fields,
		PrimaryKeys: s.PrimaryKeys,
		Unique:      unique,
		Required:    required,
	})
	if err != nil {
		if s.IfNotExists {
			return et.Items{}, nil
		}
		return et.Items{}, err
	}
	return et.Items{}, model.Init()
}

// ── DDL: DROP TABLE ───────────────────────────────────────────────────────────

func execDropTable(db *jdb.DB, s DropTableStmt) (et.Items, error) {
	err := db.DeleteModel(s.Table.Schema, s.Table.Name)
	if err != nil && s.IfExists {
		return et.Items{}, nil
	}
	return et.Items{}, err
}

// ── DDL: ALTER TABLE ──────────────────────────────────────────────────────────

func execAlterTable(db *jdb.DB, s AlterTableStmt) (et.Items, error) {
	model, err := db.GetModel(s.Table.Schema, s.Table.Name)
	if err != nil {
		return et.Items{}, err
	}

	switch s.Action {
	case AlterAddColumn:
		tp := sqlTypeToTypeData(s.Column.Type)
		if _, err = model.DefineField(s.Column.Name, tp, s.Column.Default); err != nil {
			return et.Items{}, err
		}
		if s.Column.NotNull {
			if _, err = model.DefineRequired(s.Column.Name); err != nil {
				return et.Items{}, err
			}
		}
		if s.Column.Unique {
			if _, err = model.DefineUnique(s.Column.Name); err != nil {
				return et.Items{}, err
			}
		}

	case AlterDropColumn:
		return et.Items{}, fmt.Errorf("ALTER TABLE DROP COLUMN is not supported by this engine")

	case AlterAlterColumn:
		if s.Column.Default != nil {
			tp := sqlTypeToTypeData(s.Column.Type)
			if tp == "" {
				tp = jdb.TpAny
			}
			if _, err = model.DefineField(s.Column.Name, tp, s.Column.Default); err != nil {
				return et.Items{}, err
			}
		}
		if s.Column.NotNull {
			if _, err = model.DefineRequired(s.Column.Name); err != nil {
				return et.Items{}, err
			}
		}

	case AlterAddPrimaryKey:
		cols := strings.Split(s.Column.Name, ",")
		for i := range cols {
			cols[i] = strings.TrimSpace(cols[i])
		}
		if err = model.DefinePrimaryKeys(cols...); err != nil {
			return et.Items{}, err
		}

	case AlterAddUnique:
		cols := strings.Split(s.Column.Name, ",")
		for _, col := range cols {
			col = strings.TrimSpace(col)
			if _, err = model.DefineUnique(col); err != nil {
				return et.Items{}, err
			}
		}
	}

	return et.Items{}, nil
}

// ── Cache ─────────────────────────────────────────────────────────────────────

func execSetCache(db *jdb.DB, s SetCacheStmt) (et.Items, error) {
	d := time.Duration(s.Duration) * time.Second
	return et.Items{}, db.SetCache(s.Key, s.Value, d)
}

func execGetCache(db *jdb.DB, s GetCacheStmt) (et.Items, error) {
	var val any
	found, err := db.GetCache(s.Key, &val)
	if err != nil {
		return et.Items{}, err
	}
	result := et.Items{}
	if found {
		result.Add(et.Json{"key": s.Key, "value": val})
	}
	return result, nil
}

func execDelCache(db *jdb.DB, s DelCacheStmt) (et.Items, error) {
	return et.Items{}, db.DeleteCache(s.Key)
}

func execExistCache(db *jdb.DB, s ExistCacheStmt) (et.Items, error) {
	var val any
	found, err := db.GetCache(s.Key, &val)
	if err != nil {
		return et.Items{}, err
	}
	result := et.Items{}
	result.Add(et.Json{"key": s.Key, "exists": found})
	return result, nil
}

// ── WHERE helpers ─────────────────────────────────────────────────────────────

func applyWhere(w *jdb.Where, conds []CondExpr) *jdb.Where {
	for i, c := range conds {
		cond := condToEt(c)
		if i == 0 {
			w.Where(cond)
		} else if c.Connector == ConnOr {
			w.Or(cond)
		} else {
			w.And(cond)
		}
	}
	return w
}

func applyCommandWhere(cmd *jdb.Command, conds []CondExpr) {
	for i, c := range conds {
		cond := condToEt(c)
		if i == 0 {
			cmd.Where(cond)
		} else if c.Connector == ConnOr {
			cmd.Or(cond)
		} else {
			cmd.And(cond)
		}
	}
}

func condToEt(c CondExpr) *et.Condition {
	switch c.Op {
	case OpEq:
		return jdb.Eq(c.Field, c.Value)
	case OpNeq:
		return jdb.Neg(c.Field, c.Value)
	case OpLt:
		return jdb.Less(c.Field, c.Value)
	case OpLtEq:
		return jdb.LessEq(c.Field, c.Value)
	case OpGt:
		return jdb.More(c.Field, c.Value)
	case OpGtEq:
		return jdb.MoreEq(c.Field, c.Value)
	case OpLike, OpILike:
		return jdb.Like(c.Field, c.Value)
	case OpIn:
		return jdb.In(c.Field, toIfaceSlice(c.Value))
	case OpNotIn:
		return jdb.NotIn(c.Field, toIfaceSlice(c.Value))
	case OpIsNull:
		return jdb.Null(c.Field)
	case OpIsNotNull:
		return jdb.NotNull(c.Field)
	case OpBetween:
		b := c.Value.([2]any)
		return jdb.Between(c.Field, b[0], b[1])
	case OpNotBetween:
		b := c.Value.([2]any)
		return jdb.NotBetween(c.Field, b[0], b[1])
	}
	return jdb.Eq(c.Field, c.Value)
}

func toIfaceSlice(v any) []interface{} {
	if s, ok := v.([]any); ok {
		out := make([]interface{}, len(s))
		copy(out, s)
		return out
	}
	return nil
}

// ── Type mapping ──────────────────────────────────────────────────────────────

func sqlTypeToTypeData(t SqlType) jdb.TypeData {
	switch t {
	case SqlText, SqlVarchar:
		return jdb.TpText
	case SqlInt, SqlBigInt, SqlSmallInt:
		return jdb.TpInt
	case SqlFloat, SqlReal, SqlNumeric:
		return jdb.TpFloat
	case SqlBool:
		return jdb.TpBoolean
	case SqlJson, SqlJsonb:
		return jdb.TpJson
	case SqlTimestamp, SqlDate:
		return jdb.TpDateTime
	case SqlBytes:
		return jdb.TpBytes
	case SqlKey:
		return jdb.TpKey
	default:
		return jdb.TpAny
	}
}
