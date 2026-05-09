package stmt

// ── Command batch ─────────────────────────────────────────────────────────────

// CmdOp: the DML operation kind in a batch entry.
// Mirrors jdb.Cmd (INSERT / UPDATE / DELETE / UPSERT / BULK).
type CmdOp int

const (
	CmdInsert CmdOp = iota
	CmdUpdate
	CmdDelete
	CmdUpsert
	CmdBulk
)

// CmdEntry: one DML operation inside a CmdStmt batch.
//
//	CmdInsert / CmdUpsert  → Table + Row (single record)
//	CmdBulk                → Table + Rows (multiple records)
//	CmdUpdate              → Table + Row (SET data) + Where
//	CmdDelete              → Table + Where
type CmdEntry struct {
	Op    CmdOp
	Table TableRef
	Row   map[string]any   // INSERT / UPDATE / UPSERT
	Rows  []map[string]any // BULK
	Where []CondExpr       // UPDATE / DELETE
}

// CmdStmt: a batch of DML operations executed as a single logical unit.
// The executor runs all entries inside one transaction; any failure triggers
// a full rollback.
type CmdStmt struct {
	Entries []CmdEntry
}

func (CmdStmt) stmt() {}

// ── SQL dialect ───────────────────────────────────────────────────────────────

// SqlDialect identifies the SQL output dialect for a session.
type SqlDialect string

const (
	DialectJosefina   SqlDialect = "JOSEFINA"    // default, PostgreSQL-compatible
	DialectPostgreSQL SqlDialect = "POSTGRESQL"
	DialectMySQL      SqlDialect = "MYSQL"
	DialectOracle     SqlDialect = "ORACLE"
	DialectSQLServer  SqlDialect = "SQLSERVER"
)

// SetSqlStateStmt: SET SQL STATE <dialect> — switches the session SQL dialect.
type SetSqlStateStmt struct {
	Dialect SqlDialect
}

func (SetSqlStateStmt) stmt() {}
