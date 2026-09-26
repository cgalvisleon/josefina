package tcp

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/msg"
)

/**
* stream: Un stream en uno de sus dos extremos. En cada extremo tiene un solo papel:
* lector (recibe msgData/msgEnd/msgFail y envía msgAck/msgCancel) o escritor (envía
* msgData/msgEnd/msgFail y recibe msgAck/msgCancel).
**/
type stream struct {
	l    *link
	id   uint64
	mine bool // lo abrió este lado

	// lector
	in      chan []byte   // trozos recibidos sin leer; nunca más de streamWindow
	ended   chan struct{} // se cierra al terminar el stream
	endErr  error         // io.EOF o el motivo; se lee después de <-ended
	endOnce sync.Once

	// escritor
	credits    chan struct{} // un token por trozo que se puede enviar sin esperar ack
	canceled   chan struct{} // se cierra cuando el lector deja de leer
	cancelErr  error         // motivo; se lee después de <-canceled
	cancelOnce sync.Once
	closeOnce  sync.Once
	closed     atomic.Bool // ya se envió el fin
	closeErr   error
	response   chan frame // solo RequestStream: la respuesta final
}

type StreamWriter struct {
	st *stream
}

type StreamReader struct {
	st *stream
}

/**
* newStream: Crea un extremo de stream como lector o como escritor.
* @param l *link, id uint64, mine bool, reader bool
* @return *stream
**/
func newStream(l *link, id uint64, mine, reader bool) *stream {
	st := &stream{l: l, id: id, mine: mine}
	if reader {
		st.in = make(chan []byte, streamWindow)
		st.ended = make(chan struct{})
	} else {
		st.credits = make(chan struct{}, streamWindow)
		for i := 0; i < streamWindow; i++ {
			st.credits <- struct{}{}
		}
		st.canceled = make(chan struct{})
	}
	return st
}

/**
* frame: Arma una trama de este stream, con la dirección que corresponde.
* @param kind msgType, data []byte
* @return frame
**/
func (st *stream) frame(kind msgType, data []byte) frame {
	return frame{kind: kind, id: st.id, data: data, toOpener: !st.mine}
}

/**
* streams: Retorna el mapa donde vive un stream según quién lo abrió.
* @param mine bool
* @return map[uint64]*stream
**/
func (l *link) streams(mine bool) map[uint64]*stream {
	if mine {
		return l.mine
	}
	return l.theirs
}

/**
* openStream: Abre un stream de este lado y envía la trama de apertura.
* @param kind msgType, id uint64, message []byte, reader bool
* @return *stream, error
**/
func (l *link) openStream(kind msgType, id uint64, message []byte, reader bool) (*stream, error) {
	st := newStream(l, id, true, reader)
	l.streamsMu.Lock()
	l.mine[id] = st
	l.streamsMu.Unlock()

	if err := l.write(frame{kind: kind, id: id, data: message}); err != nil {
		l.dropStream(st)
		return nil, err
	}
	return st, nil
}

/**
* acceptStream: Registra un stream que abrió el otro lado.
* @param id uint64, reader bool
* @return *stream
**/
func (l *link) acceptStream(id uint64, reader bool) *stream {
	st := newStream(l, id, false, reader)
	l.streamsMu.Lock()
	l.theirs[id] = st
	l.streamsMu.Unlock()
	return st
}

/**
* dropStream: Olvida st; las tramas que lleguen después para él se descartan.
* @param st *stream
**/
func (l *link) dropStream(st *stream) {
	l.streamsMu.Lock()
	defer l.streamsMu.Unlock()
	streams := l.streams(st.mine)
	if streams[st.id] == st {
		delete(streams, st.id)
	}
}

/**
* streamFrame: Entrega una trama de datos o de control a su stream. Si el stream ya no
* existe y llegan datos, le avisa al escritor que deje de enviar.
* @param f frame
**/
func (l *link) streamFrame(f frame) {
	l.streamsMu.Lock()
	st := l.streams(f.toOpener)[f.id]
	l.streamsMu.Unlock()

	if st == nil {
		if f.kind == msgData {
			l.write(frame{kind: msgCancel, id: f.id, data: []byte(msg.MSG_TCP_STREAM_CLOSED), toOpener: !f.toOpener})
		}
		return
	}

	switch f.kind {
	case msgData:
		if st.in == nil {
			return
		}
		select {
		case st.in <- f.data:
		default:
			// El escritor no respetó los créditos.
			st.cancel(errors.New(msg.MSG_TCP_INVALID_FRAME))
		}
	case msgEnd:
		st.finish(io.EOF)
	case msgFail:
		st.finish(errors.New(string(f.data)))
	case msgAck:
		if st.credits != nil {
			select {
			case st.credits <- struct{}{}:
			default:
			}
		}
	case msgCancel:
		st.canceledBy(errors.New(string(f.data)))
	}
}

// ---- Lector ----

/**
* finish: Marca el fin del stream (lector); los trozos ya recibidos se pueden seguir leyendo.
* @param err error
**/
func (st *stream) finish(err error) {
	if st.ended == nil {
		return
	}
	st.endOnce.Do(func() {
		st.endErr = err
		close(st.ended)
		st.l.dropStream(st)
	})
}

/**
* read: Retorna el siguiente trozo, en orden. Al terminar el stream retorna io.EOF, o el
* error con que lo cerró el escritor. Espera cada trozo hasta el timeout.
* @return []byte, error
**/
func (st *stream) read() ([]byte, error) {
	select {
	case data := <-st.in:
		return st.ack(data)
	default:
	}

	timer := time.NewTimer(st.l.timeout.get())
	defer timer.Stop()

	select {
	case data := <-st.in:
		return st.ack(data)
	case <-st.ended:
		select {
		case data := <-st.in:
			return st.ack(data)
		default:
			return nil, st.endErr
		}
	case <-st.l.done:
		return nil, st.l.err
	case <-timer.C:
		return nil, errors.New(msg.MSG_TCP_TIMEOUT)
	}
}

/**
* ack: Devuelve un crédito al escritor por el trozo consumido.
* @param data []byte
* @return []byte, error
**/
func (st *stream) ack(data []byte) ([]byte, error) {
	st.l.write(st.frame(msgAck, nil))
	return data, nil
}

/**
* cancel: Deja de leer: le avisa al escritor con reason y termina el stream de este lado.
* No hace nada si el stream ya terminó.
* @param reason error
**/
func (st *stream) cancel(reason error) {
	select {
	case <-st.ended:
		return
	default:
	}
	st.l.write(st.frame(msgCancel, []byte(reason.Error())))
	st.finish(errors.New(msg.MSG_TCP_STREAM_CLOSED))
}

// ---- Escritor ----

/**
* canceledBy: El lector dejó de leer; los Write siguientes retornan err.
* @param err error
**/
func (st *stream) canceledBy(err error) {
	if st.canceled == nil {
		return
	}
	st.cancelOnce.Do(func() {
		st.cancelErr = err
		close(st.canceled)
		st.l.dropStream(st)
	})
}

/**
* write: Envía un trozo. Si el lector tiene streamWindow trozos sin leer, espera a que
* lea (hasta el timeout).
* @param data []byte
* @return error
**/
func (st *stream) write(data []byte) error {
	select {
	case <-st.canceled:
		return st.cancelErr
	default:
	}
	if st.closed.Load() {
		return errors.New(msg.MSG_TCP_STREAM_CLOSED)
	}

	timer := time.NewTimer(st.l.timeout.get())
	defer timer.Stop()

	select {
	case <-st.credits:
	case <-st.canceled:
		return st.cancelErr
	case <-st.l.done:
		return st.l.err
	case <-timer.C:
		return errors.New(msg.MSG_TCP_TIMEOUT)
	}
	return st.l.write(st.frame(msgData, data))
}

/**
* end: Envía el fin del stream (msgEnd, o msgFail si reason no es nil). Solo la primera
* llamada tiene efecto.
* @param reason error
* @return error
**/
func (st *stream) end(reason error) error {
	st.closeOnce.Do(func() {
		st.closed.Store(true)
		if reason == nil {
			st.closeErr = st.l.write(st.frame(msgEnd, nil))
		} else {
			st.closeErr = st.l.write(st.frame(msgFail, []byte(reason.Error())))
		}
		if st.response == nil {
			st.l.dropStream(st)
		}
	})
	return st.closeErr
}

// ---- StreamWriter ----

/**
* write: Envía un trozo del stream.
* @param data []byte
* @return error
**/
func (s *StreamWriter) write(data []byte) error {
	return s.st.write(data)
}

/**
* close: Termina el stream con normalidad; el lector recibe io.EOF.
* @return error
**/
func (s *StreamWriter) close() error {
	return s.st.end(nil)
}

/**
* closeWithError: Termina el stream con un error; el lector lo recibe en lugar de io.EOF.
* @param reason error
* @return error
**/
func (s *StreamWriter) closeWithError(reason error) error {
	if reason == nil {
		reason = errors.New(msg.MSG_TCP_STREAM_CLOSED)
	}
	return s.st.end(reason)
}

/**
* closeAndReceive: Solo para RequestStream: termina el stream y espera la respuesta.
* @return []byte, error
**/
func (s *StreamWriter) closeAndReceive() ([]byte, error) {
	st := s.st
	if st.response == nil {
		return nil, errors.New(msg.MSG_TCP_NOT_REQUEST_STREAM)
	}
	defer st.l.removePending(st.id)
	defer st.l.dropStream(st)

	if err := st.end(nil); err != nil {
		return nil, err
	}
	return st.l.await(st.response)
}

// ---- StreamReader ----

/**
* read: Retorna el siguiente trozo; io.EOF al terminar el stream.
* @return []byte, error
**/
func (s *StreamReader) read() ([]byte, error) {
	return s.st.read()
}

/**
* close: Deja de leer antes del final; el escritor recibe MSG_TCP_STREAM_CANCELED.
**/
func (s *StreamReader) close() {
	s.st.cancel(errors.New(msg.MSG_TCP_STREAM_CANCELED))
}

// ---- Handlers de streams ----

/**
* setSendStream: Define la función que recibe los SendStream (reemplaza a la anterior).
* @param fn func(conn *Client, in *StreamReader)
**/
func (s *handlers) setSendStream(fn func(conn *Client, in *StreamReader)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendStreamFn = fn
}

/**
* setRequestStream: Define la función que atiende los RequestStream (reemplaza a la anterior).
* @param fn func(conn *Client, in *StreamReader) ([]byte, error)
**/
func (s *handlers) setRequestStream(fn func(conn *Client, in *StreamReader) ([]byte, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestStreamFn = fn
}

/**
* setReceiveStream: Define la función que atiende los ReceiveStream (reemplaza a la anterior).
* @param fn func(conn *Client, message []byte, out *StreamWriter) error
**/
func (s *handlers) setReceiveStream(fn func(conn *Client, message []byte, out *StreamWriter) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receiveStreamFn = fn
}

/**
* handleSendStream: Entrega un SendStream entrante a su función. Al retornar la función,
* si no leyó todo, se cancela el resto.
* @param conn *Client, st *stream
**/
func (s *handlers) handleSendStream(conn *Client, st *stream) {
	s.mu.RLock()
	fn := s.sendStreamFn
	s.mu.RUnlock()

	in := &StreamReader{st: st}
	if fn == nil {
		st.cancel(errors.New(msg.MSG_TCP_NO_HANDLER))
		return
	}
	safeCall(func() { fn(conn, in) })
	in.close()
}

/**
* handleRequestStream: Atiende un RequestStream entrante y envía la respuesta con el
* mismo ID. Un panic se convierte en error para el remitente.
* @param conn *Client, l *link, id uint64, st *stream
**/
func (s *handlers) handleRequestStream(conn *Client, l *link, id uint64, st *stream) {
	s.mu.RLock()
	fn := s.requestStreamFn
	s.mu.RUnlock()

	in := &StreamReader{st: st}
	if fn == nil {
		l.write(frame{kind: msgError, id: id, data: []byte(msg.MSG_TCP_NO_HANDLER)})
		st.cancel(errors.New(msg.MSG_TCP_NO_HANDLER))
		return
	}

	result, err := func() (result []byte, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = logs.Errorf(msg.MSG_TCP_HANDLER_PANIC, r)
			}
		}()
		return fn(conn, in)
	}()
	if err != nil {
		l.write(frame{kind: msgError, id: id, data: []byte(err.Error())})
	} else {
		l.write(frame{kind: msgResponse, id: id, data: result})
	}
	in.close()
}

/**
* handleReceiveStream: Atiende un ReceiveStream entrante: la función escribe la
* respuesta por partes en out. Si retorna nil el stream termina con normalidad; si
* retorna un error (o hace panic), el remitente lo recibe al leer.
* @param conn *Client, message []byte, st *stream
**/
func (s *handlers) handleReceiveStream(conn *Client, message []byte, st *stream) {
	s.mu.RLock()
	fn := s.receiveStreamFn
	s.mu.RUnlock()

	out := &StreamWriter{st: st}
	if fn == nil {
		out.closeWithError(errors.New(msg.MSG_TCP_NO_HANDLER))
		return
	}

	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = logs.Errorf(msg.MSG_TCP_HANDLER_PANIC, r)
			}
		}()
		return fn(conn, message, out)
	}()
	if err != nil {
		out.closeWithError(err)
	} else {
		out.close()
	}
}
