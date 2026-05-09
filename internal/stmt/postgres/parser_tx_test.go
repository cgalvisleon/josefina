package stmt

import (
	"strings"
	"testing"
)

// ── Parsing ───────────────────────────────────────────────────────────────────

func TestParseTx_BeginCommit(t *testing.T) {
	stmts, err := ParseText(`
		BEGIN;
		INSERT INTO users (username, age) VALUES ('alice', 30);
		COMMIT;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 TxStmt, got %d statements", len(stmts))
	}
	tx, ok := stmts[0].(TxStmt)
	if !ok {
		t.Fatalf("expected TxStmt, got %T", stmts[0])
	}
	if tx.Rollback {
		t.Error("expected Rollback=false for COMMIT block")
	}
	if len(tx.Stmts) != 1 {
		t.Fatalf("expected 1 inner stmt, got %d", len(tx.Stmts))
	}
	if _, ok := tx.Stmts[0].(InsertStmt); !ok {
		t.Errorf("expected InsertStmt inside tx, got %T", tx.Stmts[0])
	}
}

func TestParseTx_BeginTransaction(t *testing.T) {
	stmts, err := ParseText(`BEGIN TRANSACTION; SELECT * FROM users; COMMIT TRANSACTION;`)
	if err != nil {
		t.Fatal(err)
	}
	tx := stmts[0].(TxStmt)
	if tx.Rollback {
		t.Error("expected Rollback=false")
	}
	if len(tx.Stmts) != 1 {
		t.Fatalf("expected 1 inner stmt, got %d", len(tx.Stmts))
	}
}

func TestParseTx_BeginRollback(t *testing.T) {
	stmts, err := ParseText(`
		BEGIN;
		UPDATE users SET active = FALSE WHERE age < 18;
		DELETE FROM logs WHERE active = FALSE;
		ROLLBACK;
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := stmts[0].(TxStmt)
	if !tx.Rollback {
		t.Error("expected Rollback=true for ROLLBACK block")
	}
	if len(tx.Stmts) != 2 {
		t.Fatalf("expected 2 inner stmts, got %d", len(tx.Stmts))
	}
	if _, ok := tx.Stmts[0].(UpdateStmt); !ok {
		t.Errorf("expected UpdateStmt, got %T", tx.Stmts[0])
	}
	if _, ok := tx.Stmts[1].(DeleteStmt); !ok {
		t.Errorf("expected DeleteStmt, got %T", tx.Stmts[1])
	}
}

func TestParseTx_EmptyBlock(t *testing.T) {
	stmts, err := ParseText(`BEGIN; COMMIT;`)
	if err != nil {
		t.Fatal(err)
	}
	tx := stmts[0].(TxStmt)
	if len(tx.Stmts) != 0 {
		t.Errorf("expected 0 inner stmts, got %d", len(tx.Stmts))
	}
}

func TestParseTx_MultipleDML(t *testing.T) {
	stmts, err := ParseText(`
		BEGIN;
		INSERT INTO orders (product, amount) VALUES ('book', 15);
		INSERT INTO orders (product, amount) VALUES ('pen', 2);
		UPDATE orders SET amount = 20 WHERE product = 'book';
		COMMIT;
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := stmts[0].(TxStmt)
	if len(tx.Stmts) != 3 {
		t.Fatalf("expected 3 inner stmts, got %d", len(tx.Stmts))
	}
}

func TestParseTx_MixedWithNonTx(t *testing.T) {
	stmts, err := ParseText(`
		SELECT * FROM users;
		BEGIN;
		INSERT INTO logs (msg) VALUES ('started');
		COMMIT;
		SELECT * FROM logs;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 3 {
		t.Fatalf("expected 3 top-level stmts, got %d", len(stmts))
	}
	if _, ok := stmts[0].(SelectStmt); !ok {
		t.Errorf("expected SelectStmt at [0], got %T", stmts[0])
	}
	if _, ok := stmts[1].(TxStmt); !ok {
		t.Errorf("expected TxStmt at [1], got %T", stmts[1])
	}
	if _, ok := stmts[2].(SelectStmt); !ok {
		t.Errorf("expected SelectStmt at [2], got %T", stmts[2])
	}
}

// ── Error cases ───────────────────────────────────────────────────────────────

func TestParseTx_StrayCommit(t *testing.T) {
	_, err := ParseText(`COMMIT;`)
	if err == nil {
		t.Fatal("expected error for COMMIT outside BEGIN, got nil")
	}
}

func TestParseTx_StrayRollback(t *testing.T) {
	_, err := ParseText(`ROLLBACK;`)
	if err == nil {
		t.Fatal("expected error for ROLLBACK outside BEGIN, got nil")
	}
}

func TestParseTx_UnterminatedBlock(t *testing.T) {
	_, err := ParseText(`BEGIN; INSERT INTO users (username) VALUES ('bob');`)
	if err == nil {
		t.Fatal("expected error for unterminated transaction, got nil")
	}
	if !strings.Contains(err.Error(), "unterminated transaction") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseTx_NestedBegin(t *testing.T) {
	_, err := ParseText(`BEGIN; BEGIN; COMMIT; COMMIT;`)
	if err == nil {
		t.Fatal("expected error for nested BEGIN, got nil")
	}
	if !strings.Contains(err.Error(), "nested BEGIN") {
		t.Errorf("unexpected error message: %v", err)
	}
}
