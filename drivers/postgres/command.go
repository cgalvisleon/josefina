package postgres

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/josefina/jdb"
)

/**
* buildCommand
* @param cmd *jdb.Cmd
* @return (string, error)
**/
func (s *Driver) buildCommand(cmd *jdb.Cmd) (string, error) {
	switch cmd.Type {
	case jdb.TypeInsert:
		return s.buildInsert(cmd)
	case jdb.TypeUpdate:
		return s.buildUpdate(cmd)
	case jdb.TypeDelete:
		return s.buildDelete(cmd)
	}

	return "", nil
}

/**
* buildInsert
* @param cmd *jdb.Cmd
* @return (string, error)
**/
func (s *Driver) buildInsert(cmd *jdb.Cmd) (string, error) {
	from := cmd.Model
	table := from.Table
	data := cmd.New
	into := ""
	values := ""
	useAtribs := from.SourceField != nil && !from.IsLocked
	atribs := et.Json{}
	returning := fmt.Sprintf(`to_jsonb(%s.*) AS result`, table)
	for k, v := range data {
		col := from.FindColumn(k)
		if col != nil && col.TypeColumn == jdb.TpColumn {
			val := fmt.Sprintf(`%v`, jdb.Quoted(v))
			into = strs.Append(into, k, ", ")
			values = strs.Append(values, val, ", ")
			continue
		}

		if useAtribs {
			atribs[k] = v
		}
	}

	if useAtribs {
		into = strs.Append(into, from.SourceField.Name, ", ")
		values = strs.Append(values, fmt.Sprintf(`'%v'::jsonb`, atribs.ToString()), ", ")
		returning = fmt.Sprintf("to_jsonb(%s.*) - '%s' AS result", table, from.SourceField.Name)
	}

	sql := fmt.Sprintf("INSERT INTO %s(%s)\nVALUES(%s)\nRETURNING %s;", table, into, values, returning)
	return sql, nil
}

/**
* buildUpdate
* @param cmd *jdb.Cmd
* @return (string, error)
**/
func (s *Driver) buildUpdate(cmd *jdb.Cmd) (string, error) {
	from := cmd.Model
	table := from.Table
	data := cmd.New
	sets := ""
	atribs := ""
	where := ""
	returning := fmt.Sprintf(`to_jsonb(%s.*) AS result`, table)
	useAtribs := from.SourceField != nil && !from.IsLocked
	for k, v := range data {
		col := from.FindColumn(k)
		if col != nil && col.TypeColumn == jdb.TpColumn {
			val := fmt.Sprintf(`%v`, jdb.Quoted(v))
			sets = strs.Append(sets, fmt.Sprintf(`%s = %s`, k, val), ",\n")
			continue
		}

		if useAtribs {
			val := fmt.Sprintf(`%v`, jdb.Literal(v))
			if len(atribs) == 0 {
				atribs = fmt.Sprintf("COALESCE(%s, '{}')", from.SourceField.Name)
				atribs = strs.Format("jsonb_set(%s, '{%s}', '%v'::jsonb, true)", atribs, k, val)
			} else {
				atribs = strs.Format("jsonb_set(\n%s, \n'{%s}', '%v'::jsonb, true)", atribs, k, val)
			}
		}
	}

	if useAtribs {
		if len(atribs) > 0 {
			sets = strs.Append(sets, fmt.Sprintf(`%s = %s`, from.SourceField.Name, atribs), ",")
		}
		returning = fmt.Sprintf("to_jsonb(%s.*) - '%s' AS result", table, from.SourceField.Name)
	}

	if len(cmd.Wheres.Conditions) > 0 {
		def, err := s.buildWhere(cmd.Wheres.Conditions)
		if err != nil {
			return "", err
		}

		where = def
	}

	sql := fmt.Sprintf("UPDATE %s SET\n%s", table, sets)
	sql = strs.Append(sql, where, "\nWHERE ")
	sql = fmt.Sprintf("%s\nRETURNING %s;", sql, returning)
	return sql, nil
}

/**
* buildDelete
* @param cmd *jdb.Cmd
* @return (string, error)
**/
func (s *Driver) buildDelete(cmd *jdb.Cmd) (string, error) {
	from := cmd.Model
	table := from.Table
	where := ""
	useAtribs := from.SourceField != nil && !from.IsLocked
	returning := fmt.Sprintf(`to_jsonb(%s.*) AS result`, table)
	if len(cmd.Wheres.Conditions) > 0 {
		def, err := s.buildWhere(cmd.Wheres.Conditions)
		if err != nil {
			return "", err
		}

		where = def
	}

	if useAtribs {
		returning = fmt.Sprintf("to_jsonb(%s.*) - '%s' AS result", table, from.SourceField.Name)
	}

	sql := fmt.Sprintf(`DELETE FROM %s`, table)
	sql = strs.Append(sql, where, "\nWHERE ")
	sql = fmt.Sprintf(`%s\nRETURNING %s;`, sql, returning)
	return sql, nil
}
