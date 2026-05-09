package jsql

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/stmt"
)

// Client is an interactive REPL that connects to a jsql server over TCP.
type Client struct {
	conn      *tcp.Client
	host      string
	username  string
	database  string
	sessionID string
	dialect   string
	timing    bool
	buf       strings.Builder
	reader    *bufio.Reader
}

/**
* NewClient: connects to a jsql server and authenticates the session.
* @param host, username, database string
* @return *Client, error
**/
func NewClient(host, username, database string) (*Client, error) {
	conn := tcp.NewClient(host)
	if err := conn.Connect(); err != nil {
		return nil, err
	}

	c := &Client{
		conn:     conn,
		host:     host,
		username: username,
		database: database,
		dialect:  string(stmt.DialectJosefina),
		reader:   bufio.NewReader(os.Stdin),
	}

	res := conn.Request("QueryService.Connect", username, database, host)
	if res.Error != nil {
		conn.Close()
		return nil, res.Error
	}
	var sid string
	if err := res.Get(&sid); err != nil {
		conn.Close()
		return nil, err
	}
	c.sessionID = sid

	return c, nil
}

/**
* Start: runs the interactive REPL loop.
**/
func (s *Client) Start() {
	printBanner(s.username, s.host, s.database)

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

func (s *Client) prompt() string {
	db := s.database
	if db == "" {
		db = "josefina"
	}
	if s.buf.Len() == 0 {
		return fmt.Sprintf("%s=# ", db)
	}
	return fmt.Sprintf("%s-# ", db)
}

func hasTerminator(sql string) bool {
	return strings.HasSuffix(strings.TrimSpace(sql), ";")
}

// ── Meta commands ─────────────────────────────────────────────────────────────

func (s *Client) handleMeta(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}
	switch parts[0] {
	case `\c`, `\connect`:
		if len(parts) < 2 {
			printError(`Usage: \c <database>`)
			return
		}
		s.database = parts[1]
		printSuccess(fmt.Sprintf(`You are now connected to database "%s".`, s.database))
	case `\timing`:
		s.timing = !s.timing
		if s.timing {
			printInfo("Timing is on.")
		} else {
			printInfo("Timing is off.")
		}
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

// ── SQL execution ─────────────────────────────────────────────────────────────

func (s *Client) execSQL(sql string) {
	sql = strings.TrimRight(strings.TrimSpace(sql), ";")
	if sql == "" {
		return
	}

	// Intercept SET SQL STATE locally to update session dialect on the server.
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(upper, "SET SQL STATE ") {
		parts := strings.Fields(sql)
		if len(parts) < 4 {
			printError("Syntax: SET SQL STATE <JOSEFINA|POSTGRESQL|MYSQL|ORACLE|SQLSERVER>")
			return
		}
		s.setDialect(parts[3])
		return
	}

	start := time.Now()
	args := QueryArgs{
		SQL:       sql,
		Database:  s.database,
		SessionID: s.sessionID,
	}
	res := s.conn.Request("QueryService.Exec", args)
	elapsed := time.Since(start)

	if res.Error != nil {
		printError(res.Error.Error())
	} else {
		var items et.Items
		if err := res.Get(&items); err != nil {
			printError(err.Error())
		} else if items.Count == 0 {
			printSuccess("OK")
		} else {
			printItems(items)
		}
	}

	if s.timing {
		printInfo(fmt.Sprintf("Time: %.3f ms", float64(elapsed.Microseconds())/1000.0))
	}
}

func (s *Client) setDialect(dialect string) {
	d := strings.ToUpper(dialect)
	res := s.conn.Request("QueryService.SetDialect", s.sessionID, d)
	if res.Error != nil {
		printError(res.Error.Error())
		return
	}
	s.dialect = d
	printSuccess(fmt.Sprintf("SQL dialect set to %s.", s.dialect))
}

func (s *Client) execFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		printError(err.Error())
		return
	}
	s.execSQL(string(data))
}
