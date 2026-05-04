package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/jdb"
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
	userRecords := []et.Json{
		{jdb.INDEX: "alice", "username": "alice", "email": "alice@example.com", "age": int64(30), "active": true},
		{jdb.INDEX: "bob", "username": "bob", "email": "bob@example.com", "age": int64(25), "active": true},
		{jdb.INDEX: "carol", "username": "carol", "email": "carol@example.com", "age": int64(35), "active": false},
		{jdb.INDEX: "dave", "username": "dave", "email": "dave@example.com", "age": int64(28), "active": true},
		{jdb.INDEX: "eve", "username": "eve", "email": "eve@example.com", "age": int64(22), "active": false},
	}
	for _, u := range userRecords {
		if _, err := users.Insert(u).Exec(); err != nil {
			logs.Errorf("insert user %s: %v", u.Str("username"), err)
		}
	}
	count, _ := users.Count()
	logs.Infof("users count: %d", count)

	// ── Insert orders ─────────────────────────────────────────────────────────
	section("Insert orders")
	orderRecords := []et.Json{
		{"username": "alice", "product": "laptop", "amount": 1200.00},
		{"username": "alice", "product": "mouse", "amount": 25.00},
		{"username": "bob", "product": "keyboard", "amount": 75.00},
		{"username": "carol", "product": "monitor", "amount": 350.00},
		{"username": "dave", "product": "laptop", "amount": 1200.00},
	}
	for _, o := range orderRecords {
		if _, err := orders.Insert(o).Exec(); err != nil {
			logs.Errorf("insert order: %v", err)
		}
	}
	count, _ = orders.Count()
	logs.Infof("orders count: %d", count)

	// ── Get by primary key ────────────────────────────────────────────────────
	section("Get by primary key")
	for _, id := range []string{"alice", "carol", "zzz"} {
		item, exists, err := users.Current(id)
		if err != nil {
			logs.Errorf("current %s: %v", id, err)
			continue
		}
		if !exists {
			logs.Infof("current %s: not found", id)
			continue
		}
		logs.Infof("current %s: %v", id, item.ToString())
	}

	// ── WHERE: single indexed condition ───────────────────────────────────────
	section("WHERE username = alice")
	items, err := jdb.From(users).
		Where(jdb.Eq("username", "alice")).
		All()
	if err != nil {
		logs.Errorf("where: %v", err)
	} else {
		logItems(items)
	}

	// ── WHERE: multiple AND conditions ────────────────────────────────────────
	section("WHERE active = true AND age > 25")
	items, err = jdb.From(users).
		Where(jdb.Eq("active", true)).
		And(jdb.More("age", int64(25))).
		All()
	if err != nil {
		logs.Errorf("where and: %v", err)
	} else {
		logItems(items)
	}

	// ── WHERE: OR condition ───────────────────────────────────────────────────
	section("WHERE username = alice OR username = bob")
	items, err = jdb.From(users).
		Where(jdb.Eq("username", "alice")).
		Or(jdb.Eq("username", "bob")).
		All()
	if err != nil {
		logs.Errorf("where or: %v", err)
	} else {
		logItems(items)
	}

	// ── ORDER BY + LIMIT (pagination) ─────────────────────────────────────────
	section("ORDER BY age DESC — page 1, 3 rows")
	items, err = jdb.From(users).
		Desc("age").
		Limit(1, 3).
		All()
	if err != nil {
		logs.Errorf("order+limit: %v", err)
	} else {
		logItems(items)
	}

	section("ORDER BY age DESC — page 2, 3 rows")
	items, err = jdb.From(users).
		Desc("age").
		Limit(2, 3).
		All()
	if err != nil {
		logs.Errorf("order+limit page2: %v", err)
	} else {
		logItems(items)
	}

	// ── INNER JOIN ────────────────────────────────────────────────────────────
	// Returns only users that have at least one order.
	section("INNER JOIN users ⨝ orders ON users.username = orders.username")
	items, err = jdb.From(users).
		InnerJoin(orders, map[string]string{"username": "username"}).
		All()
	if err != nil {
		logs.Errorf("inner join: %v", err)
	} else {
		logItems(items)
	}

	// ── LEFT JOIN ─────────────────────────────────────────────────────────────
	// Returns all users; order fields are empty when no order exists.
	section("LEFT JOIN users ⟕ orders ON users.username = orders.username")
	items, err = jdb.From(users).
		LeftJoin(orders, map[string]string{"username": "username"}).
		All()
	if err != nil {
		logs.Errorf("left join: %v", err)
	} else {
		logs.Infof("count: %d (includes users with no orders)", items.Count)
	}

	// ── RIGHT JOIN ────────────────────────────────────────────────────────────
	// Returns all orders; user fields are empty when no user matches.
	section("RIGHT JOIN users ⟖ orders ON users.username = orders.username")
	items, err = jdb.From(users).
		RightJoin(orders, map[string]string{"username": "username"}).
		All()
	if err != nil {
		logs.Errorf("right join: %v", err)
	} else {
		logs.Infof("count: %d (all orders + unmatched users)", items.Count)
	}

	// ── WHERE + JOIN combined ─────────────────────────────────────────────────
	section("WHERE active = true — INNER JOIN orders — ORDER BY age ASC")
	items, err = jdb.From(users).
		Where(jdb.Eq("active", true)).
		InnerJoin(orders, map[string]string{"username": "username"}).
		Asc("age").
		All()
	if err != nil {
		logs.Errorf("where+join+order: %v", err)
	} else {
		logItems(items)
	}

	// ── Cursor: sequential iteration ──────────────────────────────────────────
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
	upserts := []et.Json{
		{jdb.INDEX: "frank", "username": "frank", "email": "frank@example.com", "age": int64(40), "active": true},
		{jdb.INDEX: "alice", "username": "alice", "email": "alice@example.com", "age": int64(31), "active": true},
	}
	for _, u := range upserts {
		if _, err := users.Upsert(u).Exec(); err != nil {
			logs.Errorf("upsert %s: %v", u.Str("username"), err)
		}
	}
	count, _ = users.Count()
	logs.Infof("users count after upsert: %d", count)
	item, _, _ := users.Current("alice")
	logs.Infof("alice after upsert: age=%v", item["age"])

	// ── Update ────────────────────────────────────────────────────────────────
	section("Update — set active=false WHERE age < 25")
	_, err = users.Update(et.Json{"active": false}).
		Where(jdb.Less("age", int64(25))).
		Exec()
	if err != nil {
		logs.Errorf("update: %v", err)
	} else {
		items, _ = jdb.From(users).Where(jdb.Eq("active", false)).All()
		logs.Infof("inactive after update: %d", items.Count)
	}

	// ── Delete ────────────────────────────────────────────────────────────────
	section("Delete — WHERE active = false")
	_, err = users.Delete().
		Where(jdb.Eq("active", false)).
		Exec()
	if err != nil {
		logs.Errorf("delete: %v", err)
	}
	count, _ = users.Count()
	logs.Infof("users count after delete: %d", count)

	// ── Bulk insert ───────────────────────────────────────────────────────────
	section("Bulk insert — 3 new users")
	_, err = users.Bulk([]et.Json{
		{jdb.INDEX: "grace", "username": "grace", "email": "grace@example.com", "age": int64(29), "active": true},
		{jdb.INDEX: "henry", "username": "henry", "email": "henry@example.com", "age": int64(33), "active": true},
		{jdb.INDEX: "iris", "username": "iris", "email": "iris@example.com", "age": int64(27), "active": true},
	}).Exec()
	if err != nil {
		logs.Errorf("bulk: %v", err)
	}
	count, _ = users.Count()
	logs.Infof("users count after bulk: %d", count)

	section("done")
	return nil
}

func section(title string) {
	logs.Infof("=== %s ===", title)
}

func logItems(items et.Items) {
	logs.Infof("count: %d", items.Count)
	for _, item := range items.Result {
		logs.Infof("%v", item.ToString())
	}
}
