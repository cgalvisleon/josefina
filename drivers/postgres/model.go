package postgres

import (
	"fmt"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/jdb"
)

/**
* buildModel
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) buildModel(model *jdb.Model) (string, error) {
	if model.IsDebug {
		logs.Debug("model:", model.ToJson().ToString())
	}

	exists, err := ExistTable(s.database.Db, model.Schema, model.Name)
	if err != nil {
		return "", err
	}

	if exists {
		return "", nil
	}

	sql, err := s.buildSchema(model)
	if err != nil {
		return "", err
	}

	def, err := s.buildTable(model)
	if err != nil {
		return "", err
	}

	if def != "" {
		sql = strs.Append(sql, def, "\n")
	}

	def, err = s.buildIndexes(model)
	if err != nil {
		return "", err
	}

	if def != "" {
		sql = strs.Append(sql, def, "\n")
	}

	def, err = s.buildUniqueIndex(model)
	if err != nil {
		return "", err
	}

	if def != "" {
		sql = strs.Append(sql, def, "\n")
	}

	def, err = s.buildTriggerBeforeInsert(model)
	if err != nil {
		return "", err
	}

	if def != "" {
		sql = strs.Append(sql, def, "\n")
	}

	return sql, nil
}

/**
* buildSchema
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) buildSchema(model *jdb.Model) (string, error) {
	if !utility.ValidStr(model.Schema, 0, []string{}) {
		return "", fmt.Errorf(MSG_ATRIB_REQUIRED, "schema")
	}

	exist, err := ExistSchema(s.database.Db, model.Schema)
	if err != nil {
		return "", err
	}

	if exist {
		return "", nil
	}

	return fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", model.Schema), nil
}

/**
* buildTable
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) buildTable(model *jdb.Model) (string, error) {
	getType := func(tp jdb.TypeData) string {
		types := map[jdb.TypeData]string{
			jdb.TpAny:      "VARCHAR(250)",
			jdb.TpBytes:    "BYTEA",
			jdb.TpInt:      "BIGINT",
			jdb.TpFloat:    "DOUBLE PRECISION",
			jdb.TpKey:      "VARCHAR(80)",
			jdb.TpText:     "VARCHAR(250)",
			jdb.TpMemo:     "TEXT",
			jdb.TpJson:     "JSONB",
			jdb.TpDateTime: "TIMESTAMP",
			jdb.TpBoolean:  "BOOLEAN",
			jdb.TpGeometry: "JSONB",
		}

		if t, ok := types[tp]; ok {
			return t
		}

		return "VARCHAR(250)"
	}

	defaultValue := func(tp jdb.TypeData) string {
		values := map[jdb.TypeData]string{
			jdb.TpAny:      "",
			jdb.TpBytes:    "''",
			jdb.TpInt:      "0",
			jdb.TpFloat:    "0.0",
			jdb.TpKey:      "''",
			jdb.TpText:     "''",
			jdb.TpMemo:     "''",
			jdb.TpJson:     "'{}'",
			jdb.TpDateTime: "NOW()",
			jdb.TpBoolean:  "FALSE",
			jdb.TpGeometry: "'{}'",
		}

		if t, ok := values[tp]; ok {
			return t
		}

		return ""
	}

	columns := model.Columns
	columnsDef := ""
	for _, column := range columns {
		tpColumn := column.TypeColumn
		if tpColumn != jdb.TpColumn {
			continue
		}
		tpData := column.TypeData
		tp := getType(tpData)
		df := column.Default
		if df == nil {
			df = defaultValue(tpData)
		}
		def := fmt.Sprintf("\n\t%s %s DEFAULT %s", column.Name, tp, df)
		columnsDef = strs.Append(columnsDef, def, ",")
	}

	def, err := s.buildPrimaryKeys(model)
	if err != nil {
		return "", err
	}

	if def != "" {
		columnsDef = strs.Append(columnsDef, def, ",\n\t")
	}

	table := model.Table
	result := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", table, columnsDef)

	return result, nil
}

/**
* buildPrimaryKeys
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) buildPrimaryKeys(model *jdb.Model) (string, error) {
	if len(model.PrimaryKeys) == 0 {
		return "", nil
	}

	columns := ""
	for _, v := range model.PrimaryKeys {
		columns = strs.Append(columns, v, ", ")
	}

	name := model.Name
	result := fmt.Sprintf("CONSTRAINT pk_%s PRIMARY KEY (%s)", name, columns)

	return result, nil
}

/**
* buildForeignKeys
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) buildForeignKeys(model *jdb.Model) (string, error) {
	if len(model.Master) == 0 {
		return "", nil
	}

	result := ""
	for name, foreignKey := range model.Master {
		to := foreignKey.To.Table
		fks := ""
		ks := ""
		for k, fk := range foreignKey.Keys {
			fks = strs.Append(fks, fmt.Sprintf("%s", k), ", ")
			ks = strs.Append(ks, fmt.Sprintf("%s", fk), ", ")
		}
		def := fmt.Sprintf("ALTER TABLE IF EXISTS %s ADD CONSTRAINT fk_%s_%s FOREIGN KEY(%s) REFERENCES %s(%s)", model.Table, model.Name, name, fks, to, ks)
		onDelete := foreignKey.OnDeleteCascade
		onUpdate := foreignKey.OnUpdateCascade
		if onDelete {
			def = strs.Append(def, fmt.Sprintf("ON DELETE CASCADE"), " ")
		}
		if onUpdate {
			def = strs.Append(def, fmt.Sprintf("ON UPDATE CASCADE"), " ")
		}
		def += ";"
		result = strs.Append(result, def, "\n")
	}

	return result, nil
}

/**
* buildIndexes
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) buildIndexes(model *jdb.Model) (string, error) {
	if len(model.Indexes) == 0 {
		return "", nil
	}

	table := model.Table
	name := model.Name
	result := ""
	for _, v := range model.Indexes {
		def := fmt.Sprintf("idx_%s_%s", name, v)
		if v == jdb.SOURCE {
			def = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIN (%s);", def, table, v)
		} else {
			def = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s);", def, table, v)
		}
		result = strs.Append(result, def, "\n")
	}

	return result, nil
}

/**
* buildUniqueIndex
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) buildUniqueIndex(model *jdb.Model) (string, error) {
	if len(model.Unique) == 0 {
		return "", nil
	}

	table := model.Table
	name := model.Name
	result := ""
	for _, v := range model.Unique {
		def := fmt.Sprintf("idx_%s_%s", name, v)
		def = fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s(%s);", def, table, v)
		result = strs.Append(result, def, "\n")
	}

	return result, nil
}

/**
* buildTriggerBeforeInsert
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) buildTriggerBeforeInsert(model *jdb.Model) (string, error) {
	if model.IndexField == nil {
		return "", nil
	}

	isCore := model.IsCore
	if isCore {
		return "", nil
	}

	table := model.Table
	result := fmt.Sprintf(`
	DROP TRIGGER IF EXISTS RECORDS_SET ON %s CASCADE;
	CREATE TRIGGER RECORDS_SET
	AFTER INSERT OR UPDATE OR DELETE ON %s
	FOR EACH ROW
	EXECUTE FUNCTION core.after_records();
	`, table, table)

	return result, nil
}

/**
* mutateModel
* @param model *jdb.Model
* @return (string, error)
**/
func (s *Driver) mutateModel(model *jdb.Model) (string, error) {
	return "", nil
}
