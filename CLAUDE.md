# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Run server (single node)
```bash
gofmt -w . && go run ./cmd/server -tcp-port 1377 -http-port 3500
```

### Run multiple server nodes (cluster)
```bash
go run ./cmd/server -tcp-port 1377 -http-port 3500
go run ./cmd/server -tcp-port 1378 -http-port 3501
go run ./cmd/server -tcp-port 1379 -http-port 3502
```

### Run client (REPL)
```bash
go run ./cmd/client -host 127.0.0.1:1377 -user <username> -password <password>
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

## Architecture

Josefina is a custom distributed document database engine written in Go with SQL-like query syntax. It has no external database dependency — all data is stored locally in its own file-based storage engine.

### Entry points
- `cmd/server/main.go` — starts the database server (TCP + HTTP + WebSocket)
- `cmd/client/main.go` — starts an interactive REPL client connected over TCP

### Layer breakdown

**`internal/store`** — Low-level WAL file storage engine
- `FileStore`: append-only segmented file store with an in-memory index (`map[string]*RecordRef`)
- Records are written to segment files (`segment-XXXXXX.dat`) and indexed by string key
- Tombstone-based deletions; automatic compaction when tombstones exceed 10% of index size
- Snapshots created on each segment roll-over
- Config: `RELSEG_SIZE` (segment size in MB, default 128), `SYNC_ON_WRITE` (default true)

**`internal/catalog`** — Schema and metadata layer
- `DB` → `Schema` → `Model` → `Field` hierarchy
- `Model` manages multiple named `FileStore` instances (one per index)
- The primary index is always called `INDEX`; additional indexes store `map[string]bool` sets pointing to primary keys
- `Trigger` structs hold JavaScript source for before/after insert/update/delete hooks (executed via `goja`)

**`internal/jdb`** — Database engine and distributed node
- `Node` embeds `*tcp.Server` and holds all runtime state (dbs, models, sessions, cache)
- Leader/follower pattern: `Lead` handles writes and catalog mutations; `Follow` handles data replication
- `Lead` contains all top-level operations: CreateDb, CreateUser, SignIn, SetCache, CreateSerie, SaveModel, etc.
- Operations route to leader via TCP RPC if current node is not the leader
- `Vm` wraps `goja.Runtime` for JavaScript trigger execution; provides `console`, `fetch`, `toJson`, `toString`, `getModel` globals
- Core internal models (dbs, models, users, series, cache) are stored in a special `josefina` database

**`internal/stmt`** — Query language parser
- Custom lexer (`lexer.go`) and parser files per statement type: `parser_db.go`, `parser_user.go`, `parser_serie.go`, `parser_cache.go`, `parser_json.go`, `parser_text.go`
- Statement types defined in `stmt_db.go`, `stmt_model.go`, `stmt_user.go`, `stmt_serie.go`, `stmt_cache.go`, `stmt_query.go`, `stmt_cmd.go`

**`internal/client`** — Interactive REPL that connects to server over TCP

**`pkg/sql`** — Public TCP server/client wrappers around `internal/jdb`
- `pkg/sql/server.go`: wraps `jdb.Node` as a TCP listener
- `pkg/sql/client.go`: TCP client for connecting to a node
- `pkg/sql/auth.go`, `pkg/sql/session.go`: authentication middleware

**`pkg/http`** — HTTP API layer using `go-chi/chi`
**`pkg/websocket`** — WebSocket hub using `gorilla/websocket`

### Configuration
- `config.json` — cluster peer addresses and `is_strict` mode (strict = schema enforcement)
- `.env` — environment variables: `TENNANT_NAME`, `TENNANT_PATH_DATA`, `TCP_PORT`, `HTTP_PORT`, `RELSEG_SIZE`, `SYNC_ON_WRITE`, `DEBUG`
- Data is persisted under `./data/<TENNANT_NAME>/dbs/<database>/`

### Key dependency
- `github.com/cgalvisleon/et` — shared utilities library providing: `tcp` (transport), `et` (JSON type), `claim` (JWT), `logs`, `envar`, `utility`, `reg`, `ws`, `server`
