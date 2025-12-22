package postgres

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/josefina/jdb"
)

/**
* Query
* @param query *jdb.Ql
* @return (string, error)
**/
func (s *Driver) buildQuery(ql *jdb.Ql) (string, error) {
	if ql.IsDebug {
		logs.Debug("query:", ql.ToJson().ToString())
	}

	sql, err := s.buildSelect(ql)
	if err != nil {
		return "", err
	}

	sql = fmt.Sprintf("SELECT %s", sql)
	def, err := s.buildFrom(ql)
	if err != nil {
		return "", err
	}

	def = fmt.Sprintf("FROM %s", def)
	sql = strs.Append(sql, def, "\n")
	def, err = s.buildJoins(ql)
	if err != nil {
		return "", err
	}

	if def != "" {
		def = fmt.Sprintf("JOIN %s", def)
		sql = strs.Append(sql, def, "\n")
	}

	wheres := ql.Wheres.Conditions
	if len(wheres) > 0 {
		def, err = s.buildWhere(wheres)
		if err != nil {
			return "", err
		}

		if def != "" {
			def = fmt.Sprintf("WHERE %s", def)
			sql = strs.Append(sql, def, "\n")
		}
	}

	def, err = s.buildGroupBy(ql)
	if err != nil {
		return "", err
	}

	if def != "" {
		def = fmt.Sprintf("GROUP BY %s", def)
		sql = strs.Append(sql, def, "\n")
	}

	def, err = s.buildWhere(ql.Havings.Conditions)
	if err != nil {
		return "", err
	}

	if def != "" {
		def = fmt.Sprintf("HAVING %s", def)
		sql = strs.Append(sql, def, "\n")
	}

	def, err = s.buildOrderBy(ql)
	if err != nil {
		return "", err
	}

	if def != "" {
		def = fmt.Sprintf("ORDER BY %s", def)
		sql = strs.Append(sql, def, "\n")
	}

	def, err = s.buildLimit(ql)
	if err != nil {
		return "", err
	}

	if def != "" {
		sql = strs.Append(sql, def, "\n")
	}

	if ql.Type == jdb.TpExists {
		return fmt.Sprintf("SELECT EXISTS(%s);", sql), nil
	} else {
		return fmt.Sprintf("%s;", sql), nil
	}
}

/**
* buildSelect
* @param ql *jdb.Ql
* @return (string, error)
**/
func (s *Driver) buildSelect(ql *jdb.Ql) (string, error) {
	if ql.Type == jdb.TpExists {
		return "", nil
	}

	if ql.Type == jdb.TpCounted {
		return "COUNT(*) AS all", nil
	}

	result := ""
	if ql.Type == jdb.TpData {
		selects := map[string]string{}
		atribs := map[string]string{}
		for _, fld := range ql.Selects {
			if fld.TypeField == jdb.TpColumn {
				selects[fld.As] = fld.AS()
			} else if fld.TypeField == jdb.TpAtrib {
				atribs[fld.As] = fld.AS()
			}
		}

		if len(atribs) == 0 {
			result = fmt.Sprintf("\n%s", jdb.SOURCE)
		} else {
			for k, v := range atribs {
				def := fmt.Sprintf("\n'%s', %s", k, v)
				result = strs.Append(result, def, ", ")
			}

			if result != "" {
				result = fmt.Sprintf("\n\tjsonb_build_object(%s\n)", result)
			}
		}

		if len(selects) == 0 {
			hidden := ql.Hidden
			hidden = append(hidden, jdb.SOURCE)
			def := fmt.Sprintf("to_jsonb(A) - ARRAY[%s]", strings.Join(hidden, ", "))
			result = strs.Append(result, def, "||")
		} else {
			sel := ""
			for k, v := range selects {
				def := fmt.Sprintf("\n'%s',  %s", k, v)
				if v == "" {
					def = fmt.Sprintf("\n'%s',  %s", k, k)
				}
				sel = strs.Append(sel, def, ", ")
			}

			if sel != "" {
				result = fmt.Sprintf("%s||jsonb_build_object(%s\n)", result, sel)
			}
		}

		return fmt.Sprintf("%s AS result", result), nil
	}

	if len(ql.Selects) == 0 {
		hidden := ql.Hidden
		if len(hidden) > 0 {
			result += fmt.Sprintf("to_jsonb(A) - ARRAY[%s]", strings.Join(hidden, ", "))
		} else {
			result += "A.*"
		}
	} else {
		selects := map[string]string{}
		for _, fld := range ql.Selects {
			if fld.TypeField == jdb.TpColumn {
				selects[fld.As] = fld.AS()
			}
		}
		for k, v := range selects {
			def := fmt.Sprintf("\n%s AS %s", v, k)
			if k == v {
				def = fmt.Sprintf("\n%s", v)
			} else if v == "" {
				def = fmt.Sprintf("\n%s", v)
			}
			result = strs.Append(result, def, ", ")
		}
	}

	return result, nil
}

/**
* buildFrom
* @param ql *jdb.Ql
* @return (string, error)
**/
func (s *Driver) buildFrom(ql *jdb.Ql) (string, error) {
	result := ""

	if len(ql.Froms) == 0 {
		return result, fmt.Errorf(jdb.MSG_FROM_REQUIRED)
	}

	for _, from := range ql.Froms {
		as := from.As
		table := from.Model.Table
		def := fmt.Sprintf("%s AS %s", table, as)
		if as == table {
			def = fmt.Sprintf("%s", table)
		}

		result = strs.Append(result, def, ", ")
		break
	}

	return result, nil
}

/**
* buildJoins
* @param ql *jdb.Ql
* @return (string, error)
**/
func (s *Driver) buildJoins(ql *jdb.Ql) (string, error) {
	result := ""

	if len(ql.Joins) == 0 {
		return result, nil
	}

	for _, join := range ql.Joins {
		def := ""
		for k, v := range join.Keys {
			if len(def) == 0 {
				def = fmt.Sprintf("%s AS %s ON %s = %s", join.To.Table, join.As, k, v)
			} else {
				def = fmt.Sprintf("%s AND %s = %s", def, k, v)
			}
		}
		result = strs.Append(result, def, "\nJOIN ")
	}

	return fmt.Sprintf("%s", result), nil
}

func (s *Driver) buildCondition(cond *jdb.Condition) string {
	switch cond.Operator {
	case jdb.OpEq:
		return fmt.Sprintf("%s = %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpNeg:
		return fmt.Sprintf("%s != %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpLess:
		return fmt.Sprintf("%s < %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpLessEq:
		return fmt.Sprintf("%s <= %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpMore:
		return fmt.Sprintf("%s > %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpMoreEq:
		return fmt.Sprintf("%s >= %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpLike:
		return fmt.Sprintf("%s LIKE %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpIn:
		return fmt.Sprintf("%s IN %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpNotIn:
		return fmt.Sprintf("%s NOT IN %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpIs:
		return fmt.Sprintf("%s IS %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpIsNot:
		return fmt.Sprintf("%s IS NOT %v", cond.Field.AS(), jdb.Quoted(cond.Value))
	case jdb.OpNull:
		return fmt.Sprintf("%s IS NULL", cond.Field.AS())
	case jdb.OpNotNull:
		return fmt.Sprintf("%s IS NOT NULL", cond.Field.AS())
	case jdb.OpBetween:
		vals := cond.Value.([]interface{})
		return fmt.Sprintf("%s BETWEEN %v AND %v", cond.Field.AS(), jdb.Quoted(vals[0]), jdb.Quoted(vals[1]))
	case jdb.OpNotBetween:
		vals := cond.Value.([]interface{})
		return fmt.Sprintf("%s NOT BETWEEN %v AND %v", cond.Field.AS(), jdb.Quoted(vals[0]), jdb.Quoted(vals[1]))
	}

	return ""
}

/**
* buildWhere
* @param wheres []jdb.Condition
* @return (string, error)
**/
func (s *Driver) buildWhere(wheres []*jdb.Condition) (string, error) {
	result := ""

	for i, cond := range wheres {
		if i == 0 {
			result = s.buildCondition(cond)
		} else if cond.Connector == jdb.Or {
			result = fmt.Sprintf("%s\nOR %s", result, s.buildCondition(cond))
		} else {
			result = fmt.Sprintf("%s\nAND %s", result, s.buildCondition(cond))
		}
	}

	return result, nil
}

/**
* buildGroupBy
* @param ql *jdb.Ql
* @return (string, error)
**/
func (s *Driver) buildGroupBy(ql *jdb.Ql) (string, error) {
	result := ""

	if len(ql.GroupsBy) == 0 {
		return result, nil
	}

	for _, v := range ql.GroupsBy {
		def := fmt.Sprintf("%s", v.AS())
		result = strs.Append(result, def, ", ")
	}

	return result, nil
}

/**
* buildOrderBy
* @param ql *jdb.Ql
* @return (string, error)
**/
func (s *Driver) buildOrderBy(ql *jdb.Ql) (string, error) {
	result := ""
	for _, fld := range ql.OrdersByAsc {
		result = strs.Append(result, fld.AS(), ", ")
	}

	if result != "" {
		result = fmt.Sprintf(`%s ASC`, result)
	}

	for _, fld := range ql.OrdersByDesc {
		result = strs.Append(result, fld.AS(), ", ")
	}

	if result != "" {
		result = fmt.Sprintf(`%s DESC`, result)
	}

	return result, nil
}

/**
* buildLimit
* @param ql *jdb.Ql
* @return (string, error)
**/
func (s *Driver) buildLimit(ql *jdb.Ql) (string, error) {
	result := ""

	if ql.Rows > ql.MaxRows {
		ql.Rows = ql.MaxRows
	}

	if ql.Page == 0 {
		return fmt.Sprintf("LIMIT %d", ql.Rows), nil
	}

	offset := (ql.Page - 1) * ql.Rows
	result = fmt.Sprintf("%d OFFSET %d", ql.Rows, offset)
	return result, nil
}
