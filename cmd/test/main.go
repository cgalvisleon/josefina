package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/jdb"
	"github.com/cgalvisleon/josefina/internal/store"
)

func main() {
	// testStore()
	testCatalog()
}

func testStore() {
	path := filepath.Join("./", "data")
	defer os.RemoveAll(path)

	fs, err := store.Open(path, "demo", store.ReadWrite)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer fs.Close()

	fs.IsDebug()
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

func testCatalog() {
	// ── Setup ─────────────────────────────────────────────────────────────────
	db, err := jdb.NewDb("./data", "test")
	if err != nil {
		logs.Fatal(err)
	}

	model, err := db.NewModel("", "user", false, 1)
	if err != nil {
		logs.Fatal(err)
	}

	model.DefineAtrib("name", jdb.TpText, "")
	model.DefineAtrib("age", jdb.TpInt, 0)
	model.DefineAtrib("id", jdb.TpKey, "")
	model.DefinePrimaryKeys("id")
	model.DefineIndexes("name", "age")

	if err := model.Init(); err != nil {
		logs.Fatal(err)
	}
	logs.Info("=== model init ok ===")

	// ── Insert 5 registros ────────────────────────────────────────────────────
	users := []et.Json{
		{"id": "pk1", "name": "Alice", "age": int64(30)},
		{"id": "pk2", "name": "Bob", "age": int64(25)},
		{"id": "pk3", "name": "Carol", "age": int64(35)},
		{"id": "pk4", "name": "Dave", "age": int64(28)},
		{"id": "pk5", "name": "Eve", "age": int64(22)},
	}
	var tx *jdb.Tx
	for _, u := range users {
		tx, err = model.Insert(u.Str("id"), u, tx)
		if err != nil {
			logs.Errorf("insert %s: %v", u.Str("id"), err)
			continue
		}
	}
	count, _ := model.Count()
	logs.Infof("inserted: count=%d", count)

	// ── Get por primary key ───────────────────────────────────────────────────
	logs.Info("--- Get by primary key ---")
	for _, id := range []string{"pk1", "pk3", "pk99"} {
		dest := et.Json{}
		exists, err := model.Get(id, &dest)
		if err != nil {
			logs.Errorf("get %s: %v", id, err)
			continue
		}
		if !exists {
			logs.Infof("get %s: not found", id)
			continue
		}
		logs.Infof("get %s: %v", id, dest)
	}

	// ── GetByIndex por nombre exacto ──────────────────────────────────────────
	logs.Info("--- GetByIndex name=Alice ---")
	pks, ok := model.EqualByIndex("name", jdb.KeyString("Alice"))
	if !ok {
		logs.Info("name=Alice: not found")
	} else {
		logs.Infof("name=Alice: pks=%v", pks)
	}

	logs.Info("--- GetByIndex name=Zzz (no existe) ---")
	pks, ok = model.EqualByIndex("name", jdb.KeyString("Zzz"))
	if !ok {
		logs.Info("name=Zzz: not found (expected)")
	} else {
		logs.Infof("name=Zzz: pks=%v", pks)
	}

	// ── RangeIndex por edad ───────────────────────────────────────────────────
	logs.Info("--- RangeIndex age [25, 30] asc ---")
	pks = model.BetweenByIndex("age", jdb.KeyInt(25), jdb.KeyInt(30), true)
	logs.Infof("age [25,30]: pks=%v", pks)

	logs.Info("--- RangeIndex age [0, 99] desc ---")
	pks = model.BetweenByIndex("age", jdb.KeyInt(0), jdb.KeyInt(99), false)
	logs.Infof("age [0,99] desc: pks=%v", pks)

	// ── Simular restart: re-init reconstruye BTrees desde FileStore ───────────
	logs.Info("--- Simulating restart ---")
	model.IsInit = false
	if err := model.Init(); err != nil {
		logs.Errorf("re-init: %v", err)
	}
	pks, ok = model.EqualByIndex("name", jdb.KeyString("Bob"))
	if !ok {
		logs.Debug("restart: name=Bob not found (BTree rebuild failed)")
	} else {
		logs.Infof("restart: name=Bob pks=%v (BTree rebuilt ok)", pks)
	}

	// ── Delete ────────────────────────────────────────────────────────────────
	logs.Info("--- RemoveObject pk2 ---")
	tx, err = model.Delete("pk2", nil)
	if err != nil {
		logs.Errorf("remove pk2: %v", err)
	}
	count, _ = model.Count()
	logs.Infof("after delete: count=%d", count)

	pks, ok = model.EqualByIndex("name", jdb.KeyString("Bob"))
	if !ok {
		logs.Info("name=Bob after delete: not found (expected)")
	} else {
		logs.Infof("name=Bob after delete: pks=%v (unexpected)", pks)
	}

	logs.Info("=== jdb test done ===")
}
