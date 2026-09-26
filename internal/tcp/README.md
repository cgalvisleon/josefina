# `internal/tcp`

Cliente y servidor TCP para enviar y recibir mensajes de manera concurrente sobre una sola conexión.

Sigue la misma organización que `internal/store`: la implementación es privada y `catalog.go` tiene solo los métodos públicos, que llaman a su versión privada del mismo nombre en minúscula.

| Archivo | Contenido |
|---|---|
| `catalog.go` | Métodos públicos del cliente y del servidor. |
| `link.go` | Lo compartido: formato de los mensajes, la conexión (`link`), las funciones de recepción (`handlers`) y el timeout (`duration`). |
| `client.go` | El cliente. |
| `stream.go` | Streams: `StreamWriter`, `StreamReader`, control de flujo y sus funciones de recepción. |
| `server.go` | El servidor; atiende cada conexión aceptada con el mismo código del cliente. |

## Uso rápido

```go
// Servidor
srv, err := tcp.NewServer()
srv.OnConnect(func(conn *tcp.Client) {
	// conn es la conexión con ese cliente: el servidor también le puede enviar.
	conn.Send([]byte("bienvenido"))
})
srv.OnRequest(func(conn *tcp.Client, message []byte) ([]byte, error) {
	return append([]byte("echo:"), message...), nil
})
srv.OnSend(func(conn *tcp.Client, message []byte) {
	conn.Send([]byte("recibido")) // responde solo a quien lo envió
})
srv.OnDisconnect(func(conn *tcp.Client, err error) {
	fmt.Println("se cerró una conexión:", err)
})
if err := srv.Listen(":4377"); err != nil {
	return err
}
defer srv.Close()

// Cliente
c, err := tcp.NewClient()
if err := c.ConnectTo("127.0.0.1:4377"); err != nil {
	return err
}
defer c.Close()

err = c.Send([]byte("hola"))                // no espera respuesta
response, err := c.Request([]byte("ping")) // espera la respuesta: "echo:ping"
```

## Métodos públicos

### Cliente: conexión

- **`NewClient() (*Client, error)`** — Crea un cliente sin conectar.
- **`ConnectTo(addr string) error`** — Abre la conexión y empieza a leer mensajes en segundo plano. Si ya está conectado, retorna `MSG_TCP_ALREADY_CONNECTED`.
- **`Close() error`** — Cierra la conexión. Los `Request` en espera terminan con `MSG_TCP_CLOSED`.
- **`SetTimeout(timeout time.Duration) error`** — Cambia el timeout de conexión, escritura y `Request` (por defecto 10 segundos). Aplica también a la conexión abierta. Debe ser mayor que cero.

### Cliente: envío

- **`Send(message []byte) error`** — Envía y no espera respuesta.
- **`Request(message []byte) ([]byte, error)`** — Envía y espera la respuesta, hasta el timeout.

### Cliente: recepción

- **`OnConnect(fn func(conn *Client))`** — Agrega una función que se llama cada vez que `ConnectTo` abre la conexión, antes de que `ConnectTo` retorne. `conn` es el mismo cliente, ya leyendo mensajes, así que la función puede usar `Send` y `Request`.
- **`OnDisconnect(fn func(conn *Client, err error))`** — Agrega una función que se llama cuando la conexión se cierra, la cierre quien la cierre; `err` es el motivo. Como el cliente ya quedó sin conexión, la función puede llamar a `conn.ConnectTo` para reconectar.
- **`OnSend(fn func(conn *Client, message []byte))`** — Agrega una función que recibe los mensajes tipo send.
- **`OnRequest(fn func(conn *Client, message []byte) ([]byte, error))`** — Agrega una función que atiende los mensajes tipo request.

En todas, `conn` es la conexión en la que ocurrió: en el cliente, el mismo cliente.

### Cliente: streams

Para enviar o recibir datos por partes (progresivamente), sin juntarlos antes en un solo mensaje.

| Método | Qué hace | Lo atiende en el otro lado |
|---|---|---|
| **`SendStream() (*StreamWriter, error)`** | Envía por partes y no espera respuesta. | `OnSendStream` |
| **`RequestStream() (*StreamWriter, error)`** | Envía por partes y espera una respuesta con `CloseAndReceive`. | `OnRequestStream` |
| **`ReceiveStream(message []byte) (*StreamReader, error)`** | Envía un mensaje y recibe la respuesta por partes. | `OnReceiveStream` |

- **`OnSendStream(fn func(conn *Client, in *StreamReader))`** — Recibe los `SendStream`. Al retornar `fn`, lo que no leyó se cancela.
- **`OnRequestStream(fn func(conn *Client, in *StreamReader) ([]byte, error))`** — Atiende los `RequestStream`; lo que retorna es la respuesta (o el error) que recibe `CloseAndReceive`.
- **`OnReceiveStream(fn func(conn *Client, message []byte, out *StreamWriter) error)`** — Atiende los `ReceiveStream`: escribe la respuesta por partes en `out`. Si retorna `nil`, el stream termina con normalidad; si retorna un error, el que lee lo recibe en lugar de `io.EOF`. No hace falta cerrar `out`.

A diferencia de `OnSend`/`OnRequest`, cada `On…Stream` **define una sola función** (la reemplaza si se llama de nuevo), porque un stream solo se puede leer una vez. Si no hay función, el otro lado recibe `MSG_TCP_NO_HANDLER`.

**`StreamWriter`**

- **`Write(data []byte) error`** — Envía un trozo. Si el lector tiene 32 trozos sin leer, espera a que lea (hasta el timeout). Si el lector dejó de leer, retorna su motivo.
- **`Close() error`** — Termina el stream; el lector recibe `io.EOF`.
- **`CloseWithError(reason error) error`** — Termina el stream con un error; el lector lo recibe en lugar de `io.EOF`.
- **`CloseAndReceive() ([]byte, error)`** — Solo para `RequestStream`: termina el stream y espera la respuesta. En los demás retorna `MSG_TCP_NOT_REQUEST_STREAM` Un `RequestStream` siempre debe terminar con `CloseAndReceive`: si no, queda registrado hasta que se cierre la conexión.

**`StreamReader`**

- **`Read() ([]byte, error)`** — El siguiente trozo, en orden. Al terminar retorna `io.EOF`, o el error con que lo cerró el escritor. Espera cada trozo hasta el timeout.
- **`Close()`** — Deja de leer antes del final; el escritor recibe `MSG_TCP_STREAM_CANCELED` en su próximo `Write`.

```go
// Cliente: sube un archivo por partes y espera un resumen.
w, err := c.RequestStream()
for _, chunk := range chunks {
	if err := w.Write(chunk); err != nil {
		return err
	}
}
summary, err := w.CloseAndReceive()

// Servidor: lo recibe por partes.
srv.OnRequestStream(func(conn *tcp.Client, in *tcp.StreamReader) ([]byte, error) {
	total := 0
	for {
		chunk, err := in.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		total += len(chunk)
	}
	return []byte(fmt.Sprint(total)), nil
})

// Cliente: pide datos y los recibe por partes.
r, err := c.ReceiveStream([]byte("clientes"))
for {
	chunk, err := r.Read()
	if err == io.EOF {
		break
	}
	...
}

// Servidor: los genera por partes.
srv.OnReceiveStream(func(conn *tcp.Client, message []byte, out *tcp.StreamWriter) error {
	for _, row := range rows {
		if err := out.Write(row); err != nil {
			return err // el cliente dejó de leer o se cayó la conexión
		}
	}
	return nil
})
```

### Servidor

- **`NewServer() (*Server, error)`** — Crea un servidor que todavía no escucha.
- **`Listen(addr string) error`** — Empieza a escuchar y acepta conexiones en segundo plano. Con `":0"` el sistema elige el puerto. Si ya escucha, retorna `MSG_TCP_ALREADY_LISTENING`.
- **`Addr() string`** — Dirección en la que escucha, o `""` si no escucha.
- **`Close() error`** — Deja de escuchar, cierra todas las conexiones y espera a que terminen. Después se puede volver a llamar a `Listen`.
- **`Count() int`** — Número de conexiones abiertas.
- **`SetTimeout(timeout time.Duration) error`** — Igual que en el cliente; aplica a todas las conexiones, también a las abiertas.
- **`OnConnect(fn func(conn *Client))`** — Agrega una función que se llama con cada conexión aceptada, en su propia goroutine. `conn` es la conexión con ese cliente: el servidor le envía con `conn.Send` y `conn.Request`, al mismo tiempo que recibe, y puede guardarla para usarla después o cerrarla con `conn.Close`.
- **`OnDisconnect(fn func(conn *Client, err error))`** — Agrega una función que se llama cuando se cierra cualquier conexión, la cierre el cliente o el servidor (también con `Close` del servidor, que espera a que terminen). Sirve para olvidar lo que se guardó de esa conexión.
- **`OnSendStream(fn)`, `OnRequestStream(fn)`, `OnReceiveStream(fn)`** — Igual que en el cliente, para los streams que abra cualquier conexión. El servidor también puede abrir streams hacia un cliente con `conn.SendStream()`, `conn.RequestStream()` y `conn.ReceiveStream(...)`.
- **`OnSend(fn)`, `OnRequest(fn)`** — Igual que en el cliente, compartidas por todas las conexiones. `conn` dice de qué conexión llegó el mensaje: sirve para identificar al cliente (por ejemplo, como llave de un mapa) o para enviarle algo solo a él con `conn.Send`.

Las conexiones que entrega `OnConnect` del servidor comparten las funciones y el timeout del servidor: llamar a `OnSend`, `OnRequest`, `OnConnect`, `OnDisconnect` o `SetTimeout` sobre una de ellas cambia los del servidor.

## Cómo funciona

- **Mensajes:** cada uno viaja como `[Len:4][Tipo:1][ID:8][Datos]` (big-endian), con un máximo de 64 MB. Los tipos son `send`, `request`, `response` y `error`. El ID ata cada respuesta a su `Request`, así que se pueden hacer muchos `Send` y `Request` a la vez desde distintas goroutines por la misma conexión.
- **Streams:** usan los mismos mensajes con tipos propios (abrir, trozo, fin, fin con error, ack y cancelar). El bit alto del tipo indica si el mensaje va hacia el lado que abrió el stream, así los streams que abre cada lado no se confunden aunque tengan el mismo ID. Se pueden tener muchos streams abiertos a la vez, en los dos sentidos, junto con `Send` y `Request` normales.
- **Control de flujo:** quien escribe puede tener a lo sumo 32 trozos sin confirmar; cada `Read` del otro lado devuelve un crédito. Así un lector lento solo frena a su escritor: no llena la memoria ni bloquea la lectura de la conexión, y los demás mensajes siguen pasando.
- **Envío:** los envíos se escriben de uno en uno para que no se mezclen en el socket. Hay una goroutine por conexión que lee todo lo que llega.
- **Recepción:** cada mensaje que llega se atiende en su propia goroutine. Así, una función lenta no frena la lectura, y una función puede llamar a `Request` sin bloquearse.
- **Concurrencia en los dos sentidos:** cliente y servidor pueden usar `Send` y `Request` al mismo tiempo, desde muchas goroutines, mientras reciben mensajes del otro lado por la misma conexión.
- **Funciones de `OnConnect` y `OnDisconnect`:** se llaman en el orden en que se agregaron. `OnDisconnect` se llama una sola vez por conexión, desde la goroutine que la leía, después de que los `Request` en espera ya terminaron con `MSG_TCP_CLOSED`. `Close` del cliente no espera a que termine; `Close` del servidor sí, así que no se debe llamar a `Close` del servidor desde su propio `OnDisconnect` (se bloquearía).
- **Funciones de `OnSend`:** todas reciben cada mensaje, en el orden en que se agregaron.
- **Funciones de `OnRequest`:** se prueban en orden y responde la primera que devuelva una respuesta o un error. Devolver `(nil, nil)` pasa el mensaje a la siguiente. Si un handler devuelve un error, el que envió el `Request` recibe ese mismo texto de error. Si ninguna responde, recibe el error `MSG_TCP_NO_HANDLER`.
- **Panic en una función:** no cae el proceso. En `OnConnect`, `OnDisconnect` y `OnSend` se registra en el log y se sigue con las demás funciones; en `OnRequest` se registra y el que envió el `Request` recibe `MSG_TCP_HANDLER_PANIC` como error.
- **Cierre:** si el otro lado cierra la conexión o se llama a `Close`, los `Request` que estaban esperando terminan con `MSG_TCP_CLOSED` y el cliente vuelve a quedar sin conexión, así que se puede llamar a `ConnectTo` de nuevo.
- **Timeout:** una respuesta que llega después del timeout se descarta.

## Errores

Los mensajes están en `internal/msg` (inglés o español según `LANG`): `MSG_TCP_NOT_CONNECTED`, `MSG_TCP_ALREADY_CONNECTED`, `MSG_TCP_ALREADY_LISTENING`, `MSG_TCP_CLOSED`, `MSG_TCP_TIMEOUT`, `MSG_TCP_INVALID_TIMEOUT`, `MSG_TCP_NO_HANDLER`, `MSG_TCP_HANDLER_PANIC`, `MSG_TCP_STREAM_CLOSED`, `MSG_TCP_STREAM_CANCELED`, `MSG_TCP_NOT_REQUEST_STREAM`, `MSG_TCP_INVALID_FRAME` y `MSG_DATA_TOO_LARGE`.
