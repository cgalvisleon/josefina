package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/jdb"
	stmt "github.com/cgalvisleon/josefina/internal/stmt"
)

// CLI is an interactive psql-style terminal backed directly by the jdb engine.
type CLI struct {
	node     *jdb.Node
	db       *jdb.DB
	dbName   string
	username string
	timing   bool
	buf      strings.Builder
	reader   *bufio.Reader
}

/**
* New: Loads the jdb engine and authenticates the user.
* @param dataPath, username, password, database string
* @return (*CLI, error)
**/
func New(dataPath, username, password, database string) (*CLI, error) {
	if dataPath != "" {
		os.Setenv("DATA_PATH", dataPath)
	}

	if _, err := jdb.Load(); err != nil {
		return nil, err
	}

	node, err := jdb.GetNode()
	if err != nil {
		return nil, err
	}

	// Bootstrap: if no users exist yet, create the first user automatically.
	existing, err := node.ListUsers()
	if err != nil {
		return nil, err
	}
	if existing.Count == 0 {
		if _, err = node.CreateUser(username, password); err != nil {
			return nil, fmt.Errorf("bootstrap: could not create user %q: %w", username, err)
		}
		fmt.Printf("Bootstrap: user %q created.\n", username)
	} else {
		user, err := node.GetUser(username, password)
		if err != nil {
			return nil, err
		}
		if !user.Ok {
			return nil, errors.New("authentication failed: invalid username or password")
		}
	}

	c := &CLI{
		node:     node,
		dbName:   "josefina",
		username: username,
		reader:   bufio.NewReader(os.Stdin),
	}

	if database != "" {
		db, dbErr := node.GetDb(database)
		if dbErr != nil {
			fmt.Printf("Warning: database %q not found; use \\c <db> to connect.\n", database)
		} else {
			c.db = db
			c.dbName = database
		}
	}

	return c, nil
}

/**
* Start: Runs the interactive REPL loop.
**/
func (s *CLI) Start() {
	printBanner(s.username, s.dbName)

	for {
		fmt.Print(s.prompt())

		line, err := s.reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")

		if s.buf.Len() == 0 {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, `\`) {
				s.handleMeta(trimmed)
				continue
			}
			if trimmed == "exit" || trimmed == "quit" {
				fmt.Println("Bye.")
				return
			}
		}

		if s.buf.Len() > 0 {
			s.buf.WriteByte('\n')
		}
		s.buf.WriteString(line)

		if hasTerminator(s.buf.String()) {
			sql := s.buf.String()
			s.buf.Reset()
			s.execSQL(sql)
		}
	}
}

// prompt returns the appropriate prompt string based on buffer state.
func (s *CLI) prompt() string {
	db := s.dbName
	if db == "" {
		db = "josefina"
	}
	if s.buf.Len() == 0 {
		return fmt.Sprintf("%s=# ", db)
	}
	return fmt.Sprintf("%s-# ", db)
}

// hasTerminator reports whether sql ends with a semicolon (ignoring whitespace).
func hasTerminator(sql string) bool {
	return strings.HasSuffix(strings.TrimSpace(sql), ";")
}

// ── Meta commands ─────────────────────────────────────────────────────────────

func (s *CLI) handleMeta(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case `\l`, `\list`:
		s.listDatabases()
	case `\c`, `\connect`:
		if len(parts) < 2 {
			printError(`Usage: \c <database>`)
			return
		}
		s.connectDB(parts[1])
	case `\dt`:
		s.listTables(parts[1:])
	case `\d`:
		if len(parts) < 2 {
			printError(`Usage: \d <table>  or  \d <schema>.<table>`)
			return
		}
		s.describeTable(parts[1])
	case `\du`:
		s.listUsers()
	case `\timing`:
		s.timing = !s.timing
		if s.timing {
			printInfo("Timing is on.")
		} else {
			printInfo("Timing is off.")
		}
	case `\backup`:
		s.backup(parts[1:])
	case `\i`:
		if len(parts) < 2 {
			printError(`Usage: \i <file>`)
			return
		}
		s.execFile(parts[1])
	case `\q`, `\quit`:
		fmt.Println("Bye.")
		os.Exit(0)
	case `\help`, `\h`, `\?`:
		printHelp()
	default:
		printError(fmt.Sprintf("Unknown command: %s  (type \\help for available commands)", parts[0]))
	}
}

// ── \l — list databases ───────────────────────────────────────────────────────

func (s *CLI) listDatabases() {
	rows := [][]string{{"Name", "Strict"}}
	for name, db := range s.node.DBS {
		strict := "no"
		if db.IsStrict {
			strict = "yes"
		}
		rows = append(rows, []string{name, strict})
	}
	printTable(rows)
	printInfo(fmt.Sprintf("(%d rows)", len(rows)-1))
}

// ── \c — connect to database ──────────────────────────────────────────────────

func (s *CLI) connectDB(name string) {
	db, err := s.node.GetDb(name)
	if err != nil {
		printError(err.Error())
		return
	}
	s.db = db
	s.dbName = name
	printSuccess(fmt.Sprintf(`You are now connected to database "%s" as user "%s".`, name, s.username))
}

// ── \dt — list tables ─────────────────────────────────────────────────────────

func (s *CLI) listTables(args []string) {
	if s.db == nil {
		printError(`No database selected. Use \c <database>`)
		return
	}

	schemaFilter := ""
	if len(args) > 0 {
		schemaFilter = args[0]
	}

	rows := [][]string{{"Schema", "Name", "Type"}}
	for _, schema := range s.db.ListSchemas() {
		if schemaFilter != "" && !strings.EqualFold(schema.Name, schemaFilter) {
			continue
		}
		for _, model := range schema.ListModels() {
			if model.IsCore && schemaFilter == "" {
				continue
			}
			tp := "table"
			rows = append(rows, []string{schema.Name, model.Name, tp})
		}
	}

	if len(rows) == 1 {
		printInfo("No tables found.")
		return
	}
	printTable(rows)
	printInfo(fmt.Sprintf("(%d rows)", len(rows)-1))
}

// ── \d — describe table ───────────────────────────────────────────────────────

func (s *CLI) describeTable(name string) {
	if s.db == nil {
		printError(`No database selected. Use \c <database>`)
		return
	}

	schemaName := "public"
	tableName := name
	if idx := strings.Index(name, "."); idx != -1 {
		schemaName = name[:idx]
		tableName = name[idx+1:]
	}

	model, err := s.db.GetModel(schemaName, tableName)
	if err != nil {
		printError(err.Error())
		return
	}

	fmt.Printf("\nTable \"%s.%s\"\n", model.Schema, model.Name)

	requiredSet := make(map[string]bool)
	for _, r := range model.Required {
		requiredSet[r.Name] = true
	}
	uniqueSet := make(map[string]bool)
	for _, u := range model.Unique {
		uniqueSet[u.Name] = true
	}

	rows := [][]string{{"Column", "Type", "Not Null", "Primary Key", "Unique", "Default"}}
	for fieldName, field := range model.Fields {
		if field.TypeField != jdb.TpAtrib {
			continue
		}
		notNull := boolStr(requiredSet[fieldName])
		isPK := boolStr(slices.Contains(model.PrimaryKeys, fieldName))
		isUniq := boolStr(uniqueSet[fieldName])
		def := fmt.Sprintf("%v", field.DefaultValue)
		if def == "<nil>" || def == "nil" {
			def = ""
		}
		rows = append(rows, []string{fieldName, string(field.TypeData), notNull, isPK, isUniq, def})
	}
	printTable(rows)

	if len(model.Indexes) > 0 {
		fmt.Println("\nIndexes:")
		for _, idx := range model.Indexes {
			fmt.Printf("  \"%s\" (%s)\n", idx.Name, idx.Type)
		}
	}

	if len(model.ForeignKeys) > 0 {
		fmt.Println("\nForeign Keys:")
		for name := range model.ForeignKeys {
			fmt.Printf("  %s\n", name)
		}
	}
}

// ── \du — list users ──────────────────────────────────────────────────────────

func (s *CLI) listUsers() {
	items, err := s.node.ListUsers()
	if err != nil {
		printError(err.Error())
		return
	}
	if items.Count == 0 {
		printInfo("No users found.")
		return
	}
	printItems(items)
}

// ── SQL execution ─────────────────────────────────────────────────────────────

func (s *CLI) execSQL(sql string) {
	sql = strings.TrimRight(strings.TrimSpace(sql), ";")
	if sql == "" {
		return
	}

	// Intercept node-level DDL before passing to the executor.
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(upper, "CREATE DATABASE ") {
		s.execCreateDatabase(sql)
		return
	}
	if strings.HasPrefix(upper, "DROP DATABASE ") {
		s.execDropDatabase(sql)
		return
	}
	if strings.HasPrefix(upper, "CREATE USER ") {
		s.execCreateUser(sql)
		return
	}

	if s.db == nil {
		printError(`No database selected. Use \c <database>  or  CREATE DATABASE <name>`)
		return
	}

	start := time.Now()
	items, err := stmt.ExecSQL(s.db, sql)
	elapsed := time.Since(start)

	if err != nil {
		printError(err.Error())
	} else if items.Count == 0 {
		printSuccess("OK")
	} else {
		printItems(items)
	}

	if s.timing {
		printInfo(fmt.Sprintf("Time: %.3f ms", float64(elapsed.Microseconds())/1000.0))
	}
}

func (s *CLI) execCreateDatabase(sql string) {
	parts := strings.Fields(sql)
	if len(parts) < 3 {
		printError("Syntax: CREATE DATABASE <name>")
		return
	}
	name := strings.ToLower(parts[2])
	db, err := s.node.CreateDb(name)
	if err != nil {
		printError(err.Error())
		return
	}
	s.db = db
	s.dbName = name
	printSuccess(fmt.Sprintf(`Database "%s" created.`, name))
}

func (s *CLI) execDropDatabase(sql string) {
	parts := strings.Fields(sql)
	if len(parts) < 3 {
		printError("Syntax: DROP DATABASE <name>")
		return
	}
	name := strings.ToLower(parts[2])
	if err := s.node.DropDb(name); err != nil {
		printError(err.Error())
		return
	}
	if s.dbName == name {
		s.db = nil
		s.dbName = "josefina"
	}
	printSuccess(fmt.Sprintf(`Database "%s" dropped.`, name))
}

func (s *CLI) execCreateUser(sql string) {
	// Syntax: CREATE USER <name> PASSWORD '<pass>'
	upper := strings.ToUpper(sql)
	pwIdx := strings.Index(upper, " PASSWORD ")
	if pwIdx == -1 {
		printError("Syntax: CREATE USER <name> PASSWORD '<password>'")
		return
	}
	namePart := strings.TrimSpace(sql[len("CREATE USER"):pwIdx])
	passPart := strings.TrimSpace(sql[pwIdx+len(" PASSWORD "):])
	passPart = strings.Trim(passPart, `'"`)
	if namePart == "" || passPart == "" {
		printError("Syntax: CREATE USER <name> PASSWORD '<password>'")
		return
	}
	if _, err := s.node.CreateUser(namePart, passPart); err != nil {
		printError(err.Error())
		return
	}
	printSuccess(fmt.Sprintf(`User "%s" created.`, namePart))
}

// ── \i — execute file ─────────────────────────────────────────────────────────

func (s *CLI) execFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		printError(err.Error())
		return
	}
	s.execSQL(string(data))
}

// ── \backup — backup database ─────────────────────────────────────────────────

/**
* backup: Exports all non-core models as NDJSON files under <path>/<dbname>/<schema>/<model>.ndjson
* @param args []string
**/
func (s *CLI) backup(args []string) {
	db := s.db
	dbName := s.dbName
	basePath := "."

	switch len(args) {
	case 1:
		basePath = args[0]
	case 2:
		var err error
		dbName = args[0]
		db, err = s.node.GetDb(dbName)
		if err != nil {
			printError(err.Error())
			return
		}
		basePath = args[1]
	}

	if db == nil {
		printError(`No database selected. Use \c <database>  or  \backup <db> <path>`)
		return
	}

	backupRoot := filepath.Join(basePath, dbName)
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		printError(err.Error())
		return
	}

	tableCount := 0
	for _, schema := range db.ListSchemas() {
		for _, model := range schema.ListModels() {
			if model.IsCore {
				continue
			}

			schemaDir := filepath.Join(backupRoot, schema.Name)
			if err := os.MkdirAll(schemaDir, 0o755); err != nil {
				printError(err.Error())
				return
			}

			filePath := filepath.Join(schemaDir, model.Name+".ndjson")
			f, err := os.Create(filePath)
			if err != nil {
				printError(err.Error())
				return
			}

			enc := json.NewEncoder(f)
			cursor, err := model.NewCursor(true, 0, 0)
			if err != nil {
				f.Close()
				printError(err.Error())
				return
			}

			rowCount := 0
			for cursor.Next() {
				var record et.Json
				if scanErr := cursor.Scan(&record); scanErr != nil || record == nil {
					continue
				}
				enc.Encode(record)
				rowCount++
			}
			cursor.Close()
			f.Close()

			printInfo(fmt.Sprintf("  %s.%s: %d rows → %s", schema.Name, model.Name, rowCount, filePath))
			tableCount++
		}
	}

	printSuccess(fmt.Sprintf("Backup complete: %d tables → %s", tableCount, backupRoot))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
