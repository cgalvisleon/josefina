# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Run server (single node)
```bash
gofmt -w . && go run ./cmd/server -port 1377 -http 3500
```

### Run multiple server nodes (cluster)
```bash
go run ./cmd/server -port 1377 -http 3500
go run ./cmd/server -port 1378 -http 3501
go run ./cmd/server -port 1379 -http 3502
```

### Run client (REPL)
```bash
# Remote (TCP) mode — requires a running server
go run ./cmd/client -host 127.0.0.1:1377 -user <username> -password <password>

# Local (embedded) mode — boots jdb in-process, no server needed
go run ./cmd/client -user admin -password secret -database mydb
```

### Test
```bash
go test ./internal/stmt/...          # parser + dialect tests (fast, no server needed)
go test ./...                        # all packages
go test -run TestName ./internal/stmt/  # single test
```

### Build
```bash
go build ./cmd/server
go build ./cmd/client
```

### Format
```bash
gofmt -w .
```

### Go version
```bash
goenv local 1.23.0   # project uses Go 1.23.0
```

## Code style

### Comments
All doc comments for functions, methods, and types must use this block style:

```go
/**
* FunctionName: Brief description.
* @param paramName type
* @return type
**/
```

- Use `@param` for each parameter and `@return` for the return value(s).
- Inline comments inside function bodies stay as `//`.
- Never use single-line `//` doc comments above a function or type declaration.

## Architecture

Josefina is a custom distributed document database engine written in Go with SQL-like query syntax. It has no external database dependency — all data is stored locally in its own file-based storage engine.

### Entry points
- `cmd/server/main.go` — starts the database server. Flags: `-port` (TCP, default 1377), `-http` (HTTP, default 3500), `-strict` (schema enforcement)
- `cmd/client/main.go` — starts an interactive REPL; omit `-host` for local embedded mode

### Layer breakdown

**`internal/store`** — Low-level WAL file storage engine
- `FileStore`: append-only segmented file store with an in-memory index (`map[string]*RecordRef`)
- Records are written to segment files (`segment-XXXXXX.dat`) and indexed by string key
- Tombstone-based deletions; automatic compaction when tombstones exceed 10% of index size
- Snapshots created on each segment roll-over
- Config: `RELSEG_SIZE` (segment size in MB, default 128), `SYNC_ON_WRITE` (default true)

**`internal/jdb`** — Schema, catalog, database engine, and distributed node (all in one package)
- `DB` → `Schema` → `Model` → `Field` hierarchy; catalog for a DB lives at `.catalog/.catalog`
- `Model` owns a primary `FileStore` (key → document) plus one `FileStore` per secondary index and one in-memory B+ tree (`btree.go`, degree 32) per index — rebuilt from the FileStore on startup
- `BTree` secondary indexes return `[]string` (primary keys) and support `Get`, `Range`, `GT`, `GTE`, `LT`, `LTE`, `NotEqual`
- `Node` owns all databases, sessions, and the TCP transport (`tcp *tcp.Node`); loaded via `jdb.Load(NodeParams)`
- `Vm` wraps `goja.Runtime` for JavaScript trigger execution in `Model` before/after insert/update/delete hooks; globals: `console`, `fetch`, `toJson`, `toString`, `getModel`
- Core system models (dbs, models, users, sessions, errors, transactions) are stored inside a special `.catalog` database

**`internal/stmt`** — Query language parser and SQL dialect translators
- Custom lexer (`lexer.go`) and per-statement parsers: `parser_ddl.go`, `parser_dml.go`, `parser_db.go`, `parser_user.go`, `parser_serie.go`, `parser_cache.go`, `parser_tx.go`, `parser_cmd.go`, `parser_json.go`, `parser_text.go`
- Statement types defined in `stmt_*.go` files; execution entry point is `ExecSQL(db, sql) (et.Items, error)` in `executor.go`
- Dialect subdirs: `internal/stmt/mysql`, `internal/stmt/oracle`, `internal/stmt/sqlserver` — each implement DDL/DML/TX translators and have their own tests
- SQL state can be switched per session: `SET SQL STATE MYSQL|POSTGRESQL|ORACLE|SQLSERVER|JOSEFINA`

**`internal/jsql`** — Public server/client/auth layer wrapping `internal/jdb`
- `server.go`: singleton `Server` embedding `*jdb.Node`; created with `NewServer(port)`
- `client.go`: TCP client connecting to a node
- `auth.go`, `session.go`: authentication middleware
- `dialect.go`: active SQL dialect for the session

**`internal/server`** — Top-level server wiring (HTTP + WebSocket + TCP started from here)

**`internal/cli`** — Terminal rendering for the REPL (table formatting, output)

**`internal/client`** — REPL input loop and meta-commands (`\c`, `\timing`, `\i`, `\q`)

**`pkg/http`** — HTTP API layer using `go-chi/chi`

**`pkg/websocket`** — WebSocket hub using `gorilla/websocket`

### Configuration
- `config.json` — cluster peer addresses and `is_strict` mode
- `.env` — environment variables: `TENNANT_NAME`, `TENNANT_PATH_DATA`, `PORT`, `HTTP`, `RELSEG_SIZE`, `SYNC_ON_WRITE`, `DEBUG`
- Data is persisted under `./data/<TENNANT_NAME>/dbs/<database>/`

### Key dependency
- `github.com/cgalvisleon/et` — shared utilities providing: `tcp` (transport), `et` (JSON type), `claim` (JWT), `logs`, `envar`, `utility`, `reg`, `ws`, `vm` (goja wrapper)
