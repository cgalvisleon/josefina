package stmt

// ── SQL → jdb type mapping ────────────────────────────────────────────────────

// SqlType: SQL column type as written in CREATE TABLE.
// Maps to jdb.TypeData in the executor.
type SqlType string

const (
	SqlText      SqlType = "TEXT"
	SqlVarchar   SqlType = "VARCHAR"
	SqlInt       SqlType = "INT"
	SqlBigInt    SqlType = "BIGINT"
	SqlSmallInt  SqlType = "SMALLINT"
	SqlFloat     SqlType = "FLOAT"
	SqlReal      SqlType = "REAL"
	SqlNumeric   SqlType = "NUMERIC"
	SqlBool      SqlType = "BOOLEAN"
	SqlJson      SqlType = "JSON"
	SqlJsonb     SqlType = "JSONB"
	SqlTimestamp SqlType = "TIMESTAMP"
	SqlDate      SqlType = "DATE"
	SqlBytes     SqlType = "BYTEA"
	SqlKey       SqlType = "KEY" // josefina-specific: natural key / primary ident
)

// ── Column definition ─────────────────────────────────────────────────────────

// ColumnDef: a single column definition inside CREATE TABLE / ALTER TABLE ADD.
type ColumnDef struct {
	Name       string
	Type       SqlType
	Default    any  // nil = no DEFAULT
	NotNull    bool // NOT NULL constraint
	PrimaryKey bool // inline PRIMARY KEY shorthand
	Unique     bool // inline UNIQUE shorthand
}

// ── DDL statements ────────────────────────────────────────────────────────────

// CreateTableStmt: CREATE TABLE [IF NOT EXISTS] [schema.]name (col_defs...) [constraints]
type CreateTableStmt struct {
	Table       TableRef
	Columns     []ColumnDef
	PrimaryKeys []string // table-level PRIMARY KEY (col, ...)
	Unique      []string // table-level UNIQUE (col) shorthand
	Required    []string // NOT NULL columns collected from ColumnDef
	IfNotExists bool
}

func (CreateTableStmt) stmt() {}

// DropTableStmt: DROP TABLE [IF EXISTS] [schema.]name
type DropTableStmt struct {
	Table    TableRef
	IfExists bool
}

func (DropTableStmt) stmt() {}

// AlterAction: the kind of modification in ALTER TABLE.
type AlterAction int

const (
	AlterAddColumn AlterAction = iota
	AlterDropColumn
	AlterAlterColumn
	AlterAddPrimaryKey
	AlterAddUnique
)

// AlterTableStmt: ALTER TABLE [schema.]name action
type AlterTableStmt struct {
	Table   TableRef
	Action  AlterAction
	Column  ColumnDef // ADD COLUMN / ALTER COLUMN
	ColName string    // DROP COLUMN — just the name
}

func (AlterTableStmt) stmt() {}
