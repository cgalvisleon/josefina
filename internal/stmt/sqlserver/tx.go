package sqlserver

import (
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ── Transaction ───────────────────────────────────────────────────────────────

/**
* genTx: renders a TxStmt as a semicolon-separated T-SQL block.
* The block opens with BEGIN TRANSACTION and ends with COMMIT or ROLLBACK.
* @param s stmt.TxStmt
* @return string, error
**/
func genTx(s stmt.TxStmt) (string, error) {
	parts := []string{"BEGIN TRANSACTION"}
	for _, st := range s.Stmts {
		sql, err := ToSQL(st)
		if err != nil {
			return "", err
		}
		parts = append(parts, sql)
	}
	if s.Rollback {
		parts = append(parts, "ROLLBACK")
	} else {
		parts = append(parts, "COMMIT")
	}
	return strings.Join(parts, ";\n"), nil
}
