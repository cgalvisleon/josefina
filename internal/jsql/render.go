package jsql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	lg "github.com/cgalvisleon/et/stdrout"
)

// colorPrint writes a pre-colored string to stdout and appends a true ANSI
// reset (\033[0m) so that the terminal color never bleeds into the next line.
func colorPrint(s string) {
	fmt.Printf("%s\033[0m\n", s)
}

func printBanner(username, host, db string) {
	w := lg.Color(nil, lg.Blue, "\n╔══════════════════════════════════╗")
	lg.Color(w, lg.Blue, "\n║   Josefina Database Engine        ║")
	lg.Color(w, lg.Blue, "\n║   Type \\help for commands         ║")
	lg.Color(w, lg.Blue, "\n╚══════════════════════════════════╝\n")
	colorPrint(*w)
	fmt.Printf("Connected as \033[33m%s\033[0m to \033[33m%s\033[0m @ %s\n", username, db, host)
}

func printError(message string) {
	w := lg.Color(nil, lg.Red, "ERROR:  "+message)
	colorPrint(*w)
}

func printInfo(message string) {
	w := lg.Color(nil, lg.Cyan, message)
	colorPrint(*w)
}

func printSuccess(message string) {
	w := lg.Color(nil, lg.Green, message)
	colorPrint(*w)
}

func printHelp() {
	w := lg.Color(nil, lg.Yellow, "\nAvailable commands:")
	colorPrint(*w)
	cmds := [][]string{
		{`\c <db>`, "Connect to (switch) database"},
		{`SET SQL STATE <dialect>`, "Switch SQL dialect (JOSEFINA, POSTGRESQL, MYSQL, ORACLE, SQLSERVER)"},
		{`\timing`, "Toggle query timing"},
		{`\i <file>`, "Execute SQL from file"},
		{`\q`, "Quit"},
		{`\help`, "Show this help"},
	}
	rows := make([][]string, 0, len(cmds)+1)
	rows = append(rows, []string{"Command", "Description"})
	rows = append(rows, cmds...)
	printTable(rows)
	w = lg.Color(nil, lg.Yellow, "\nEnd SQL statements with ';'\n")
	colorPrint(*w)
}

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
