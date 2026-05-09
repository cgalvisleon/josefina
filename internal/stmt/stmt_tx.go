package stmt

// ── Transaction statements ────────────────────────────────────────────────────

// BeginStmt: BEGIN [TRANSACTION]
// Marks the start of an explicit transaction block.
type BeginStmt struct{}

func (BeginStmt) stmt() {}

// CommitStmt: COMMIT [TRANSACTION]
// Persists all DML operations executed since the matching BEGIN.
type CommitStmt struct{}

func (CommitStmt) stmt() {}

// RollbackStmt: ROLLBACK [TRANSACTION]
// Discards all DML operations executed since the matching BEGIN.
type RollbackStmt struct{}

func (RollbackStmt) stmt() {}

// TxStmt: a complete transaction block assembled by ParseText.
// Stmts holds the DML statements between BEGIN and the closing keyword.
// Rollback is true when the block is explicitly closed with ROLLBACK;
// the executor also rolls back automatically on any execution error.
type TxStmt struct {
	Stmts    []Stmt
	Rollback bool // true = closed with ROLLBACK, false = closed with COMMIT
}

func (TxStmt) stmt() {}
