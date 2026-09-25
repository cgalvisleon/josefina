package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/msg"
)

const (
	packageName     = "store"
	maxIdLen        = 65535
	fixedHeaderSize = 19 // LSN(8) + DataLen(4) + CRC(4) + IDLen(2) + Status(1)
)

/**
* Normalize: Limpia un nombre para usarlo como nombre de archivo.
* @param input string
* @return string
**/
func Normalize(input string) string {
	// 1. Quitar espacios al inicio y final
	s := strings.TrimSpace(input)

	// 2. Reemplazar uno o más espacios por _
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "_")

	// 3. Eliminar todo lo que no sea letra, número, _ o .
	s = regexp.MustCompile(`[^a-zA-Z0-9_.]`).ReplaceAllString(s, "")

	// 4. Garantizar que no empiece con número
	s = regexp.MustCompile(`^[0-9]+`).ReplaceAllString(s, "")

	return s
}

/**
* newRecordHeaderAt: Codifica el encabezado de un registro con el LSN dado.
* @param lsn uint64, id string, data []byte, status byte
* @return recordHeader, []byte, error
**/
func newRecordHeaderAt(lsn uint64, id string, data []byte, status byte) (recordHeader, []byte, error) {
	idBytes := []byte(id)
	idLen := len(idBytes)

	if idLen == 0 || idLen > maxIdLen {
		return recordHeader{}, nil, errors.New(msg.MSG_INVALID_ID_LENGTH)
	}

	dataLen := len(data)
	if dataLen > math.MaxUint32 {
		return recordHeader{}, nil, errors.New(msg.MSG_DATA_TOO_LARGE)
	}

	headerLen := fixedHeaderSize + idLen
	result := recordHeader{
		LSN:     lsn,
		DataLen: uint32(dataLen),
		CRC:     checksum(data),
		IDLen:   uint16(idLen),
		Status:  status,
	}

	// Layout: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1]
	header := make([]byte, headerLen)
	binary.BigEndian.PutUint64(header[0:8], result.LSN)
	putUint32(header[8:12], result.DataLen)
	putUint32(header[12:16], result.CRC)
	putUint16(header[16:18], result.IDLen)
	copy(header[18:18+idLen], idBytes)
	header[18+idLen] = status

	return result, header, nil
}

type Mode int

const (
	ReadOnly Mode = iota
	ReadWrite
)

type FileStore struct {
	ID                  string                `json:"id"`
	Name                string                `json:"name"`
	WAL                 counter[uint64]       `json:"wal"`         // último LSN escrito
	TombStones          counter[int]          `json:"tomb_stones"` // registros obsoletos pendientes de compactar
	Path                string                `json:"path"`
	PathSnapshot        string                `json:"path_snapshot"`
	PathCompact         string                `json:"path_compact"`
	PathRecover         string                `json:"path_recover"` // cuarentena de bytes corruptos (Recover)
	MaxSegment          int64                 `json:"max_segment"`
	SyncOnWrite         bool                  `json:"sync_on_write"`
	Size                counter[int64]        `json:"size"`
	MinThresholdCompact int                   `json:"min_threshold_compact"`
	writeMu             sync.Mutex            `json:"-"` // serializa log + índice: append, rotación, compaction, close
	indexMu             sync.RWMutex          `json:"-"` // protege index, segments y active
	segments            []*segment            `json:"-"` // segmentos de datos
	active              *segment              `json:"-"` // segmento activo para escritura
	index               map[string]*recordRef `json:"-"` // índice en memoria
	mode                Mode                  `json:"-"` // modo de operación
	compacting          int32                 `json:"-"` // 0 = idle, 1 = running
	compactWg           sync.WaitGroup        `json:"-"` // espera que termine la goroutine de compaction
	compactMu           sync.Mutex            `json:"-"` // una sola compaction a la vez (automática o Prune)
	isDebug             bool                  `json:"-"`
	reading             counter[int]          `json:"-"` // Get/ForEach en ejecución
	inserting           counter[int]          `json:"-"` // Insert en ejecución (incluye los que esperan writeMu)
	updating            counter[int]          `json:"-"` // Update en ejecución (incluye los que esperan writeMu)
	deleting            counter[int]          `json:"-"` // Delete en ejecución (incluye los que esperan writeMu)
	syncFns             []func(Change)        `json:"-"` // funciones ancladas con Sync (bajo writeMu)
	statsFns            []func(et.Json)       `json:"-"` // funciones ancladas con OnStats
	statsMu             sync.Mutex            `json:"-"` // protege statsFns
	statsCh             chan struct{}         `json:"-"` // aviso de cambio de Stats (buffer 1)
	statsDone           chan struct{}         `json:"-"` // detiene statsLoop
	statsStart          sync.Once             `json:"-"`
	statsStop           sync.Once             `json:"-"`
}

/**
* ToJson: Retorna el estado del store como JSON.
* @return et.Json
**/
func (s *FileStore) ToJson() et.Json {
	return et.Json{
		"id":                    s.ID,
		"name":                  s.Name,
		"wal":                   s.WAL.count(),
		"tomb_stones":           s.TombStones.count(),
		"path":                  s.Path,
		"path_snapshot":         s.PathSnapshot,
		"path_compact":          s.PathCompact,
		"path_recover":          s.PathRecover,
		"max_segment":           s.MaxSegment,
		"sync_on_write":         s.SyncOnWrite,
		"size":                  s.Size.count(),
		"min_threshold_compact": s.MinThresholdCompact,
	}
}

/**
* ToString: Retorna el estado del store como texto.
* @return string
**/
func (s *FileStore) ToString() string {
	return s.ToJson().ToString()
}

/**
* IsDebug: Activa los logs de depuración.
* @return *FileStore
**/
func (s *FileStore) IsDebug() *FileStore {
	s.isDebug = true
	return s
}

/**
* Count: Retorna el número de claves vivas.
* @return int
**/
func (s *FileStore) Count() int {
	return s.countIndex()
}

/**
* loadSegments: Abre los archivos de segmento existentes.
* @return error
**/
func (s *FileStore) loadSegments() error {
	files, err := os.ReadDir(s.Path)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, f := range files {
		name := f.Name()
		path := filepath.Join(s.Path, name)
		st, _ := os.Stat(path)

		flag := os.O_CREATE | os.O_RDWR
		if s.mode == ReadOnly {
			flag = os.O_RDONLY
		}
		fd, err := os.OpenFile(path, flag, 0644)
		if err != nil {
			return err
		}

		size := st.Size()
		var seg *segment
		if s.mode == ReadOnly {
			seg = newReadOnlySegment(fd, size, name)
		} else {
			if _, err := fd.Seek(size, io.SeekStart); err != nil {
				return err
			}
			seg = newSegment(fd, size, name)
		}
		s.segments = append(s.segments, seg)
		s.Size.add(size)
		if s.isDebug {
			logs.Log(packageName, "load:segments:", s.Path, ":", seg.toString())
		}
	}

	if len(s.segments) == 0 {
		if s.mode == ReadOnly {
			return nil
		}
		return s.newSegment()
	}

	s.active = s.segments[len(s.segments)-1]
	return nil
}

/**
* newSegment: Sella el segmento activo y crea uno nuevo (requiere writeMu).
* @return error
**/
func (s *FileStore) newSegment() error {
	if s.active != nil {
		if err := s.active.seal(); err != nil {
			return err
		}
	}

	name := fmt.Sprintf("segment-%06d.dat", len(s.segments)+1)
	path := filepath.Join(s.Path, name)

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	// Get/Read/ForEach leen segments bajo indexMu: el cambio lo toma aunque ya se tenga writeMu.
	seg := newSegment(fd, 0, name)
	s.indexMu.Lock()
	s.segments = append(s.segments, seg)
	s.active = seg
	s.indexMu.Unlock()
	if s.isDebug {
		logs.Log(packageName, "new:segment:", s.Path, ":", seg.toString())
	}

	return nil
}

/**
* appendRecordLocked: Agrega un registro al segmento activo, rotándolo si está lleno (requiere writeMu).
* lsn 0 asigna el siguiente LSN local; lsn > 0 conserva el LSN recibido (replicación).
* @param lsn uint64, id string, data []byte, status byte
* @return *recordRef, uint64, error
**/
func (s *FileStore) appendRecordLocked(lsn uint64, id string, data []byte, status byte) (*recordRef, uint64, error) {
	recordSize := int64(fixedHeaderSize) + int64(len(id)) + int64(len(data))
	if s.active.size+recordSize > s.MaxSegment {
		if err := s.newSegment(); err != nil {
			return nil, 0, err
		}

		if err := s.createSnapshot(); err != nil {
			return nil, 0, err
		}
	}

	if lsn == 0 {
		lsn = s.WAL.inc()
	} else {
		s.WAL.setMax(lsn)
	}

	ref, err := s.active.writeRecord(lsn, id, data, status)
	if err != nil {
		return nil, 0, err
	}
	ref.segment = len(s.segments) - 1
	s.Size.add(recordSize)
	if s.SyncOnWrite {
		if err := s.active.flush(); err != nil {
			return nil, 0, err
		}
	}

	return ref, lsn, nil
}

/**
* compactIfNeeded: Lanza una compactación en segundo plano si los tombstones superan el 10% del
* índice (o MinThresholdCompact).
**/
func (s *FileStore) compactIfNeeded() {
	threshold := max(int(float64(s.countIndex())*0.1), s.MinThresholdCompact)
	if s.TombStones.count() <= threshold {
		return
	}

	if atomic.CompareAndSwapInt32(&s.compacting, 0, 1) {
		s.compactWg.Add(1)
		go func() {
			defer s.compactWg.Done()
			defer atomic.StoreInt32(&s.compacting, 0)
			if err := s.Compact(); err != nil && s.isDebug {
				logs.Debug("compact error:", err)
			}
		}()
	}
}

/**
* getIndex: Retorna la referencia de id.
* @param id string
* @return *recordRef, bool
**/
func (s *FileStore) getIndex(id string) (*recordRef, bool) {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	ref, exists := s.index[id]
	return ref, exists
}

/**
* countIndex: Retorna el número de claves del índice.
* @return int
**/
func (s *FileStore) countIndex() int {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return len(s.index)
}

/**
* setIndex: Guarda la referencia de id; retorna true si ya existía.
* @param id string, ref *recordRef
* @return bool (true if id already existed)
**/
func (s *FileStore) setIndex(id string, ref *recordRef) bool {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	return s.setIndexLocked(id, ref)
}

/**
* setIndexLocked: Igual que setIndex, pero requiere indexMu tomado.
* @param id string, ref *recordRef
* @return bool (true if id already existed)
**/
func (s *FileStore) setIndexLocked(id string, ref *recordRef) bool {
	_, exists := s.index[id]
	s.index[id] = ref
	if s.isDebug {
		logs.Debug("put:", s.Path, ":lsn:", s.WAL.count(), ":ID:", id, ":ref:", ref.toString())
	}
	return exists
}

/**
* deleteIndex: Quita id del índice; retorna true si existía.
* @param id string
* @return bool (true if id existed)
**/
func (s *FileStore) deleteIndex(id string) bool {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	return s.deleteIndexLocked(id)
}

/**
* deleteIndexLocked: Igual que deleteIndex, pero requiere indexMu tomado.
* @param id string
* @return bool (true if id existed)
**/
func (s *FileStore) deleteIndexLocked(id string) bool {
	if _, exists := s.index[id]; !exists {
		return false
	}
	delete(s.index, id)
	return true
}

/**
* rebuildIndex: Aplica un segmento al índice (requiere indexMu).
* @param segIndex int
* @return error
**/
func (s *FileStore) rebuildIndex(segIndex int) error {
	if s.index == nil {
		s.index = make(map[string]*recordRef)
	}

	return s.segments[segIndex].scan(0, func(offset int64, h recordHeader, data []byte) error {
		if h.Status == Active {
			s.setIndexLocked(h.ID, &recordRef{segment: segIndex, offset: offset, length: h.DataLen})
		} else if h.Status == Deleted {
			s.deleteIndexLocked(h.ID)
		}

		// Restaurar WAL al LSN más alto visto en disco
		s.WAL.setMax(h.LSN)
		return nil
	})
}

/**
* buildIndex: Aplica el segmento activo sobre el índice cargado del snapshot.
* @return error
**/
func (s *FileStore) buildIndex() error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	idx := len(s.segments) - 1
	return s.rebuildIndex(idx)
}

/**
* sortedKeysLocked: Retorna las claves ordenadas asc/desc con offset y limit (limit <= 0 sin límite);
* requiere indexMu.
* @param asc bool, offset int, limit int
* @return []string
**/
func (s *FileStore) sortedKeysLocked(asc bool, offset, limit int) []string {
	offset = max(offset, 0)
	if offset >= len(s.index) {
		return []string{}
	}

	keys := slices.Sorted(maps.Keys(s.index))
	if !asc {
		slices.Reverse(keys)
	}

	end := len(keys)
	if limit > 0 {
		end = min(offset+limit, end)
	}

	return keys[offset:end]
}

/**
* pinLocked: Retorna el segmento de ref marcado como en lectura; requiere indexMu y llamar release().
* @param ref *recordRef
* @return *segment, error
**/
func (s *FileStore) pinLocked(ref *recordRef) (*segment, error) {
	if ref == nil || ref.segment < 0 || ref.segment >= len(s.segments) {
		return nil, errors.New(msg.MSG_CORRUPTED_RECORD)
	}

	seg := s.segments[ref.segment]
	seg.acquire()
	return seg, nil
}

/**
* rebuildIndexes: Reconstruye el índice recorriendo todos los segmentos.
* @return error
**/
func (s *FileStore) rebuildIndexes() error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	s.index = make(map[string]*recordRef)
	for i := range s.segments {
		if err := s.rebuildIndex(i); err != nil {
			return err
		}
	}

	return nil
}

/**
* Close: Espera la compactación, hace fsync y retira todos los segmentos.
* @return error
**/
func (s *FileStore) Close() error {
	s.stopStats()
	s.compactWg.Wait()

	// compactMu también espera una Compact() lanzada por Prune.
	s.compactMu.Lock()
	defer s.compactMu.Unlock()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	// fsync y retiro de cada segmento: se cierra ahora o cuando termine su último lector.
	var closeErr error
	for _, seg := range s.segments {
		if err := seg.seal(); err != nil && closeErr == nil {
			closeErr = err
		}
		seg.retire()
	}

	return closeErr
}

/**
* Empty: Cierra el store y borra sus datos.
* @return error
**/
func (s *FileStore) Empty() error {
	err := s.Close()
	if err != nil {
		return err
	}

	s.indexMu.Lock()
	s.index = make(map[string]*recordRef)
	s.indexMu.Unlock()
	s.WAL.set(0)
	s.TombStones.set(0)
	s.Size.set(0)

	defer os.RemoveAll(s.Path)
	return nil
}

/**
* syncRef: Apunta id a una referencia de otro store (no escribe en el log).
* @param id string, ref *recordRef, ownerId string
**/
func (s *FileStore) syncRef(id string, ref *recordRef, ownerId string) {
	if s.ID == ownerId {
		return
	}
	s.setIndex(id, ref)
}

/**
* checkWrite: Valida que el store acepte escrituras y que id no esté vacío.
* @param id string
* @return error
**/
func (s *FileStore) checkWrite(id string) error {
	if s.mode == ReadOnly {
		return errors.New(msg.MSG_STORE_IS_READ_ONLY)
	}

	if id == "" {
		return errors.New(msg.MSG_ID_IS_REQUIRED)
	}

	return nil
}

/**
* put: Escribe data en el log y actualiza el índice (requiere writeMu); retorna true si id ya existía.
* @param id string, data []byte
* @return bool (true if id already existed), uint64 (LSN), error
**/
func (s *FileStore) put(id string, data []byte) (bool, uint64, error) {
	ref, lsn, err := s.appendRecordLocked(0, id, data, Active)
	if err != nil {
		return false, 0, err
	}

	exists := s.setIndex(id, ref)
	if exists {
		s.TombStones.inc()
	}

	return exists, lsn, nil
}

/**
* Insert: Guarda data solo si id no existe; retorna true si insertó.
* @param id string, data []byte
* @return bool (true if inserted), error
**/
func (s *FileStore) Insert(id string, data []byte) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	defer s.track(&s.inserting)()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, exists := s.getIndex(id); exists {
		return false, nil
	}

	_, lsn, err := s.put(id, data)
	if err != nil {
		return false, err
	}

	s.emitLocked(Change{Op: OpInsert, ID: id, Data: data, LSN: lsn})

	return true, nil
}

/**
* Update: Reemplaza data solo si id existe y el valor cambió; retorna true si actualizó.
* @param id string, data []byte
* @return bool (true if updated), error
**/
func (s *FileStore) Update(id string, data []byte) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	defer s.track(&s.updating)()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	ref, exists := s.getIndex(id)
	if !exists {
		return false, nil
	}

	old, err := s.readPinned(ref, id)
	if err != nil {
		return false, err
	}
	if bytes.Equal(old, data) {
		return false, nil
	}

	_, lsn, err := s.put(id, data)
	if err != nil {
		return false, err
	}

	s.emitLocked(Change{Op: OpUpdate, ID: id, Data: data, LSN: lsn})
	s.compactIfNeeded()

	return true, nil
}

/**
* Delete: Elimina id solo si existe; retorna true si eliminó.
* @param id string
* @return bool (true if deleted), error
**/
func (s *FileStore) Delete(id string) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	defer s.track(&s.deleting)()

	// Verificación, tombstone e índice bajo writeMu: nada se cuela entre ellos.
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, exists := s.getIndex(id); !exists {
		return false, nil
	}

	_, lsn, err := s.appendRecordLocked(0, id, nil, Deleted)
	if err != nil {
		return false, err
	}

	s.deleteIndex(id)
	s.TombStones.inc()

	if s.isDebug {
		logs.Debug("deleted:", s.Path, ":total:", s.countIndex(), ":ID:", id)
	}

	s.emitLocked(Change{Op: OpDelete, ID: id, LSN: lsn})
	s.compactIfNeeded()

	return true, nil
}

/**
* IsExist: Indica si id existe.
* @param id string
* @return bool
**/
func (s *FileStore) IsExist(id string) bool {
	if id == "" {
		return false
	}

	_, existed := s.getIndex(id)
	return existed
}

/**
* readPinned: Lee el registro de ref sin sostener indexMu durante la lectura; con id vacío lo lee del disco.
* @param ref *recordRef, id string
* @return []byte, error
**/
func (s *FileStore) readPinned(ref *recordRef, id string) ([]byte, error) {
	s.indexMu.RLock()
	seg, err := s.pinLocked(ref)
	s.indexMu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer seg.release()

	if id == "" {
		return seg.read(ref)
	}
	return seg.readRecord(ref, id)
}

/**
* read: Lee el registro de ref.
* @param ref *recordRef
* @return []byte, error
**/
func (s *FileStore) read(ref *recordRef) ([]byte, error) {
	return s.readPinned(ref, "")
}

/**
* readHeader: Lee el encabezado del registro de ref.
* @param ref *recordRef
* @return recordHeader, error
**/
func (s *FileStore) readHeader(ref *recordRef) (recordHeader, error) {
	s.indexMu.RLock()
	seg, err := s.pinLocked(ref)
	s.indexMu.RUnlock()
	if err != nil {
		return recordHeader{}, err
	}
	defer seg.release()

	return seg.readHeader(ref)
}

/**
* Get: Retorna los datos de id, si existe.
* @param id string
* @return []byte, bool (true if id exists), error
**/
func (s *FileStore) Get(id string) ([]byte, bool, error) {
	defer s.track(&s.reading)()

	// Búsqueda y fijado bajo un mismo RLock; la lectura a disco va después.
	s.indexMu.RLock()
	ref, exists := s.index[id]
	if !exists {
		s.indexMu.RUnlock()
		return nil, false, nil
	}
	seg, err := s.pinLocked(ref)
	s.indexMu.RUnlock()
	if err != nil {
		return nil, false, err
	}
	defer seg.release()

	data, err := seg.readRecord(ref, id)
	if err != nil {
		return nil, false, err
	}

	return data, true, nil
}

/**
* forEachItem: Registro seleccionado por ForEach, con su segmento fijado.
**/
type forEachItem struct {
	id  string
	ref *recordRef
	seg *segment
}

/**
* forEachResult: Resultado de leer un forEachItem.
**/
type forEachResult struct {
	data []byte
	err  error
}

/**
* ForEach: Llama a fn por cada registro en orden de clave (asc/desc) con offset y limit
* (limit <= 0 sin límite). Lee en paralelo con hasta NumCPU goroutines, pero llama
* a fn de a uno y en orden; si fn retorna false se detiene. No sostiene locks,
* así que fn puede usar el store.
* @param fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *FileStore) ForEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	defer s.track(&s.reading)()

	// ---- Selección: claves y segmentos fijados bajo un solo RLock ----
	s.indexMu.RLock()
	keys := s.sortedKeysLocked(asc, offset, limit)
	items := make([]forEachItem, len(keys))
	pinned := make(map[*segment]bool)
	for i, id := range keys {
		ref := s.index[id]
		seg, err := s.pinLocked(ref)
		if err != nil {
			s.indexMu.RUnlock()
			for seg := range pinned {
				seg.release()
			}
			return err
		}
		if pinned[seg] {
			seg.release() // basta un pin por segmento
		}
		pinned[seg] = true
		items[i] = forEachItem{id: id, ref: ref, seg: seg}
	}
	s.indexMu.RUnlock()
	defer func() {
		for seg := range pinned {
			seg.release()
		}
	}()

	n := len(items)
	if n == 0 {
		return nil
	}

	// ---- Lectura concurrente con entrega ordenada ----
	// Los workers leen en paralelo y dejan cada resultado en slots[i%window]; esta
	// goroutine los toma en orden y llama a fn. sem limita lo pendiente a window.
	workers := min(runtime.NumCPU(), n)
	window := workers * 4
	slots := make([]chan forEachResult, window)
	for i := range slots {
		slots[i] = make(chan forEachResult, 1)
	}
	jobs := make(chan int, window)
	sem := make(chan struct{}, window)
	done := make(chan struct{})

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for i := range jobs {
				select {
				case <-done:
					slots[i%window] <- forEachResult{}
					continue
				default:
				}
				data, err := items[i].seg.readRecord(items[i].ref, items[i].id)
				slots[i%window] <- forEachResult{data: data, err: err}
			}
		})
	}

	wg.Go(func() {
		defer close(jobs)
		for i := range n {
			select {
			case sem <- struct{}{}:
			case <-done:
				return
			}
			jobs <- i
		}
	})

	// Detener el productor y esperar a todos antes de liberar los segmentos (incluso si fn entra en pánico).
	defer func() {
		close(done)
		wg.Wait()
	}()

	for i := range n {
		r := <-slots[i%window]
		<-sem
		if r.err != nil {
			return r.err
		}

		cont, err := fn(items[i].id, r.data)
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}
	}

	return nil
}

/**
* Keys: Retorna las claves ordenadas asc/desc con offset y limit, sin leer los registros.
* @param asc bool, offset int, limit int
* @return []string
**/
func (s *FileStore) Keys(asc bool, offset, limit int) []string {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	return s.sortedKeysLocked(asc, offset, limit)
}

/**
* prune: Compacta y reconstruye el índice.
* @return error
**/
func (s *FileStore) prune() error {
	err := s.Compact()
	if err != nil {
		return err
	}

	err = s.rebuildIndexes()
	if err != nil {
		return err
	}

	return nil
}

/**
* newFileStore: Arma un FileStore con sus rutas y configuración, sin tocar el disco.
* @param pathData, pathWald, name string, mode Mode
* @return *FileStore
**/
func newFileStore(pathData, pathWald, name string, mode Mode) *FileStore {
	name = Normalize(name)
	return &FileStore{
		Name:                name,
		Path:                filepath.Join(pathData, "segments", name),
		PathSnapshot:        filepath.Join(pathWald, "snapshot", name),
		PathCompact:         filepath.Join(pathWald, "compact", name),
		PathRecover:         filepath.Join(pathWald, "recover", name),
		MaxSegment:          envar.GetInt64("RELSEG_SIZE", 128) * 1024 * 1024,
		MinThresholdCompact: envar.GetInt("MIN_THRESHOLD_COMPACT", 1000),
		SyncOnWrite:         envar.GetBool("SYNC_ON_WRITE", true),
		index:               make(map[string]*recordRef),
		mode:                mode,
		statsCh:             make(chan struct{}, 1),
		statsDone:           make(chan struct{}),
	}
}

/**
* Open: Abre (o crea) el store y carga su índice.
* @param pathData, pathWald, name string, mode Mode
* @return *FileStore, error
**/
func Open(pathData, pathWald, name string, mode Mode) (*FileStore, error) {
	fs := newFileStore(pathData, pathWald, name, mode)

	if mode == ReadOnly {
		if _, err := os.Stat(fs.Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: store not found at %s", errors.New(msg.MSG_STORE_NOT_FOUND), fs.Path)
		}
	} else {
		if err := os.MkdirAll(fs.Path, 0755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(fs.PathSnapshot, 0755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(fs.PathCompact, 0755); err != nil {
			return nil, err
		}
	}

	if err := fs.loadSegments(); err != nil {
		return nil, fmt.Errorf("loadSegments: %w", err)
	}

	// Con snapshot solo se aplica el último segmento; sin snapshot se recorren todos.
	snapshotLoaded, snapErr := fs.tryLoadSnapshot()
	if snapErr != nil && fs.isDebug {
		logs.Debug("snapshot unavailable, full rebuild:", snapErr)
	}
	if snapshotLoaded {
		if err := fs.buildIndex(); err != nil {
			return nil, fmt.Errorf("buildIndex: %w", err)
		}
	} else {
		if err := fs.rebuildIndexes(); err != nil {
			return nil, fmt.Errorf("rebuildIndexes: %w", err)
		}
	}

	return fs, nil
}
