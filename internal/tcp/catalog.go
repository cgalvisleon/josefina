package tcp

import "time"

// ---- Cliente: conexión ----

/**
* NewClient: Crea un cliente sin conectar.
* @return *Client, error
**/
func NewClient() (*Client, error) {
	return newClient()
}

/**
* ConnectTo: Abre la conexión con addr y empieza a leer mensajes en segundo plano.
* @param addr string
* @return error
**/
func (s *Client) ConnectTo(addr string) error {
	return s.connectTo(addr)
}

/**
* Close: Cierra la conexión; los Request en espera retornan MSG_TCP_CLOSED.
* @return error
**/
func (s *Client) Close() error {
	return s.close()
}

/**
* SetTimeout: Cambia el timeout de conexión, escritura y Request (por defecto 10s); aplica
* también a la conexión abierta.
* @param timeout time.Duration
* @return error
**/
func (s *Client) SetTimeout(timeout time.Duration) error {
	return s.setTimeout(timeout)
}

// ---- Cliente: envío ----

/**
* Send: Envía message sin esperar respuesta.
* @param message []byte
* @return error
**/
func (s *Client) Send(message []byte) error {
	return s.send(message)
}

/**
* Request: Envía message y espera su respuesta, hasta el timeout. Admite llamadas concurrentes.
* @param message []byte
* @return []byte, error
**/
func (s *Client) Request(message []byte) ([]byte, error) {
	return s.request(message)
}

// ---- Cliente: recepción ----

/**
* OnConnect: Agrega fn a las funciones que se llaman cada vez que ConnectTo abre la
* conexión, antes de que ConnectTo retorne.
* @param fn func(conn *Client)
**/
func (s *Client) OnConnect(fn func(conn *Client)) {
	s.onConnect(fn)
}

/**
* OnDisconnect: Agrega fn a las funciones que se llaman cuando la conexión se cierra, la
* cierre quien la cierre; err es el motivo.
* @param fn func(conn *Client, err error)
**/
func (s *Client) OnDisconnect(fn func(conn *Client, err error)) {
	s.onDisconnect(fn)
}

/**
* OnSend: Agrega fn a las funciones que reciben los mensajes tipo send, en orden; conn
* es la conexión por la que llegó.
* @param fn func(conn *Client, message []byte)
**/
func (s *Client) OnSend(fn func(conn *Client, message []byte)) {
	s.onSend(fn)
}

/**
* OnRequest: Agrega fn a las funciones que atienden los mensajes tipo request. Responde
* la primera que retorne una respuesta distinta de nil o un error; (nil, nil) cede a la siguiente.
* @param fn func(conn *Client, message []byte) ([]byte, error)
**/
func (s *Client) OnRequest(fn func(conn *Client, message []byte) ([]byte, error)) {
	s.onRequest(fn)
}

// ---- Cliente: streams ----

/**
* SendStream: Abre un stream para enviar datos por partes, sin esperar respuesta. Se
* escribe con Write y se termina con Close; el otro lado lo recibe en OnSendStream.
* @return *StreamWriter, error
**/
func (s *Client) SendStream() (*StreamWriter, error) {
	return s.sendStream()
}

/**
* RequestStream: Abre un stream para enviar datos por partes y esperar una respuesta.
* Se escribe con Write y se termina con CloseAndReceive, que retorna la respuesta; el
* otro lado lo atiende en OnRequestStream.
* @return *StreamWriter, error
**/
func (s *Client) RequestStream() (*StreamWriter, error) {
	return s.requestStream()
}

/**
* ReceiveStream: Envía message y recibe la respuesta por partes con Read, hasta io.EOF;
* el otro lado la escribe en OnReceiveStream.
* @param message []byte
* @return *StreamReader, error
**/
func (s *Client) ReceiveStream(message []byte) (*StreamReader, error) {
	return s.receiveStream(message)
}

/**
* OnSendStream: Define la función que recibe los SendStream; reemplaza a la anterior
* porque un stream solo se puede leer una vez. Al retornar, lo que no leyó se cancela.
* @param fn func(conn *Client, in *StreamReader)
**/
func (s *Client) OnSendStream(fn func(conn *Client, in *StreamReader)) {
	s.onSendStream(fn)
}

/**
* OnRequestStream: Define la función que atiende los RequestStream; lo que retorna es
* la respuesta. Reemplaza a la anterior.
* @param fn func(conn *Client, in *StreamReader) ([]byte, error)
**/
func (s *Client) OnRequestStream(fn func(conn *Client, in *StreamReader) ([]byte, error)) {
	s.onRequestStream(fn)
}

/**
* OnReceiveStream: Define la función que atiende los ReceiveStream: escribe la respuesta
* por partes en out. Si retorna nil el stream termina bien; si retorna un error, el
* remitente lo recibe al leer. Reemplaza a la anterior.
* @param fn func(conn *Client, message []byte, out *StreamWriter) error
**/
func (s *Client) OnReceiveStream(fn func(conn *Client, message []byte, out *StreamWriter) error) {
	s.onReceiveStream(fn)
}

// ---- Servidor ----

/**
* NewServer: Crea un servidor que todavía no escucha.
* @return *Server, error
**/
func NewServer() (*Server, error) {
	return newServer()
}

/**
* Listen: Empieza a escuchar en addr y acepta conexiones en segundo plano.
* @param addr string
* @return error
**/
func (s *Server) Listen(addr string) error {
	return s.listen(addr)
}

/**
* Addr: Retorna la dirección en la que escucha, o "" si no está escuchando.
* @return string
**/
func (s *Server) Addr() string {
	return s.addr()
}

/**
* Close: Deja de escuchar, cierra todas las conexiones y espera a que terminen.
* @return error
**/
func (s *Server) Close() error {
	return s.close()
}

/**
* Count: Retorna el número de conexiones abiertas.
* @return int
**/
func (s *Server) Count() int {
	return s.count()
}

/**
* SetTimeout: Cambia el timeout de escritura y Request (por defecto 10s); aplica también
* a las conexiones abiertas.
* @param timeout time.Duration
* @return error
**/
func (s *Server) SetTimeout(timeout time.Duration) error {
	return s.setTimeout(timeout)
}

/**
* OnConnect: Agrega fn a las funciones que se llaman con cada conexión aceptada. Con
* conn el servidor envía a ese cliente (conn.Send, conn.Request) mientras sigue recibiendo.
* @param fn func(conn *Client)
**/
func (s *Server) OnConnect(fn func(conn *Client)) {
	s.onConnect(fn)
}

/**
* OnDisconnect: Agrega fn a las funciones que se llaman cuando se cierra cualquier
* conexión, la cierre el cliente o el servidor; err es el motivo.
* @param fn func(conn *Client, err error)
**/
func (s *Server) OnDisconnect(fn func(conn *Client, err error)) {
	s.onDisconnect(fn)
}

/**
* OnSend: Agrega fn a las funciones que reciben los mensajes tipo send de cualquier
* conexión; conn es la conexión por la que llegó, para responderle o identificarla.
* @param fn func(conn *Client, message []byte)
**/
func (s *Server) OnSend(fn func(conn *Client, message []byte)) {
	s.onSend(fn)
}

/**
* OnRequest: Agrega fn a las funciones que atienden los mensajes tipo request de
* cualquier conexión, con la misma regla que Client.OnRequest.
* @param fn func(conn *Client, message []byte) ([]byte, error)
**/
func (s *Server) OnRequest(fn func(conn *Client, message []byte) ([]byte, error)) {
	s.onRequest(fn)
}

/**
* OnSendStream: Define la función que recibe los SendStream de cualquier conexión.
* @param fn func(conn *Client, in *StreamReader)
**/
func (s *Server) OnSendStream(fn func(conn *Client, in *StreamReader)) {
	s.onSendStream(fn)
}

/**
* OnRequestStream: Define la función que atiende los RequestStream de cualquier conexión.
* @param fn func(conn *Client, in *StreamReader) ([]byte, error)
**/
func (s *Server) OnRequestStream(fn func(conn *Client, in *StreamReader) ([]byte, error)) {
	s.onRequestStream(fn)
}

/**
* OnReceiveStream: Define la función que atiende los ReceiveStream de cualquier conexión.
* @param fn func(conn *Client, message []byte, out *StreamWriter) error
**/
func (s *Server) OnReceiveStream(fn func(conn *Client, message []byte, out *StreamWriter) error) {
	s.onReceiveStream(fn)
}

// ---- StreamWriter ----

/**
* Write: Envía un trozo. Si el lector tiene muchos trozos sin leer, espera a que lea
* (hasta el timeout). Si el lector dejó de leer, retorna su motivo.
* @param data []byte
* @return error
**/
func (s *StreamWriter) Write(data []byte) error {
	return s.write(data)
}

/**
* Close: Termina el stream con normalidad; el lector recibe io.EOF.
* @return error
**/
func (s *StreamWriter) Close() error {
	return s.close()
}

/**
* CloseWithError: Termina el stream con un error; el lector lo recibe en lugar de io.EOF.
* @param reason error
* @return error
**/
func (s *StreamWriter) CloseWithError(reason error) error {
	return s.closeWithError(reason)
}

/**
* CloseAndReceive: Solo para RequestStream: termina el stream y espera la respuesta.
* @return []byte, error
**/
func (s *StreamWriter) CloseAndReceive() ([]byte, error) {
	return s.closeAndReceive()
}

// ---- StreamReader ----

/**
* Read: Retorna el siguiente trozo, en orden; io.EOF al terminar el stream, o el error
* con que lo cerró el escritor. Espera cada trozo hasta el timeout.
* @return []byte, error
**/
func (s *StreamReader) Read() ([]byte, error) {
	return s.read()
}

/**
* Close: Deja de leer antes del final; el escritor recibe MSG_TCP_STREAM_CANCELED.
**/
func (s *StreamReader) Close() {
	s.close()
}
