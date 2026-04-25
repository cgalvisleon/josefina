# Josefina

Custom distributed document database engine written in Go with SQL-like query syntax.

## Setup

```bash
goenv local 1.23.0
go mod init github.com/cgalvisleon/josefina
go get github.com/cgalvisleon/et@v1.0.14
go get github.com/gorilla/websocket
git remote add origin https://github.com/cgalvisleon/josefina.git
```

## Server

```bash
# Single node
gofmt -w . && go run ./cmd/server -tcp-port 1377 -http-port 3500

# Cluster (3 nodes)
gofmt -w . && go run ./cmd/server -port 3500 -rpc 4300
gofmt -w . && go run ./cmd/server -port 3501 -rpc 4301
gofmt -w . && go run ./cmd/server -port 3502 -rpc 4302

# Client REPL
gofmt -w . && go run ./cmd/client -host 127.0.0.1:1377 -user cgalvisl -password 123456
```

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

| Constant | Description |
|---|---|
| `TpKey` | String identifier (UUID, etc.) |
| `TpText` / `TpMemo` | Short / long text |
| `TpInt` / `TpAutoIncrement` | Integer |
| `TpFloat` | Floating point |
| `TpBoolean` | Boolean |
| `TpDateTime` | Timestamp (RFC3339) |
| `TpJson` | Nested JSON object |
| `TpAny` | Any value |

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

| Method | Operator | Example |
|---|---|---|
| `GTIndex(field, key, asc)` | `>` | `model.GTIndex("age", KeyInt(28), true)` |
| `GTEIndex(field, key, asc)` | `>=` | `model.GTEIndex("age", KeyInt(25), true)` |
| `LTIndex(field, key, asc)` | `<` | `model.LTIndex("age", KeyInt(30), true)` |
| `LTEIndex(field, key, asc)` | `<=` | `model.LTEIndex("age", KeyInt(30), true)` |
| `NotEqualIndex(field, key)` | `!=` | `model.NotEqualIndex("name", KeyString("Bob"))` |

---

### `IndexKey` — Typed Keys

Keys carry type information so ordering is always correct (numeric for numbers, lexicographic for strings).

| Constructor | Go type | Ordering |
|---|---|---|
| `KeyString(v string)` | string | Lexicographic |
| `KeyInt(v int64)` | int64 | Numeric |
| `KeyFloat(v float64)` | float64 | Numeric |
| `KeyBool(v bool)` | bool | false < true |
| `KeyDateTime(v time.Time)` | time.Time | Chronological |
| `KeyFromAny(v any)` | any (JSON) | Auto-detect |

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
    "github.com/cgalvisleon/josefina/internal/catalog"
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
