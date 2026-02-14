package jdb

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Console struct {
	addr   string
	client *Session
}

var console *Console

func startConsole(client *Session) {
	if console == nil {
		console = &Console{
			client: client,
		}
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("===================================")
	fmt.Println("  TCP Client Console")
	fmt.Println("  Escribe 'help' para comandos")
	fmt.Println("===================================")

	for {
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error leyendo comando:", err)
			continue
		}

		input = strings.TrimSpace(input)
		console.handleCommand(input)
	}
}

func (s *Console) handleCommand(cmd string) {
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
