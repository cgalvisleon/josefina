package oracle

import (
	"strings"

	"github.com/cgalvisleon/josefina/internal/stmt"
)

// ── Transaction ───────────────────────────────────────────────────────────────

/**
* genTx: renders a TxStmt as a semicolon-separated Oracle SQL block.
* Oracle starts transactions implicitly, so no BEGIN statement is emitted.
* The block ends with COMMIT or ROLLBACK depending on TxStmt.Rollback.
* @param s stmt.TxStmt
* @return string, error
**/
func genTx(s stmt.TxStmt) (string, error) {
	var parts []string
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
