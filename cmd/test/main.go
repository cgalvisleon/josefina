package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/store"
)

func main() {
	path := filepath.Join("./", "data")
	// defer os.RemoveAll(path)

	fs, err := store.Open(path, "demo", true)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer fs.Close()

	// ── Put ────────────────────────────────────────────────────────────────
	records := []et.Json{
		{"id": "1", "name": "Alice", "age": 30},
		{"id": "2", "name": "Bob", "age": 25},
		{"id": "3", "name": "Carol", "age": 35},
		{"id": "4", "name": "Dave", "age": 28},
		{"id": "5", "name": "Eve", "age": 22},
	}
	for _, r := range records {
		id := r.Str("id")
		if err := fs.Put(id, r); err != nil {
			logs.Errorf("put:%s: %v", id, err)
			continue
		}
	}
	logs.Infof("count:%d", fs.Count())

	// ── Get ────────────────────────────────────────────────────────────────
	logs.Info("Get")
	for _, id := range []string{"1", "3", "99"} {
		var dest et.Json
		existed, err := fs.Get(id, &dest)
		if err != nil {
			logs.Errorf("get:%s: %v", id, err)
			continue
		}
		if !existed {
			logs.Infof("get:%s: not found", id)
			continue
		}
		logs.Infof("get:%s: %v", id, dest)
	}

	// ── IsExist ────────────────────────────────────────────────────────────
	logs.Info("IsExist")
	for _, id := range []string{"2", "99"} {
		existed := fs.IsExist(id)
		logs.Infof("isExist:%s: %v", id, existed)
	}

	// ── Delete ─────────────────────────────────────────────────────────────
	logs.Info("Delete")
	deleted, err := fs.Delete("2")
	if err != nil {
		logs.Errorf("delete:%s: %v", "2", err)
	} else {
		logs.Infof("delete:%s: deleted=%v lsn=%d", "2", deleted, fs.WAL)
	}
	logs.Infof("count:%d", fs.Count())

	// ── Put actualiza registro existente ───────────────────────────────────
	logs.Info("Put (update)")
	if err := fs.Put("1", et.Json{"id": "1", "name": "Alice", "age": 31}); err != nil {
		logs.Errorf("update:%s: %v", "1", err)
	} else {
		logs.Infof("update:%s: lsn=%d tombstones=%d", "1", fs.WAL, fs.TombStones)
	}

	// ── Iterate ────────────────────────────────────────────────────────────
	logs.Info("Iterate (asc, offset=0, limit=0, workers=2)")
	err = fs.Iterate(func(id string, data []byte) (bool, error) {
		logs.Infof("  %-8s  %s", id, data)
		return true, nil
	}, true, 0, 0, 2)
	if err != nil {
		logs.Errorf("iterate: %v", err)
	}

	// ── Keys ───────────────────────────────────────────────────────────────
	logs.Info("Keys (desc, offset=0, limit=3)")
	for _, k := range fs.Keys(false, 0, 3) {
		logs.Infof("  %s", k)
	}

	// ── WalSince ───────────────────────────────────────────────────────────
	logs.Info("WalSince(3)")
	entries, err := fs.WalSince(3)
	if err != nil {
		logs.Errorf("walsince: %v", err)
	} else {
		for _, e := range entries {
			status := "active"
			if e.Status == store.Deleted {
				status = "deleted"
			}
			logs.Infof("  lsn=%-3d  %-8s  %-8s  %s", e.LSN, status, e.ID, e.Data)
		}
	}
	logs.Info("done")

	count := fs.Count()
	logs.Infof("=== done  lsn=%d  count=%d  tombstones=%d ===", fs.WAL, count, fs.TombStones)
}
