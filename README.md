# Josefina

Motor de base de datos documental escrito en Go (módulo `github.com/josefina`). Se usa únicamente a través de una API **HTTP + JSON**: las consultas y comandos son documentos JSON que se traducen a un constructor fluido de consultas en Go. Debajo, cada colección se guarda en `internal/store`, un store tipo WAL (log de solo agregado) de un nodo, concurrente, que es la base sobre la que se construirá la configuración multinodo.

## Requisitos

- Go 1.25.0 (`goenv local 1.25.0`).
- El repo forma parte del `go.work` del workspace `cgalvisleon/`, así que los cambios locales en `github.com/cgalvisleon/et` se usan sin publicar una versión.

## Ejecución

```bash
gofmt -w . && go run ./cmd/server -port 1377 -rpct 4377 -name josefina
```

| Flag | Variable de entorno | Por defecto | Uso |
|---|---|---|---|
| `-port` | `PORT` | `1370` | Puerto HTTP |
| `-rpct` | `RPC_PORT` | `4370` | Puerto RPC |
| `-name` | `DB_NAME` | `josefina` | Nombre del servicio |
| `-path_data` | `DB_PATH_DATA` | `./data/collections` | Datos (segmentos) |
| `-path_wald` | `DB_PATH_WALD` | `./data/wal` | Snapshot, compactación y recuperación (ver nota) |
| `-path_system` | `DB_PATH_SYSTEM` | `./data/system` | Catálogo del servidor |

> Nota: `internal/jdb` lee la ruta del WAL desde `DB_PATH_WAL`, no `DB_PATH_WALD`; mientras no se unifique, configúrala en el `.env` como `DB_PATH_WAL`.

Otros comandos:

```bash
go build ./cmd/server   # binario del servidor
go vet ./...
gofmt -w .
```

## Estructura

| Ruta | Contenido |
|---|---|
| `cmd/server` | Punto de entrada del servidor. |
| `pkg/server` | API REST: `/signin`, `/signout`, `/system`, `/query`, `/command`, `/uploadXls`, `/uploadCsv`, `/uploadDb`, `/version`, `/routes`. |
| `internal/jdb` | Servidor, catálogo (bases, esquemas, modelos, usuarios, sesiones) y motor de consultas/comandos. |
| `internal/store` | Motor de almacenamiento (ver abajo). |
| `internal/server` | Arma el servidor HTTP y monta las rutas. |
| `internal/msg` | Mensajes de error en inglés/español según `LANG`. |

## `internal/store` — motor de almacenamiento

Store de un solo nodo que guarda `id → []byte` en archivos de segmento de solo agregado, con un índice en memoria.

- Lecturas en paralelo, sin bloquearse con las escrituras.
- Escrituras y eliminaciones seguras bajo concurrencia; se aplican de a una, en orden.
- Cada registro lleva un número de secuencia (LSN) y un CRC; ante una caída se recupera leyendo el log.
- Guarda binario: codificar a JSON (u otro formato) es tarea de quien lo usa.

### Uso rápido

```go
fs, err := store.Open("./data/collections", "./data/wal", "clientes", store.ReadWrite)
if err != nil {
	return err
}
defer fs.Close()

ok, err := fs.Insert("c-001", []byte(`{"nombre":"Ana"}`)) // ok=false si ya existía
ok, err = fs.Update("c-001", []byte(`{"nombre":"Ana María"}`))

data, exists, err := fs.Get("c-001")

err = fs.ForEach(func(id string, data []byte) (bool, error) {
	fmt.Println(id, string(data))
	return true, nil // false para detener
}, true, 0, 100)

ok, err = fs.Delete("c-001")
```

### Métodos públicos

#### Apertura y ciclo de vida

- **`Open(pathData, pathWald, name string, mode Mode) (*FileStore, error)`**
  Abre el store `name` o lo crea si no existe. Los segmentos quedan en `pathData/segments/<name>/`; el snapshot, los temporales de compactación y la cuarentena de `Recover` quedan en `pathWald/{snapshot,compact,recover}/<name>/`. El nombre pasa por `Normalize`. Con `ReadOnly` el store debe existir y rechaza escrituras; con `ReadWrite` se puede escribir.

- **`Close() error`**
  Espera a que termine cualquier compactación, fuerza a disco lo escrito y cierra los archivos. Las lecturas que ya estaban en curso terminan antes de que se cierre su archivo. Detiene también `OnStats`.

- **`Empty() error`**
  Cierra el store y borra sus segmentos. Deja el store vacío.

- **`Recover(pathData, pathWald, name string) (et.Json, error)`**
  Repara el store en disco después de una falla (caída del proceso, corte de energía, compactación interrumpida). Debe ejecutarse **con el store cerrado** y antes de `Open`:
  - restaura o limpia los directorios de una compactación interrumpida;
  - borra el snapshot para que el índice se reconstruya completo;
  - trunca cada segmento en su último registro válido, copiando antes los bytes descartados a `pathWald/recover/<name>/`.

  Retorna un reporte con lo que hizo por segmento.

  ```go
  report, err := store.Recover("./data/collections", "./data/wal", "clientes")
  fs, err := store.Open("./data/collections", "./data/wal", "clientes", store.ReadWrite)
  ```

#### Escritura

- **`Insert(id string, data []byte) (bool, error)`**
  Guarda `data` solo si `id` no existe. Retorna `true` si insertó y `false` si el id ya existía.

- **`Update(id string, data []byte) (bool, error)`**
  Reemplaza `data` solo si `id` existe **y** el valor cambió. Si el binario es idéntico no escribe nada y retorna `false`.

- **`Delete(id string) (bool, error)`**
  Elimina `id` solo si existe. Retorna `true` si eliminó.

No hay un "upsert" a propósito. Para insertar o actualizar, se combinan:

```go
ok, err := fs.Update(id, data)
if err == nil && !ok && !fs.IsExist(id) {
	ok, err = fs.Insert(id, data)
}
```

#### Lectura

- **`Get(id string) ([]byte, bool, error)`**
  Retorna los datos de `id` y si existe.

- **`IsExist(id string) bool`**
  Indica si `id` existe, sin leer el disco.

- **`Count() int`**
  Número de ids vivos.

- **`Keys(asc bool, offset, limit int) []string`**
  Ids ordenados (ascendente o descendente) desde `offset`, hasta `limit` (`limit <= 0` sin límite). No lee los registros.

- **`ForEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error`**
  Recorre los registros en orden de id con `offset` y `limit`. Lee del disco en paralelo con hasta `runtime.NumCPU()` goroutines, pero llama a `fn` **de a uno y en orden**, desde la goroutine que llamó, así que `fn` no necesita locks. Si `fn` retorna `false` se detiene; si retorna un error, `ForEach` lo retorna. El conjunto de registros se fija al inicio y no se sostiene ningún lock, así que `fn` puede usar el store (por ejemplo, llamar a `Get` o `Update`).

#### Mantenimiento

- **`Compact() error`**
  Reescribe solo los registros vivos en segmentos nuevos y libera el espacio de los eliminados o sobrescritos. Se ejecuta sola cuando los tombstones superan el 10% de los ids (o `MIN_THRESHOLD_COMPACT`); llamarla a mano es opcional. No bloquea las lecturas y solo bloquea las escrituras al inicio y al final.

#### Monitoreo

- **`Stats() et.Json`**
  Tamaño y actividad del store: `count` (ids vivos), `size` (bytes en disco), `wal` (último LSN), `tomb_stones` (registros obsoletos) y las operaciones en ejecución `reading`, `inserting`, `updating`, `deleting`. Las escrituras que esperan su turno cuentan como en ejecución.

- **`OnStats(fn func(stats et.Json))`**
  Ancla `fn` para recibir `Stats` cada vez que cambian las operaciones en ejecución. Se llama desde una goroutine de fondo y agrupa cambios seguidos en el último estado, así que no frena las operaciones.

  ```go
  fs.OnStats(func(st et.Json) {
  	log.Println("lecturas:", st.Int("reading"), "escrituras:", st.Int("inserting")+st.Int("updating"))
  })
  ```

- **`ToJson() et.Json`, `ToString() string`**
  Configuración y estado del store (nombre, rutas, tamaño máximo de segmento, WAL, tombstones, tamaño).

- **`IsDebug() *FileStore`**
  Activa los logs de depuración. Retorna el mismo store para encadenar.

#### Replicación (base multinodo)

- **`Sync(fn func(change Change))`**
  Ancla `fn` para recibir cada `Insert`, `Update` y `Delete` aplicado, como `Change{Op, ID, Data, LSN}` con `Op` = `OpInsert`, `OpUpdate` u `OpDelete` (`Data` es `nil` en `OpDelete`). Los cambios llegan en orden exacto de LSN, sin huecos; es el lado que **envía** en una configuración multinodo. Las escrituras que no cambian nada no se emiten. Reglas para `fn`:
  - debe ser rápida: se ejecuta mientras las demás escrituras esperan;
  - no debe escribir en el mismo store (se bloquearía);
  - debe copiar `Data` si lo guarda para después.

  ```go
  // Nodo A replica cada cambio en el nodo B.
  a.Sync(func(c store.Change) {
  	status := store.Active
  	if c.Op == store.OpDelete {
  		status = store.Deleted
  	}
  	b.ApplyWalEntry(store.WalEntry{LSN: c.LSN, ID: c.ID, Data: c.Data, Status: status})
  })
  ```

- **`ApplyWalEntry(entry WalEntry) error`**
  Aplica un cambio recibido de otro nodo conservando su LSN. Es el lado que **recibe**. `WalEntry` tiene `LSN`, `ID`, `Data` y `Status` (`Active` o `Deleted`). También se emite a las funciones ancladas con `Sync`.

- **`WalSince(since uint64) ([]WalEntry, error)`**
  Retorna, en orden de escritura, las entradas del log con LSN mayor que `since`, para que un nodo se ponga al día. La compactación descarta los registros eliminados y sobrescritos, así que un nodo que quedó atrás de una compactación necesita una copia completa en lugar de `WalSince`.

#### Utilidades y tipos

- **`Normalize(input string) string`**
  Limpia un nombre para usarlo como nombre de archivo: quita espacios de los extremos, cambia espacios por `_`, elimina todo lo que no sea letra, número, `_` o `.`, y quita los números iniciales.

- **`Mode`**: `ReadOnly` o `ReadWrite` (modo de `Open`).
- **`Active`, `Deleted`**: estado de un registro en `WalEntry`.
- **`Op`, `Change`**: tipo de operación y cambio entregado por `Sync`.

### Configuración

| Variable | Por defecto | Uso |
|---|---|---|
| `RELSEG_SIZE` | `128` | Tamaño máximo de cada segmento, en MB. |
| `SYNC_ON_WRITE` | `true` | Fuerza a disco (fsync) cada escritura. Más seguro; con `false` es más rápido pero se pueden perder las últimas escrituras ante un corte de energía. |
| `MIN_THRESHOLD_COMPACT` | `1000` | Mínimo de tombstones para compactar automáticamente. |

### Consideraciones

- `Open` no ejecuta `Recover` por sí solo: después de un apagado inesperado, ejecuta `Recover` antes de `Open`.
- Las escrituras se aplican de a una; con `SYNC_ON_WRITE=true` su velocidad depende de la latencia del disco.
- El CRC de cada registro protege los datos, no el encabezado ni el id.
