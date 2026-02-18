package sql

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cgalvisleon/et/logs"
	lg "github.com/cgalvisleon/et/stdrout"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/jdb"
)

type Client struct {
	*tcp.Client
	addr     string
	session  *jdb.Session
	username string
	database string
}

/**
* NewClient
* @param host, username, database string
* @return *Client
**/
func NewClient(host, username, database string) (*Client, error) {
	client := tcp.NewClient(host)
	err := client.Connect()
	if err != nil {
		return nil, err
	}

	result := &Client{
		Client:   client,
		addr:     host,
		username: username,
		database: database,
	}

	return result, nil
}

/**
* Start
* @return
**/
func (s *Client) Start() {
	res := s.Request("Lead.GetDb", "test")
	if res.Error != nil {
		logs.Error(res.Error)
	}

	var db *catalog.DB
	err := res.Get(&db)
	if err != nil {
		logs.Error(err)
	}

	reader := bufio.NewReader(os.Stdin)

	w := lg.Color(nil, lg.Blue, "\n===================================")
	lg.Color(w, lg.Blue, "\n  TCP Client Console")
	lg.Color(w, lg.Blue, "\n  Escribe 'help' para comandos")
	lg.Color(w, lg.Blue, "\n===================================")
	println(*w)

	for {
		w := lg.Color(nil, lg.Reset, "> ")
		fmt.Print(*w)

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error leyendo comando:", err)
			continue
		}

		input = strings.TrimSpace(input)
		s.handleCommand(input)
	}
}

/**
* handleCommand
* @param cmd string
**/
func (s *Client) handleCommand(cmd string) {
	args := strings.Split(cmd, " ")

	switch args[0] {

	case "help":
		fmt.Println("Comandos disponibles:")
		fmt.Println("  help        - mostrar comandos")
		fmt.Println("  nodes       - listar nodos")
		fmt.Println("  clients     - listar clientes")
		fmt.Println("  leader      - mostrar líder")
		fmt.Println("  stats       - estadísticas")
		fmt.Println("  stop        - detener servidor")

	case "nodes":
		fmt.Println("Nodos:")

	case "clients":
		fmt.Println("Clientes:")
		fmt.Println("  ", s.addr)
	case "leader":
		fmt.Println("Leader:", s.addr)

	case "stats":
		fmt.Println("Estadísticas:")
		fmt.Println("  Nodos:", 0)
		fmt.Println("  Clientes:", 1)
		fmt.Println("  Líder:", s.addr)

	case "stop":
		fmt.Println("Deteniendo cliente...")
		os.Exit(0)

	default:
		fmt.Println("Comando no reconocido:", cmd)
	}
}
