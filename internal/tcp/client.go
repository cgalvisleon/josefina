package tcp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/josefina/internal/msg"
)

type Client struct {
	timeout *duration
	nextID  atomic.Uint64
	hs      *handlers    // funciones que reciben los mensajes entrantes
	mu      sync.RWMutex // protege link
	link    *link        // conexión actual; nil si no hay
}

/**
* newClient: Crea un cliente sin conectar.
* @return *Client, error
**/
func newClient() (*Client, error) {
	return &Client{
		timeout: newDuration(defaultTimeout),
		hs:      &handlers{},
	}, nil
}

/**
* connectTo: Abre la conexión con addr, empieza a leer mensajes en segundo plano y
* llama a las funciones de onConnect antes de retornar.
* @param addr string
* @return error
**/
func (s *Client) connectTo(addr string) error {
	if _, err := s.current(); err == nil {
		return errors.New(msg.MSG_TCP_ALREADY_CONNECTED)
	}

	conn, err := net.DialTimeout("tcp", addr, s.timeout.get())
	if err != nil {
		return err
	}

	l := newLink(conn, s.timeout)
	s.mu.Lock()
	if s.link != nil {
		s.mu.Unlock()
		conn.Close()
		return errors.New(msg.MSG_TCP_ALREADY_CONNECTED)
	}
	s.link = l
	s.mu.Unlock()

	go s.readLoop(l)
	s.hs.handleConnect(s)
	return nil
}

/**
* readLoop: Lee los mensajes de l; cuando la conexión se cierra, deja el cliente sin
* conexión (para que se pueda volver a llamar a connectTo) y llama a onDisconnect.
* @param l *link
**/
func (s *Client) readLoop(l *link) {
	l.readLoop(s, s.hs)
	s.detach(l)
	s.hs.handleDisconnect(s, l.err)
}

/**
* detach: Deja el cliente sin conexión si l sigue siendo la actual.
* @param l *link
**/
func (s *Client) detach(l *link) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.link == l {
		s.link = nil
	}
}

/**
* close: Cierra la conexión; los Request en espera retornan MSG_TCP_CLOSED.
* @return error
**/
func (s *Client) close() error {
	s.mu.Lock()
	l := s.link
	s.link = nil
	s.mu.Unlock()

	if l == nil {
		return nil
	}
	return l.close()
}

/**
* current: Retorna la conexión actual.
* @return *link, error
**/
func (s *Client) current() (*link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.link == nil {
		return nil, errors.New(msg.MSG_TCP_NOT_CONNECTED)
	}
	return s.link, nil
}

/**
* setTimeout: Cambia el timeout de conexión, escritura y Request; aplica también a la
* conexión abierta.
* @param timeout time.Duration
* @return error
**/
func (s *Client) setTimeout(timeout time.Duration) error {
	return s.timeout.set(timeout)
}

/**
* send: Envía message sin esperar respuesta.
* @param message []byte
* @return error
**/
func (s *Client) send(message []byte) error {
	l, err := s.current()
	if err != nil {
		return err
	}
	return l.write(frame{kind: msgSend, data: message})
}

/**
* request: Envía message y espera su respuesta, hasta el timeout. Puede llamarse desde
* varias goroutines a la vez: cada solicitud lleva su propio ID.
* @param message []byte
* @return []byte, error
**/
func (s *Client) request(message []byte) ([]byte, error) {
	l, err := s.current()
	if err != nil {
		return nil, err
	}
	return l.request(s.nextID.Add(1), message)
}

/**
* sendStream: Abre un stream para enviar datos por partes sin esperar respuesta.
* @return *StreamWriter, error
**/
func (s *Client) sendStream() (*StreamWriter, error) {
	l, err := s.current()
	if err != nil {
		return nil, err
	}
	st, err := l.openStream(msgOpenSend, s.nextID.Add(1), nil, false)
	if err != nil {
		return nil, err
	}
	return &StreamWriter{st: st}, nil
}

/**
* requestStream: Abre un stream para enviar datos por partes; la respuesta se espera
* con CloseAndReceive.
* @return *StreamWriter, error
**/
func (s *Client) requestStream() (*StreamWriter, error) {
	l, err := s.current()
	if err != nil {
		return nil, err
	}
	id := s.nextID.Add(1)
	ch := l.addPending(id)
	st, err := l.openStream(msgOpenRequest, id, nil, false)
	if err != nil {
		l.removePending(id)
		return nil, err
	}
	st.response = ch
	return &StreamWriter{st: st}, nil
}

/**
* receiveStream: Envía message y recibe la respuesta por partes.
* @param message []byte
* @return *StreamReader, error
**/
func (s *Client) receiveStream(message []byte) (*StreamReader, error) {
	l, err := s.current()
	if err != nil {
		return nil, err
	}
	st, err := l.openStream(msgOpenReceive, s.nextID.Add(1), message, true)
	if err != nil {
		return nil, err
	}
	return &StreamReader{st: st}, nil
}

/**
* onSendStream: Define la función que recibe los SendStream (reemplaza a la anterior).
* @param fn func(conn *Client, in *StreamReader)
**/
func (s *Client) onSendStream(fn func(conn *Client, in *StreamReader)) {
	s.hs.setSendStream(fn)
}

/**
* onRequestStream: Define la función que atiende los RequestStream (reemplaza a la anterior).
* @param fn func(conn *Client, in *StreamReader) ([]byte, error)
**/
func (s *Client) onRequestStream(fn func(conn *Client, in *StreamReader) ([]byte, error)) {
	s.hs.setRequestStream(fn)
}

/**
* onReceiveStream: Define la función que atiende los ReceiveStream (reemplaza a la anterior).
* @param fn func(conn *Client, message []byte, out *StreamWriter) error
**/
func (s *Client) onReceiveStream(fn func(conn *Client, message []byte, out *StreamWriter) error) {
	s.hs.setReceiveStream(fn)
}

/**
* onConnect: Agrega fn a las funciones que se llaman cada vez que connectTo abre la conexión.
* @param fn func(conn *Client)
**/
func (s *Client) onConnect(fn func(conn *Client)) {
	s.hs.onConnect(fn)
}

/**
* onDisconnect: Agrega fn a las funciones que se llaman cuando la conexión se cierra.
* @param fn func(conn *Client, err error)
**/
func (s *Client) onDisconnect(fn func(conn *Client, err error)) {
	s.hs.onDisconnect(fn)
}

/**
* onSend: Agrega fn a las funciones que reciben los mensajes tipo send.
* @param fn func(conn *Client, message []byte)
**/
func (s *Client) onSend(fn func(conn *Client, message []byte)) {
	s.hs.onSend(fn)
}

/**
* onRequest: Agrega fn a las funciones que atienden los mensajes tipo request.
* @param fn func(conn *Client, message []byte) ([]byte, error)
**/
func (s *Client) onRequest(fn func(conn *Client, message []byte) ([]byte, error)) {
	s.hs.onRequest(fn)
}
