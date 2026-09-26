package store

import "github.com/cgalvisleon/et/et"

// ---- Apertura y ciclo de vida ----

/**
* Open: Abre (o crea) el store y carga su índice.
* @param pathData, pathWald, name string, mode Mode
* @return *FileStore, error
**/
func Open(pathData, pathWald, name string, mode Mode) (*FileStore, error) {
	return open(pathData, pathWald, name, mode)
}

/**
* Close: Espera la compactación, hace fsync y retira todos los segmentos.
* @return error
**/
func (s *FileStore) Close() error {
	return s.close()
}

/**
* Empty: Cierra el store y borra sus datos.
* @return error
**/
func (s *FileStore) Empty() error {
	return s.empty()
}

/**
* Recover: Repara en disco un store después de una falla. El store debe estar cerrado.
* @param pathData, pathWald, name string
* @return et.Json, error
**/
func Recover(pathData, pathWald, name string) (et.Json, error) {
	return recover(pathData, pathWald, name)
}

// ---- Escritura ----

/**
* Insert: Guarda data solo si id no existe; retorna true si insertó.
* @param id string, data []byte
* @return bool (true if inserted), error
**/
func (s *FileStore) Insert(id string, data []byte) (bool, error) {
	return s.insert(id, data)
}

/**
* Update: Reemplaza data solo si id existe y el valor cambió; retorna true si actualizó.
* @param id string, data []byte
* @return bool (true if updated), error
**/
func (s *FileStore) Update(id string, data []byte) (bool, error) {
	return s.update(id, data)
}

/**
* Delete: Elimina id solo si existe; retorna true si eliminó.
* @param id string
* @return bool (true if deleted), error
**/
func (s *FileStore) Delete(id string) (bool, error) {
	return s.delete(id)
}

// ---- Lectura ----

/**
* Get: Retorna los datos de id, si existe.
* @param id string
* @return []byte, bool (true if id exists), error
**/
func (s *FileStore) Get(id string) ([]byte, bool, error) {
	return s.get(id)
}

/**
* IsExist: Indica si id existe.
* @param id string
* @return bool
**/
func (s *FileStore) IsExist(id string) bool {
	return s.isExist(id)
}

/**
* Count: Retorna el número de claves vivas.
* @return int
**/
func (s *FileStore) Count() int {
	return s.count()
}

/**
* Keys: Retorna las claves ordenadas asc/desc con offset y limit, sin leer los registros.
* @param asc bool, offset int, limit int
* @return []string
**/
func (s *FileStore) Keys(asc bool, offset, limit int) []string {
	return s.keys(asc, offset, limit)
}

/**
* ForEach: Recorre los registros vivos ordenados asc/desc con offset y limit; fn se
* llama de a uno y en orden, y si retorna false se detiene.
* @param fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *FileStore) ForEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	return s.forEach(fn, asc, offset, limit)
}

// ---- Mantenimiento ----

/**
* Compact: Reescribe solo los registros vivos en segmentos nuevos.
* @return error
**/
func (s *FileStore) Compact() error {
	return s.compact()
}

// ---- Monitoreo ----

/**
* Stats: Retorna el tamaño del store y cuántas operaciones están en ejecución.
* @return et.Json
**/
func (s *FileStore) Stats() et.Json {
	return s.stats()
}

/**
* OnStats: Ancla fn para recibir Stats cada vez que cambian las operaciones en ejecución.
* @param fn func(stats et.Json)
**/
func (s *FileStore) OnStats(fn func(stats et.Json)) {
	s.onStats(fn)
}

/**
* ToJson: Retorna el estado del store como JSON.
* @return et.Json
**/
func (s *FileStore) ToJson() et.Json {
	return s.toJson()
}

/**
* ToString: Retorna el estado del store como texto.
* @return string
**/
func (s *FileStore) ToString() string {
	return s.toString()
}

/**
* IsDebug: Activa los logs de depuración.
* @return *FileStore
**/
func (s *FileStore) IsDebug() *FileStore {
	return s.isDebug()
}

// ---- Replicación ----

/**
* Sync: Ancla fn para recibir cada Insert, Update y Delete aplicado, en orden de LSN.
* @param fn func(change Change)
**/
func (s *FileStore) Sync(fn func(change Change)) {
	s.sync(fn)
}

/**
* ApplyWalEntry: Aplica una entrada recibida del líder conservando su LSN (solo replicación).
* @param entry WalEntry
* @return error
**/
func (s *FileStore) ApplyWalEntry(entry WalEntry) error {
	return s.applyWalEntry(entry)
}

/**
* WalSince: Retorna, en orden de escritura, las entradas con LSN mayor que since.
* @param since uint64
* @return []WalEntry, error
**/
func (s *FileStore) WalSince(since uint64) ([]WalEntry, error) {
	return s.walSince(since)
}

/**
* ToJson: Retorna el cambio como JSON.
* @return et.Json
**/
func (s *Change) ToJson() et.Json {
	return s.toJson()
}

/**
* ToJson: Retorna la entrada del WAL como JSON.
* @return et.Json
**/
func (e *WalEntry) ToJson() et.Json {
	return e.toJson()
}

// ---- Utilidades ----

/**
* Normalize: Limpia un nombre para usarlo como nombre de archivo.
* @param input string
* @return string
**/
func Normalize(input string) string {
	return normalize(input)
}
