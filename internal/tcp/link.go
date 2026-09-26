package tcp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/msg"
)

type msgType byte

const (
	msgSend     msgType = 1 // solo envía, no espera respuesta
	msgRequest  msgType = 2 // espera una respuesta con el mismo ID
	msgResponse msgType = 3 // respuesta a un msgRequest
	msgError    msgType = 4 // respuesta a un msgRequest que falló; Data es el texto del error

	// Streams. El ID es el del lado que abrió el stream.
	msgOpenSend    msgType = 5  // abre un SendStream
	msgOpenRequest msgType = 6  // abre un RequestStream; la respuesta llega como msgResponse/msgError
	msgOpenReceive msgType = 7  // abre un ReceiveStream; Data es el mensaje
	msgData        msgType = 8  // un trozo del stream
	msgEnd         msgType = 9  // fin normal del stream
	msgFail        msgType = 10 // fin con error; Data es el texto del error
	msgAck         msgType = 11 // el lector consumió un trozo: un crédito más para el escritor
	msgCancel      msgType = 12 // el lector dejó de leer; Data es el motivo

	// toOpener: bit alto del byte de tipo. Marca las tramas de stream que van hacia el
	// lado que abrió el stream, así los IDs de los streams que abre cada lado no se cruzan.
	toOpenerBit byte = 0x80
)

const (
	headerSize     = 13       // Len(4) + Type(1) + ID(8)
	maxMessageSize = 64 << 20 // 64 MB por mensaje
	streamWindow   = 32       // trozos que un escritor puede enviar sin recibir ack
	defaultTimeout = 10 * time.Second
)

type frame struct {
	kind     msgType
	id       uint64
	data     []byte
	toOpener bool // solo streams: va hacia el lado que abrió el stream
}

/**
* duration: Timeout que se puede cambiar mientras hay conexiones en uso.
**/
type duration struct {
	ns atomic.Int64
}

/**
* newDuration: Crea un duration con el valor d.
* @param d time.Duration
* @return *duration
**/
func newDuration(d time.Duration) *duration {
	result := &duration{}
	result.set(d)
	return result
}

/**
* get: Retorna el valor actual.
* @return time.Duration
**/
func (s *duration) get() time.Duration {
	return time.Duration(s.ns.Load())
}

/**
* set: Cambia el valor; d debe ser mayor que cero.
* @param d time.Duration
* @return error
**/
func (s *duration) set(d time.Duration) error {
	if d <= 0 {
		return errors.New(msg.MSG_TCP_INVALID_TIMEOUT)
	}
	s.ns.Store(int64(d))
	return nil
}

/**
* handlers: Funciones que se llaman al abrir y cerrar una conexión, y las que reciben
* los mensajes tipo send y atienden los tipo request. Todas reciben conn, la conexión
* en la que ocurrió. Un cliente tiene las suyas; el servidor comparte las suyas con
* todas sus conexiones.
**/
type handlers struct {
	mu            sync.RWMutex                                         // protege las listas de funciones
	connectFns    []func(conn *Client)                                 // al abrir una conexión
	disconnectFns []func(conn *Client, err error)                      // al cerrarse una conexión
	sendFns       []func(conn *Client, message []byte)                 // mensajes tipo send
	requestFns    []func(conn *Client, message []byte) ([]byte, error) // mensajes tipo request

	// Un stream solo se puede leer una vez: una función por tipo de stream.
	sendStreamFn    func(conn *Client, in *StreamReader)
	requestStreamFn func(conn *Client, in *StreamReader) ([]byte, error)
	receiveStreamFn func(conn *Client, message []byte, out *StreamWriter) error
}

/**
* safeCall: Llama a fn registrando en el log un panic en lugar de propagarlo.
* @param fn func()
**/
func safeCall(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logs.Errorf(msg.MSG_TCP_HANDLER_PANIC, r)
		}
	}()
	fn()
}

/**
* onConnect: Agrega fn a las funciones que se llaman al abrir una conexión. Se llaman
* en el orden en que se agregaron.
* @param fn func(conn *Client)
**/
func (s *handlers) onConnect(fn func(conn *Client)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connectFns = append(s.connectFns, fn)
}

/**
* onDisconnect: Agrega fn a las funciones que se llaman cuando una conexión se cierra,
* la cierre quien la cierre. Se llaman en el orden en que se agregaron.
* @param fn func(conn *Client, err error)
**/
func (s *handlers) onDisconnect(fn func(conn *Client, err error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disconnectFns = append(s.disconnectFns, fn)
}

/**
* onSend: Agrega fn a las funciones que reciben los mensajes tipo send. Se llaman en
* el orden en que se agregaron.
* @param fn func(conn *Client, message []byte)
**/
func (s *handlers) onSend(fn func(conn *Client, message []byte)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendFns = append(s.sendFns, fn)
}

/**
* onRequest: Agrega fn a las funciones que atienden los mensajes tipo request. Se
* prueban en el orden en que se agregaron: responde la primera que retorne una
* respuesta distinta de nil o un error; retornar (nil, nil) cede el mensaje a la
* siguiente. Si ninguna responde, el remitente recibe MSG_TCP_NO_HANDLER.
* @param fn func(conn *Client, message []byte) ([]byte, error)
**/
func (s *handlers) onRequest(fn func(conn *Client, message []byte) ([]byte, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestFns = append(s.requestFns, fn)
}

/**
* handleConnect: Llama a cada función de onConnect con conn, que ya está leyendo
* mensajes, así que pueden usar Send y Request. Si una hace panic, se registra en el
* log y se sigue con las demás.
* @param conn *Client
**/
func (s *handlers) handleConnect(conn *Client) {
	s.mu.RLock()
	fns := s.connectFns
	s.mu.RUnlock()

	for _, fn := range fns {
		safeCall(func() { fn(conn) })
	}
}

/**
* handleDisconnect: Llama a cada función de onDisconnect con conn, ya cerrada, y el
* motivo del cierre. Si una hace panic, se registra en el log y se sigue con las demás.
* @param conn *Client, err error
**/
func (s *handlers) handleDisconnect(conn *Client, err error) {
	s.mu.RLock()
	fns := s.disconnectFns
	s.mu.RUnlock()

	for _, fn := range fns {
		safeCall(func() { fn(conn, err) })
	}
}

/**
* handleSend: Entrega un mensaje tipo send a cada función de onSend. Si una hace
* panic, se registra en el log y se sigue con las demás.
* @param conn *Client, message []byte
**/
func (s *handlers) handleSend(conn *Client, message []byte) {
	s.mu.RLock()
	fns := s.sendFns
	s.mu.RUnlock()

	for _, fn := range fns {
		safeCall(func() { fn(conn, message) })
	}
}

/**
* handleRequest: Atiende un mensaje tipo request con las funciones de onRequest y
* envía la respuesta con el mismo ID por l. Si una función hace panic, el remitente
* recibe MSG_TCP_HANDLER_PANIC como error.
* @param conn *Client, l *link, f frame
**/
func (s *handlers) handleRequest(conn *Client, l *link, f frame) {
	s.mu.RLock()
	fns := s.requestFns
	s.mu.RUnlock()

	for _, fn := range fns {
		result, err := callRequest(fn, conn, f.data)
		if err != nil {
			l.write(frame{kind: msgError, id: f.id, data: []byte(err.Error())})
			return
		}
		if result != nil {
			l.write(frame{kind: msgResponse, id: f.id, data: result})
			return
		}
	}

	l.write(frame{kind: msgError, id: f.id, data: []byte(msg.MSG_TCP_NO_HANDLER)})
}

/**
* callRequest: Llama a fn convirtiendo un panic en error.
* @param fn func(conn *Client, message []byte) ([]byte, error), conn *Client, message []byte
* @return []byte, error
**/
func callRequest(fn func(conn *Client, message []byte) ([]byte, error), conn *Client, message []byte) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = logs.Errorf(msg.MSG_TCP_HANDLER_PANIC, r)
			result = nil
		}
	}()
	return fn(conn, message)
}

/**
* link: Una conexión abierta, con sus solicitudes en espera de respuesta.
* Cada conexión tiene el suyo, así una reconexión no mezcla respuestas viejas.
**/
type link struct {
	conn      net.Conn
	timeout   *duration
	writeMu   sync.Mutex            // una trama a la vez en el socket
	pendingMu sync.Mutex            // protege pending
	pending   map[uint64]chan frame // solicitudes esperando respuesta, por ID
	streamsMu sync.Mutex            // protege mine y theirs
	mine      map[uint64]*stream    // streams abiertos por este lado, por ID
	theirs    map[uint64]*stream    // streams abiertos por el otro lado, por ID
	done      chan struct{}         // se cierra cuando termina readLoop
	err       error                 // motivo del cierre; se lee después de <-done
}

/**
* newLink: Envuelve conn; readLoop debe correr para que lleguen mensajes.
* @param conn net.Conn, timeout *duration
* @return *link
**/
func newLink(conn net.Conn, timeout *duration) *link {
	return &link{
		conn:    conn,
		timeout: timeout,
		pending: make(map[uint64]chan frame),
		mine:    make(map[uint64]*stream),
		theirs:  make(map[uint64]*stream),
		done:    make(chan struct{}),
	}
}

/**
* write: Escribe f completa en el socket; las escrituras concurrentes no se mezclan.
* @param f frame
* @return error
**/
func (l *link) write(f frame) error {
	if len(f.data) > maxMessageSize {
		return errors.New(msg.MSG_DATA_TOO_LARGE)
	}

	buf := make([]byte, headerSize+len(f.data))
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(f.data)))
	buf[4] = byte(f.kind)
	if f.toOpener {
		buf[4] |= toOpenerBit
	}
	binary.BigEndian.PutUint64(buf[5:13], f.id)
	copy(buf[headerSize:], f.data)

	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	l.conn.SetWriteDeadline(time.Now().Add(l.timeout.get()))
	_, err := l.conn.Write(buf)
	return err
}

/**
* addPending: Registra id para recibir su respuesta (msgResponse o msgError).
* @param id uint64
* @return chan frame
**/
func (l *link) addPending(id uint64) chan frame {
	ch := make(chan frame, 1)
	l.pendingMu.Lock()
	l.pending[id] = ch
	l.pendingMu.Unlock()
	return ch
}

/**
* removePending: Deja de esperar la respuesta de id.
* @param id uint64
**/
func (l *link) removePending(id uint64) {
	l.pendingMu.Lock()
	delete(l.pending, id)
	l.pendingMu.Unlock()
}

/**
* await: Espera la respuesta en ch, hasta el timeout o el cierre de la conexión.
* @param ch chan frame
* @return []byte, error
**/
func (l *link) await(ch chan frame) ([]byte, error) {
	timer := time.NewTimer(l.timeout.get())
	defer timer.Stop()

	select {
	case f := <-ch:
		if f.kind == msgError {
			return nil, errors.New(string(f.data))
		}
		return f.data, nil
	case <-l.done:
		return nil, l.err
	case <-timer.C:
		return nil, errors.New(msg.MSG_TCP_TIMEOUT)
	}
}

/**
* request: Envía message y espera la respuesta con el mismo id, hasta el timeout.
* @param id uint64, message []byte
* @return []byte, error
**/
func (l *link) request(id uint64, message []byte) ([]byte, error) {
	ch := l.addPending(id)
	defer l.removePending(id)

	if err := l.write(frame{kind: msgRequest, id: id, data: message}); err != nil {
		return nil, err
	}
	return l.await(ch)
}

/**
* readLoop: Lee tramas hasta que la conexión se cierra. Las respuestas se entregan al
* request que las espera; cada send y request entrante se atiende en su propia
* goroutine con hs, así una función lenta (o que a su vez llama a Request) no frena
* la lectura. conn es el Client dueño de l, el que reciben las funciones.
* @param conn *Client, hs *handlers
**/
func (l *link) readLoop(conn *Client, hs *handlers) {
	reader := bufio.NewReader(l.conn)
	for {
		f, err := readFrame(reader)
		if err != nil {
			l.finish(err)
			return
		}

		switch f.kind {
		case msgResponse, msgError:
			l.resolve(f)
		case msgSend:
			go hs.handleSend(conn, f.data)
		case msgRequest:
			go hs.handleRequest(conn, l, f)
		case msgOpenSend:
			go hs.handleSendStream(conn, l.acceptStream(f.id, true))
		case msgOpenRequest:
			go hs.handleRequestStream(conn, l, f.id, l.acceptStream(f.id, true))
		case msgOpenReceive:
			go hs.handleReceiveStream(conn, f.data, l.acceptStream(f.id, false))
		default:
			l.streamFrame(f)
		}
	}
}

/**
* resolve: Entrega una respuesta al request que la espera; si ya no espera (timeout), se
* descarta. Saca el ID de pending, así una respuesta repetida no bloquea la lectura.
* @param f frame
**/
func (l *link) resolve(f frame) {
	l.pendingMu.Lock()
	ch, ok := l.pending[f.id]
	delete(l.pending, f.id)
	l.pendingMu.Unlock()
	if ok {
		ch <- f
	}
}

/**
* close: Cierra la conexión y espera a que termine readLoop.
* @return error
**/
func (l *link) close() error {
	err := l.conn.Close()
	<-l.done
	return err
}

/**
* finish: Cierra la conexión y despierta a los request en espera con el motivo.
* @param err error
**/
func (l *link) finish(err error) {
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		l.err = errors.New(msg.MSG_TCP_CLOSED)
	} else {
		l.err = fmt.Errorf("%s: %w", msg.MSG_TCP_CLOSED, err)
	}
	l.conn.Close()
	close(l.done)
}

/**
* readFrame: Lee una trama [Len:4][Type:1][ID:8][Data]; el bit alto de Type es toOpener.
* @param r io.Reader
* @return frame, error
**/
func readFrame(r io.Reader) (frame, error) {
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return frame{}, err
	}

	size := binary.BigEndian.Uint32(header[0:4])
	if size > maxMessageSize {
		return frame{}, errors.New(msg.MSG_DATA_TOO_LARGE)
	}

	f := frame{
		kind:     msgType(header[4] &^ toOpenerBit),
		id:       binary.BigEndian.Uint64(header[5:13]),
		data:     make([]byte, size),
		toOpener: header[4]&toOpenerBit != 0,
	}
	if f.kind < msgSend || f.kind > msgCancel {
		return frame{}, errors.New(msg.MSG_TCP_INVALID_FRAME)
	}
	if _, err := io.ReadFull(r, f.data); err != nil {
		return frame{}, err
	}

	return f, nil
}
