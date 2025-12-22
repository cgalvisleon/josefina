package server

const (
	SOURCE     string = "source"
	KEY        string = "id"
	INDEX      string = "index"
	STATUS     string = "status"
	VERSION    string = "version"
	PROJECT_ID string = "project_id"
	TENANT_ID  string = "tenant_id"
	CREATED_AT string = "created_at"
	UPDATED_AT string = "updated_at"
)

type TypeColumn string

func (s TypeColumn) Str() string {
	return string(s)
}

const (
	TpColumn      TypeColumn = "column"
	TpAtrib       TypeColumn = "atrib"
	TpDetail      TypeColumn = "detail"
	TpRollup      TypeColumn = "rollup"
	TpRelation    TypeColumn = "relation"
	TpAggregation TypeColumn = "aggregation"
	TpValue       TypeColumn = "value"
	TpExpression  TypeColumn = "expression"
)

type TypeData string

func (s TypeData) Str() string {
	return string(s)
}

const (
	TpAny      TypeData = "any"
	TpBytes    TypeData = "bytes"
	TpInt      TypeData = "int"
	TpFloat    TypeData = "float"
	TpKey      TypeData = "key"
	TpText     TypeData = "text"
	TpMemo     TypeData = "memo"
	TpJson     TypeData = "json"
	TpDateTime TypeData = "datetime"
	TpBoolean  TypeData = "boolean"
	TpGeometry TypeData = "geometry"
	TpCalc     TypeData = "calc"
)

type Column struct {
	From       string     `json:"from"`
	Name       string     `json:"name"`
	TypeColumn TypeColumn `json:"type_column"`
	TypeData   TypeData   `json:"type_data"`
	Default    any        `json:"default"`
	Definition []byte     `json:"definition"`
}
