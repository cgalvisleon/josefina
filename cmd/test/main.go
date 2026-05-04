package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/jdb"
	"github.com/cgalvisleon/josefina/internal/stmt"
)

const dataPath = "./data/test"

func main() {
	// os.RemoveAll(dataPath)
	// defer os.RemoveAll(dataPath)

	if err := run(); err != nil {
		logs.Fatal(err)
	}
}

func run() error {
	// ── Database ──────────────────────────────────────────────────────────────
	db, err := jdb.NewDb("./data", "test")
	if err != nil {
		return err
	}
	if err := db.Init(); err != nil {
		return err
	}

	// ── Model: users ──────────────────────────────────────────────────────────
	// db.Define is still used for model creation: SQL CREATE TABLE does not
	// support josefina-specific BTree index declarations.
	users, err := db.Define(jdb.DModel{
		Schema:  "apps",
		Name:    "users",
		Version: 1,
		Fields: map[string]jdb.DField{
			"username": {Type: jdb.TpKey, Default: ""},
			"email":    {Type: jdb.TpText, Default: ""},
			"age":      {Type: jdb.TpInt, Default: 0},
			"active":   {Type: jdb.TpBoolean, Default: true},
		},
		PrimaryKeys: []string{"username"},
		Indexes:     []jdb.DIndex{{Name: "email"}, {Name: "age"}, {Name: "active"}},
		Unique:      []jdb.DIndex{{Name: "email"}},
		Required:    []jdb.DIndex{{Name: "email"}},
	})
	if err != nil {
		return err
	}
	if err := users.Init(); err != nil {
		return err
	}
	count, _ := users.Count()
	logs.Infof("users count: %d", count)

	// ── Model: orders ─────────────────────────────────────────────────────────
	orders, err := db.Define(jdb.DModel{
		Schema:  "apps",
		Name:    "orders",
		Version: 1,
		Fields: map[string]jdb.DField{
			"username": {Type: jdb.TpKey, Default: ""},
			"product":  {Type: jdb.TpText, Default: ""},
			"amount":   {Type: jdb.TpFloat, Default: 0.0},
		},
		Indexes:  []jdb.DIndex{{Name: "username"}, {Name: "product"}},
		Required: []jdb.DIndex{{Name: "username"}, {Name: "product"}},
	})
	if err != nil {
		return err
	}
	if err := orders.Init(); err != nil {
		return err
	}
	count, _ = orders.Count()
	logs.Infof("orders count: %d", count)

	// ── Insert users ──────────────────────────────────────────────────────────
	section("Insert users")
	if _, err := stmt.ExecSQL(db, `
		INSERT INTO apps.users (_idx, username, email, age, active) VALUES
			('alice', 'alice', 'alice@example.com', 30, TRUE),
			('bob',   'bob',   'bob@example.com',   25, TRUE),
			('carol', 'carol', 'carol@example.com', 35, FALSE),
			('dave',  'dave',  'dave@example.com',  28, TRUE),
			('eve',   'eve',   'eve@example.com',   22, FALSE)
	`); err != nil {
		logs.Errorf("insert users: %v", err)
	}
	count, _ = users.Count()
	logs.Infof("users count: %d", count)

	// ── Insert orders ─────────────────────────────────────────────────────────
	section("Insert orders")
	if _, err := stmt.ExecSQL(db, `
		INSERT INTO apps.orders (username, product, amount) VALUES
			('alice', 'laptop',   1200.00),
			('alice', 'mouse',      25.00),
			('bob',   'keyboard',   75.00),
			('carol', 'monitor',   350.00),
			('dave',  'laptop',   1200.00)
	`); err != nil {
		logs.Errorf("insert orders: %v", err)
	}
	count, _ = orders.Count()
	logs.Infof("orders count: %d", count)

	// ── Get by primary key ────────────────────────────────────────────────────
	section("Get by primary key")
	for _, id := range []string{"alice", "carol", "zzz"} {
		items, err := stmt.ExecSQL(db,
			"SELECT * FROM apps.users WHERE username = '"+id+"'")
		if err != nil {
			logs.Errorf("select %s: %v", id, err)
			continue
		}
		if items.Count == 0 {
			logs.Infof("select %s: not found", id)
			continue
		}
		logs.Infof("select %s: %v", id, items.Result[0].ToString())
	}

	// ── WHERE: single indexed condition ───────────────────────────────────────
	section("WHERE username = 'alice'")
	logSQL(db, `SELECT * FROM apps.users WHERE username = 'alice'`)

	// ── WHERE: multiple AND conditions ────────────────────────────────────────
	section("WHERE active = TRUE AND age > 25")
	logSQL(db, `SELECT * FROM apps.users WHERE active = TRUE AND age > 25`)

	// ── WHERE: OR condition ───────────────────────────────────────────────────
	section("WHERE username = 'alice' OR username = 'bob'")
	logSQL(db, `SELECT * FROM apps.users WHERE username = 'alice' OR username = 'bob'`)

	// ── ORDER BY + LIMIT (pagination) ─────────────────────────────────────────
	section("ORDER BY age DESC LIMIT 3")
	logSQL(db, `SELECT * FROM apps.users ORDER BY age DESC LIMIT 3`)

	section("ORDER BY age DESC LIMIT 3 OFFSET 3")
	logSQL(db, `SELECT * FROM apps.users ORDER BY age DESC LIMIT 3 OFFSET 3`)

	// ── INNER JOIN ────────────────────────────────────────────────────────────
	section("INNER JOIN users ⨝ orders ON users.username = orders.username")
	logSQL(db, `
		SELECT * FROM apps.users
		INNER JOIN apps.orders ON users.username = orders.username
	`)

	// ── LEFT JOIN ─────────────────────────────────────────────────────────────
	section("LEFT JOIN users ⟕ orders")
	items, err := stmt.ExecSQL(db, `
		SELECT * FROM apps.users
		LEFT JOIN apps.orders ON users.username = orders.username
	`)
	if err != nil {
		logs.Errorf("left join: %v", err)
	} else {
		logs.Infof("count: %d (includes users with no orders)", items.Count)
	}

	// ── RIGHT JOIN ────────────────────────────────────────────────────────────
	section("RIGHT JOIN users ⟖ orders")
	items, err = stmt.ExecSQL(db, `
		SELECT * FROM apps.users
		RIGHT JOIN apps.orders ON users.username = orders.username
	`)
	if err != nil {
		logs.Errorf("right join: %v", err)
	} else {
		logs.Infof("count: %d (all orders + unmatched users)", items.Count)
	}

	// ── WHERE + JOIN combined ─────────────────────────────────────────────────
	section("WHERE active = TRUE — INNER JOIN orders — ORDER BY age ASC")
	logSQL(db, `
		SELECT * FROM apps.users
		INNER JOIN apps.orders ON users.username = orders.username
		WHERE active = TRUE
		ORDER BY age ASC
	`)

	// ── Cursor: sequential iteration (no SQL equivalent) ─────────────────────
	section("Cursor — asc, offset 1, limit 3")
	cursor, err := users.NewCursor(true, 1, 3)
	if err != nil {
		logs.Errorf("new cursor: %v", err)
	} else {
		defer cursor.Close()
		logs.Infof("snapshot: %d records", cursor.Len())
		for cursor.Next() {
			var row et.Json
			if err := cursor.Scan(&row); err != nil {
				logs.Errorf("cursor.scan: %v", err)
				break
			}
			logs.Infof("  [pos %d] %v", cursor.Pos(), row)
		}
	}

	// ── Upsert ────────────────────────────────────────────────────────────────
	section("Upsert — frank (new) / alice age → 31 (existing)")
	if _, err := stmt.ExecSQL(db, `
		UPSERT INTO apps.users (_idx, username, email, age, active) VALUES
			('frank', 'frank', 'frank@example.com', 40, TRUE),
			('alice', 'alice', 'alice@example.com', 31, TRUE)
	`); err != nil {
		logs.Errorf("upsert: %v", err)
	}
	count, _ = users.Count()
	logs.Infof("users count after upsert: %d", count)
	aliceItems, _ := stmt.ExecSQL(db, `SELECT * FROM apps.users WHERE username = 'alice'`)
	if aliceItems.Count > 0 {
		logs.Infof("alice after upsert: age=%v", aliceItems.Result[0]["age"])
	}

	// ── Update ────────────────────────────────────────────────────────────────
	section("UPDATE — set active = FALSE WHERE age < 25")
	if _, err := stmt.ExecSQL(db, `UPDATE apps.users SET active = FALSE WHERE age < 25`); err != nil {
		logs.Errorf("update: %v", err)
	} else {
		items, _ := stmt.ExecSQL(db, `SELECT * FROM apps.users WHERE active = FALSE`)
		logs.Infof("inactive after update: %d", items.Count)
	}

	// ── Delete ────────────────────────────────────────────────────────────────
	section("DELETE — WHERE active = FALSE")
	if _, err := stmt.ExecSQL(db, `DELETE FROM apps.users WHERE active = FALSE`); err != nil {
		logs.Errorf("delete: %v", err)
	}
	count, _ = users.Count()
	logs.Infof("users count after delete: %d", count)

	// ── Bulk insert ───────────────────────────────────────────────────────────
	section("Bulk insert — 3 new users")
	if _, err := stmt.ExecSQL(db, `
		INSERT INTO apps.users (_idx, username, email, age, active) VALUES
			('grace', 'grace', 'grace@example.com', 29, TRUE),
			('henry', 'henry', 'henry@example.com', 33, TRUE),
			('iris',  'iris',  'iris@example.com',  27, TRUE)
	`); err != nil {
		logs.Errorf("bulk insert: %v", err)
	}
	count, _ = users.Count()
	logs.Infof("users count after bulk: %d", count)

	section("done")
	return nil
}

func section(title string) {
	logs.Infof("=== %s ===", title)
}

func logSQL(db *jdb.DB, sql string) {
	items, err := stmt.ExecSQL(db, sql)
	if err != nil {
		logs.Errorf("sql: %v", err)
		return
	}
	logs.Infof("count: %d", items.Count)
	for _, item := range items.Result {
		logs.Infof("%v", item.ToString())
	}
}
