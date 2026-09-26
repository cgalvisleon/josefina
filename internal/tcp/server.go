package tcp

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/msg"
)

type Server struct {
	timeout  *duration            // compartido con todas las conexiones
	hs       *handlers            // funciones compartidas por todas las conexiones
	mu       sync.Mutex           // protege listener y peers
	listener net.Listener         // nil si no está escuchando
	peers    map[*Client]struct{} // conexiones abiertas
	wg       sync.WaitGroup       // acceptLoop y el readLoop de cada conexión
}

/**
* newServer: Crea un servidor que todavía no escucha.
* @return *Server, error
**/
func newServer() (*Server, error) {
	return &Server{
		timeout: newDuration(defaultTimeout),
		hs:      &handlers{},
		peers:   make(map[*Client]struct{}),
	}, nil
}

/**
* listen: Empieza a escuchar en addr y acepta conexiones en segundo plano.
* Con addr ":0" el sistema elige el puerto; se consulta con addr().
* @param addr string
* @return error
**/
func (s *Server) listen(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return errors.New(msg.MSG_TCP_ALREADY_LISTENING)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = ln

	s.wg.Add(1)
	go s.acceptLoop(ln)
	return nil
}

/**
* addr: Retorna la dirección en la que escucha, o "" si no está escuchando.
* @return string
**/
func (s *Server) addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

/**
* acceptLoop: Acepta conexiones de ln hasta que se cierra.
* @param ln net.Listener
**/
func (s *Server) acceptLoop(ln net.Listener) {
	defer s.wg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			logs.Alert(err)
			time.Sleep(10 * time.Millisecond)
			continue
		}
		s.attach(ln, conn)
	}
}

/**
* attach: Atiende conn con las funciones y el timeout del servidor hasta que se cierra,
* y llama a las funciones de onConnect en su propia goroutine para no frenar acceptLoop.
* @param ln net.Listener, conn net.Conn
**/
func (s *Server) attach(ln net.Listener, conn net.Conn) {
	l := newLink(conn, s.timeout)
	peer := &Client{timeout: s.timeout, hs: s.hs, link: l}

	s.mu.Lock()
	if s.listener != ln {
		s.mu.Unlock()
		conn.Close()
		return
	}
	s.peers[peer] = struct{}{}
	s.wg.Add(1)
	s.mu.Unlock()

	go s.hs.handleConnect(peer)
	go func() {
		defer s.wg.Done()
		l.readLoop(peer, s.hs)
		peer.detach(l)
		s.mu.Lock()
		delete(s.peers, peer)
		s.mu.Unlock()
		s.hs.handleDisconnect(peer, l.err)
	}()
}

/**
* close: Deja de escuchar, cierra todas las conexiones y espera a que terminen.
* @return error
**/
func (s *Server) close() error {
	s.mu.Lock()
	ln := s.listener
	s.listener = nil
	peers := make([]*Client, 0, len(s.peers))
	for peer := range s.peers {
		peers = append(peers, peer)
	}
	s.mu.Unlock()

	if ln == nil {
		return nil
	}
	err := ln.Close()
	for _, peer := range peers {
		peer.close()
	}
	s.wg.Wait()
	return err
}

/**
* count: Retorna el número de conexiones abiertas.
* @return int
**/
func (s *Server) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.peers)
}

/**
* setTimeout: Cambia el timeout de escritura y Request; aplica también a las conexiones abiertas.
* @param timeout time.Duration
* @return error
**/
func (s *Server) setTimeout(timeout time.Duration) error {
	return s.timeout.set(timeout)
}

/**
* onConnect: Agrega fn a las funciones que se llaman con cada conexión aceptada. conn es
* la conexión con ese cliente: el servidor puede usar conn.Send y conn.Request para
* enviarle mensajes, y guardarla para usarla después.
* @param fn func(conn *Client)
**/
func (s *Server) onConnect(fn func(conn *Client)) {
	s.hs.onConnect(fn)
}

/**
* onDisconnect: Agrega fn a las funciones que se llaman cuando se cierra cualquier
* conexión, la cierre el cliente o el servidor.
* @param fn func(conn *Client, err error)
**/
func (s *Server) onDisconnect(fn func(conn *Client, err error)) {
	s.hs.onDisconnect(fn)
}

/**
* onSend: Agrega fn a las funciones que reciben los mensajes tipo send de cualquier conexión.
* @param fn func(conn *Client, message []byte)
**/
func (s *Server) onSend(fn func(conn *Client, message []byte)) {
	s.hs.onSend(fn)
}

/**
* onRequest: Agrega fn a las funciones que atienden los mensajes tipo request de cualquier conexión.
* @param fn func(conn *Client, message []byte) ([]byte, error)
**/
func (s *Server) onRequest(fn func(conn *Client, message []byte) ([]byte, error)) {
	s.hs.onRequest(fn)
}

/**
* onSendStream: Define la función que recibe los SendStream de cualquier conexión.
* @param fn func(conn *Client, in *StreamReader)
**/
func (s *Server) onSendStream(fn func(conn *Client, in *StreamReader)) {
	s.hs.setSendStream(fn)
}

/**
* onRequestStream: Define la función que atiende los RequestStream de cualquier conexión.
* @param fn func(conn *Client, in *StreamReader) ([]byte, error)
**/
func (s *Server) onRequestStream(fn func(conn *Client, in *StreamReader) ([]byte, error)) {
	s.hs.setRequestStream(fn)
}

/**
* onReceiveStream: Define la función que atiende los ReceiveStream de cualquier conexión.
* @param fn func(conn *Client, message []byte, out *StreamWriter) error
**/
func (s *Server) onReceiveStream(fn func(conn *Client, message []byte, out *StreamWriter) error) {
	s.hs.setReceiveStream(fn)
}
