package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	lg "github.com/cgalvisleon/et/stdrout"
)

/**
* printBanner: Prints the welcome banner.
* @param username, db string
**/
func printBanner(username, db string) {
	w := lg.Color(nil, lg.Blue, "\n╔══════════════════════════════════╗")
	lg.Color(w, lg.Blue, "\n║   Josefina Database Engine        ║")
	lg.Color(w, lg.Blue, "\n║   Type \\help for commands         ║")
	lg.Color(w, lg.Blue, "\n╚══════════════════════════════════╝\n")
	println(*w)
	info := fmt.Sprintf("Connected as %s%s%s to %s%s%s\n",
		lg.Yellow, username, lg.Reset, lg.Yellow, db, lg.Reset)
	fmt.Print(info)
}

/**
* printError: Prints an error message in red.
* @param message string
**/
func printError(message string) {
	w := lg.Color(nil, lg.Red, "ERROR:  "+message)
	println(*w)
}

/**
* printInfo: Prints an informational message in cyan.
* @param message string
**/
func printInfo(message string) {
	w := lg.Color(nil, lg.Cyan, message)
	println(*w)
}

/**
* printSuccess: Prints a success message in green.
* @param message string
**/
func printSuccess(message string) {
	w := lg.Color(nil, lg.Green, message)
	println(*w)
}

/**
* printHelp: Prints the available backslash commands.
**/
func printHelp() {
	w := lg.Color(nil, lg.Yellow, "\nAvailable commands:")
	println(*w)

	cmds := [][]string{
		{`\l`, "List databases"},
		{`\c <db>`, "Connect to database"},
		{`\dt [schema]`, "List tables"},
		{`\d <table>`, "Describe table (schema.table or just table)"},
		{`\du`, "List users"},
		{`\timing`, "Toggle query timing"},
		{`\backup [db] <path>`, "Backup database to path as NDJSON files"},
		{`\i <file>`, "Execute SQL from file"},
		{`\q`, "Quit"},
		{`\help`, "Show this help"},
	}
	rows := make([][]string, 0, len(cmds)+1)
	rows = append(rows, []string{"Command", "Description"})
	rows = append(rows, cmds...)
	printTable(rows)

	w = lg.Color(nil, lg.Yellow, "\nEnd SQL statements with ';'\n")
	println(*w)
}

// ── ASCII table ───────────────────────────────────────────────────────────────

/**
* printTable: Renders a slice of string rows as a bordered ASCII table.
* rows[0] is treated as the header row.
* @param rows [][]string
**/
func printTable(rows [][]string) {
	if len(rows) == 0 {
		return
	}
	cols := len(rows[0])
	widths := make([]int, cols)
	for _, row := range rows {
		for j := 0; j < cols && j < len(row); j++ {
			if l := len(row[j]); l > widths[j] {
				widths[j] = l
			}
		}
	}

	sep := buildSep(widths)
	fmt.Println(sep)
	printRow(rows[0], widths)
	fmt.Println(sep)
	for _, row := range rows[1:] {
		printRow(row, widths)
	}
	fmt.Println(sep)
}

func buildSep(widths []int) string {
	var b strings.Builder
	b.WriteByte('+')
	for _, w := range widths {
		b.WriteString(strings.Repeat("-", w+2))
		b.WriteByte('+')
	}
	return b.String()
}

func printRow(row []string, widths []int) {
	var b strings.Builder
	b.WriteByte('|')
	for j, w := range widths {
		cell := ""
		if j < len(row) {
			cell = row[j]
		}
		b.WriteByte(' ')
		b.WriteString(cell)
		b.WriteString(strings.Repeat(" ", w-len(cell)))
		b.WriteString(" |")
	}
	fmt.Println(b.String())
}

// ── Items renderer ────────────────────────────────────────────────────────────

/**
* printItems: Renders et.Items as a bordered ASCII table.
* @param items et.Items
**/
func printItems(items et.Items) {
	if items.Count == 0 {
		printInfo("(0 rows)")
		return
	}

	keys := collectKeys(items.Result)
	rows := make([][]string, 0, items.Count+1)
	rows = append(rows, keys)

	for _, item := range items.Result {
		row := make([]string, len(keys))
		for i, k := range keys {
			v := item[k]
			if v == nil {
				row[i] = ""
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		rows = append(rows, row)
	}

	printTable(rows)
	printInfo(fmt.Sprintf("(%d rows)", items.Count))
}

func collectKeys(results []et.Json) []string {
	seen := make(map[string]bool)
	keys := make([]string, 0)
	for _, item := range results {
		for k := range item {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}
