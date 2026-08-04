# Josefina

Custom distributed document database engine written in Go with SQL-like query syntax.

## Setup

```bash
goenv local 1.23.0
go mod init github.com/josefina
go get github.com/cgalvisleon/et@v1.0.22
go get github.com/gorilla/websocket
git remote add origin https://github.com/josefina.git
```

## Running

### Server

```bash
# Single node
go run ./cmd/server -port 1377 -http 3500

# Three-node cluster (separate terminals)
go run ./cmd/server -port 1377 -http 3500
go run ./cmd/server -port 1378 -http 3501
go run ./cmd/server -port 1379 -http 3502
```

### Client — TCP mode (remote)

Connects to a running server over TCP. Requires the server to be started first.

```bash
go run ./cmd/client -host localhost:1377 -user admin -password secret -database mydb
```

Flags:

| Flag        | Default | Description                                               |
| ----------- | ------- | --------------------------------------------------------- |
| `-host`     | `""`    | Server address (`host:port`). When set, TCP mode is used. |
| `-user`     | `admin` | Username                                                  |
| `-password` | `""`    | Password                                                  |
| `-database` | `""`    | Database to connect to                                    |

#### Interactive session example

```
mydb=# CREATE DATABASE shop;
Database "shop" created.

mydb=# \c shop
You are now connected to database "shop".

shop=# CREATE TABLE IF NOT EXISTS products (
shop-#   product_id KEY     NOT NULL,
shop-#   name       TEXT    NOT NULL,
shop-#   price      NUMERIC DEFAULT 0.0,
shop-#   active     BOOLEAN DEFAULT TRUE,
shop-#   PRIMARY KEY (product_id)
shop-# );
OK

shop=# INSERT INTO products (_idx, product_id, name, price, active) VALUES
shop-#   ('p1', 'p1', 'Laptop', 999.99, TRUE),
shop-#   ('p2', 'p2', 'Mouse',   29.99, TRUE);
OK

shop=# SELECT * FROM products WHERE active = TRUE ORDER BY price DESC;
+------------+--------+---------+--------+
| active     | name   | price   | product_id |
+------------+--------+---------+--------+
| true       | Laptop | 999.99  | p1     |
| true       | Mouse  | 29.99   | p2     |
+------------+--------+---------+--------+
(2 rows)

shop=# SET SQL STATE MYSQL;
SQL dialect set to MYSQL.

shop=# SET SQL STATE JOSEFINA;
SQL dialect set to JOSEFINA.

shop=# \timing
Timing is on.

shop=# SELECT * FROM products;
...
Time: 0.412 ms

shop=# \q
Bye.
```

#### Available meta-commands

| Command     | Description             |
| ----------- | ----------------------- |
| `\c <db>`   | Switch database         |
| `\timing`   | Toggle query timing     |
| `\i <file>` | Execute SQL from a file |
| `\help`     | Show all commands       |
| `\q`        | Quit                    |

#### SET SQL STATE

Changes the active SQL dialect for the session. Josefina executes queries natively; this command controls which dialect the server uses to translate and report generated SQL.

```sql
SET SQL STATE JOSEFINA;    -- default (PostgreSQL-compatible)
SET SQL STATE POSTGRESQL;
SET SQL STATE MYSQL;
SET SQL STATE ORACLE;
SET SQL STATE SQLSERVER;
```

### Client — Local mode (embedded)

Loads the database engine in-process. No server needed; useful for development and scripting.

```bash
go run ./cmd/client -user admin -password secret -database mydb
```

When `-host` is omitted the client boots the jdb engine directly from the local data directory (`-data ./data`).

---

## Package `internal/catalog`

The `catalog` package is the schema and data layer of Josefina. It provides a typed document model with primary key storage, secondary indexes backed by a B+ tree, and rich query operators.

### Architecture

```
DB
└── Schema
    └── Model
        ├── FileStore (primary)     — documents keyed by primary key
        ├── FileStore (per index)   — inverted index: fieldValue → {pk...}  (durable)
        └── BTree    (per index)    — in-memory B+ tree for fast queries
```

On startup (`Init`) each secondary BTree is rebuilt from its FileStore — O(distinct values), not O(documents).

On every write (`PutObject`, `RemoveObject`) both the FileStore and the BTree are kept in sync.

---

### Types

#### `DB`

Top-level container for schemas.

```go
db, err := catalog.NewDb("mydb")
```

#### `Schema`

Groups models inside a database. Created automatically by `db.NewModel`.

#### `Model`

A typed document collection. Holds field definitions, indexes, triggers, and the underlying stores.

```go
model, err := db.NewModel("public", "user", false, 1)
```

#### `Field` — `TypeData` constants

| Constant                    | Description                    |
| --------------------------- | ------------------------------ |
| `TpKey`                     | String identifier (UUID, etc.) |
| `TpText` / `TpMemo`         | Short / long text              |
| `TpInt` / `TpAutoIncrement` | Integer                        |
| `TpFloat`                   | Floating point                 |
| `TpBoolean`                 | Boolean                        |
| `TpDateTime`                | Timestamp (RFC3339)            |
| `TpJson`                    | Nested JSON object             |
| `TpAny`                     | Any value                      |

---

### Defining a Model

```go
db, _ := catalog.NewDb("mydb")
model, _ := db.NewModel("public", "user", false, 1)

// Fields
model.DefineAtrib("id",    catalog.TpKey,  "")
model.DefineAtrib("name",  catalog.TpText, "")
model.DefineAtrib("age",   catalog.TpInt,  0)
model.DefineAtrib("email", catalog.TpText, "")

// Constraints
model.DefinePrimaryKeys("id")
model.DefineUnique("email")
model.DefineRequired("name")

// Secondary indexes (B+ tree)
model.DefineIndexes("name", "age", "email")

// Open stores and rebuild in-memory indexes
model.Init()
```

---

### Writing Documents

```go
// Insert or update — idx is the primary key
err := model.PutObject("pk1", et.Json{
    "id":    "pk1",
    "name":  "Alice",
    "age":   int64(30),
    "email": "alice@example.com",
})

// Delete
err = model.RemoveObject("pk1")
```

---

### Reading Documents

#### By primary key

```go
dest := et.Json{}
exists, err := model.Get("pk1", &dest)
```

#### Exact match on secondary index

```go
pks, ok := model.GetByIndex("name", catalog.KeyString("Alice"))
// pks = ["pk1", "pk8", ...]
```

---

### Index Queries

All query methods return `[]string` — the list of matching primary keys.

#### Range — `[from, to]` inclusive

```go
// age between 25 and 35
pks := model.RangeIndex("age", catalog.KeyInt(25), catalog.KeyInt(35), true)

// Pass IndexKey{} for open bounds
pks = model.RangeIndex("age", catalog.KeyInt(25), catalog.IndexKey{}, true) // age >= 25
```

#### Comparison operators

| Method                      | Operator | Example                                         |
| --------------------------- | -------- | ----------------------------------------------- |
| `GTIndex(field, key, asc)`  | `>`      | `model.GTIndex("age", KeyInt(28), true)`        |
| `GTEIndex(field, key, asc)` | `>=`     | `model.GTEIndex("age", KeyInt(25), true)`       |
| `LTIndex(field, key, asc)`  | `<`      | `model.LTIndex("age", KeyInt(30), true)`        |
| `LTEIndex(field, key, asc)` | `<=`     | `model.LTEIndex("age", KeyInt(30), true)`       |
| `NotEqualIndex(field, key)` | `!=`     | `model.NotEqualIndex("name", KeyString("Bob"))` |

---

### `IndexKey` — Typed Keys

Keys carry type information so ordering is always correct (numeric for numbers, lexicographic for strings).

| Constructor                | Go type    | Ordering      |
| -------------------------- | ---------- | ------------- |
| `KeyString(v string)`      | string     | Lexicographic |
| `KeyInt(v int64)`          | int64      | Numeric       |
| `KeyFloat(v float64)`      | float64    | Numeric       |
| `KeyBool(v bool)`          | bool       | false < true  |
| `KeyDateTime(v time.Time)` | time.Time  | Chronological |
| `KeyFromAny(v any)`        | any (JSON) | Auto-detect   |

`KeyFromAny` auto-detects the type from JSON-unmarshalled values (`float64` → `KeyInt` for whole numbers, RFC3339 strings → `KeyDateTime`).

---

### Triggers

JavaScript hooks that run before/after insert, update, and delete operations.

```go
model.AddBeforeInsert("validate", `
    if (!record.name) throw new Error("name required");
`)

model.AddAfterInsert("notify", `
    console.log("inserted", record.id);
`)
```

Available hooks: `AddBeforeInsert`, `AddAfterInsert`, `AddBeforeUpdate`, `AddAfterUpdate`, `AddBeforeDelete`, `AddAfterDelete`.

---

### Full Example

```go
package main

import (
    "github.com/cgalvisleon/et/et"
    "github.com/josefina/internal/catalog"
)

func main() {
    db, _ := catalog.NewDb("shop")
    model, _ := db.NewModel("public", "product", false, 1)

    model.DefineAtrib("id",       catalog.TpKey,   "")
    model.DefineAtrib("name",     catalog.TpText,  "")
    model.DefineAtrib("price",    catalog.TpFloat, 0.0)
    model.DefineAtrib("category", catalog.TpText,  "")
    model.DefinePrimaryKeys("id")
    model.DefineIndexes("name", "price", "category")
    model.Init()

    // Insert
    model.PutObject("p1", et.Json{"id":"p1","name":"Laptop","price":999.99,"category":"electronics"})
    model.PutObject("p2", et.Json{"id":"p2","name":"Mouse","price":29.99,"category":"electronics"})
    model.PutObject("p3", et.Json{"id":"p3","name":"Desk","price":249.00,"category":"furniture"})

    // price <= 100
    pks := model.LTEIndex("price", catalog.KeyFloat(100.0), true)

    // category = electronics
    pks, _ = model.GetByIndex("category", catalog.KeyString("electronics"))

    // price between 50 and 500
    pks = model.RangeIndex("price", catalog.KeyFloat(50.0), catalog.KeyFloat(500.0), true)

    _ = pks
}
```
