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
* normalize
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
* newRecordHeaderAt: Encodes a WAL record header at the given LSN.
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

type SetIndexFn func(fls *FileStore, idx string, ref *RecordRef)
type Putfn func(fls *FileStore, idx string, old, new []byte)
type Deletefn func(fls *FileStore, idx string, old []byte)

type FileStore struct {
	ID                  string                `json:"id"`
	Name                string                `json:"name"`
	WAL                 counter[uint64]       `json:"wal"`         // último LSN escrito
	TombStones          counter[int]          `json:"tomb_stones"` // registros obsoletos pendientes de compactar
	Path                string                `json:"path"`
	PathSnapshot        string                `json:"path_snapshot"`
	PathCompact         string                `json:"path_compact"`
	MaxSegment          int64                 `json:"max_segment"`
	SyncOnWrite         bool                  `json:"sync_on_write"`
	Size                counter[int64]        `json:"size"`
	MinThresholdCompact int                   `json:"min_threshold_compact"`
	writeMu             sync.Mutex            `json:"-"` // serializa log + índice: append, rotación, compaction, close
	indexMu             sync.RWMutex          `json:"-"` // protege index, keys, segments y active
	segments            []*segment            `json:"-"` // segmentos de datos
	active              *segment              `json:"-"` // segmento activo para escritura
	index               map[string]*RecordRef `json:"-"` // índice en memoria
	mode                Mode                  `json:"-"` // modo de operación
	compacting          int32                 `json:"-"` // 0 = idle, 1 = running
	compactWg           sync.WaitGroup        `json:"-"` // espera que termine la goroutine de compaction
	compactMu           sync.Mutex            `json:"-"` // una sola compaction a la vez (automática o Prune)
	isDebug             bool                  `json:"-"`
}

/**
* ToJson
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
		"max_segment":           s.MaxSegment,
		"sync_on_write":         s.SyncOnWrite,
		"size":                  s.Size.count(),
		"min_threshold_compact": s.MinThresholdCompact,
	}
}

/**
* ToString
* @return string
 */
func (s *FileStore) ToString() string {
	return s.ToJson().ToString()
}

/**
* IsDebug
* @return *FileStore
**/
func (s *FileStore) IsDebug() *FileStore {
	s.isDebug = true
	return s
}

/**
* Count
* @return int
**/
func (s *FileStore) Count() int {
	return s.countIndex()
}

/**
* loadSegments
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
			logs.Log(packageName, "load:segments:", s.Path, ":", seg.ToString())
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
* newSegment
* @return error
**/
func (s *FileStore) newSegment() error {
	if s.active != nil {
		if err := s.active.Seal(); err != nil {
			return err
		}
	}

	name := fmt.Sprintf("segment-%06d.dat", len(s.segments)+1)
	path := filepath.Join(s.Path, name)

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	// segments/active are read under indexMu by Get/Read/ForEach, so the swap
	// must take it even though the caller already holds writeMu.
	seg := newSegment(fd, 0, name)
	s.indexMu.Lock()
	s.segments = append(s.segments, seg)
	s.active = seg
	s.indexMu.Unlock()
	if s.isDebug {
		logs.Log(packageName, "new:segment:", s.Path, ":", seg.ToString())
	}

	return nil
}

/**
* appendRecordLocked: Appends a record to the active segment, rotating it when full.
* Caller must hold writeMu for the whole write + index update so both happen as one step.
* lsn == 0 assigns the next local LSN; lsn > 0 keeps the caller's LSN (replication path)
* and advances the local counter if needed.
* @param lsn uint64, id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *FileStore) appendRecordLocked(lsn uint64, id string, data []byte, status byte) (*RecordRef, error) {
	recordSize := int64(fixedHeaderSize) + int64(len(id)) + int64(len(data))
	if s.active.size+recordSize > s.MaxSegment {
		if err := s.newSegment(); err != nil {
			return nil, err
		}

		if err := s.CreateSnapshot(); err != nil {
			return nil, err
		}
	}

	if lsn == 0 {
		lsn = s.WAL.inc()
	} else {
		s.WAL.setMax(lsn)
	}

	ref, err := s.active.WriteRecord(lsn, id, data, status)
	if err != nil {
		return nil, err
	}
	ref.segment = len(s.segments) - 1
	s.Size.add(recordSize)
	if s.SyncOnWrite {
		if err := s.active.Sync(); err != nil {
			return nil, err
		}
	}

	return ref, nil
}

/**
* compactIfNeeded: Starts a background compaction once tombstones exceed 10% of
* the index (or MinThresholdCompact, whichever is greater).
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
* getIndex: Returns the reference stored for id
* @param id string
* @return *RecordRef, bool
**/
func (s *FileStore) getIndex(id string) (*RecordRef, bool) {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	ref, exists := s.index[id]
	return ref, exists
}

/**
* countIndex: Returns the number of live keys
* @return int
**/
func (s *FileStore) countIndex() int {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return len(s.index)
}

/**
* setIndex: Stores ref for id
* @param id string, ref *RecordRef
* @return bool (true if id already existed)
**/
func (s *FileStore) setIndex(id string, ref *RecordRef) bool {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	return s.setIndexLocked(id, ref)
}

/**
* setIndexLocked: Same as setIndex but assumes the caller already holds indexMu
* @param id string, ref *RecordRef
* @return bool (true if id already existed)
**/
func (s *FileStore) setIndexLocked(id string, ref *RecordRef) bool {
	_, exists := s.index[id]
	s.index[id] = ref
	if s.isDebug {
		logs.Debug("put:", s.Path, ":lsn:", s.WAL.count(), ":ID:", id, ":ref:", ref.ToString())
	}
	return exists
}

/**
* deleteIndex: Removes id from the index
* @param id string
* @return bool (true if id existed)
**/
func (s *FileStore) deleteIndex(id string) bool {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	return s.deleteIndexLocked(id)
}

/**
* deleteIndexLocked: Same as deleteIndex but assumes the caller already holds indexMu
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
* rebuildIndex: Replays a segment into the index. Caller must hold indexMu.
* @param segIndex int
* @return error
**/
func (s *FileStore) rebuildIndex(segIndex int) error {
	if s.index == nil {
		s.index = make(map[string]*RecordRef)
	}

	return s.segments[segIndex].scan(0, func(offset int64, h recordHeader, data []byte) error {
		if h.Status == Active {
			s.setIndexLocked(h.ID, &RecordRef{segment: segIndex, offset: offset, length: h.DataLen})
		} else if h.Status == Deleted {
			s.deleteIndexLocked(h.ID)
		}

		// Restaurar WAL al LSN más alto visto en disco
		s.WAL.setMax(h.LSN)
		return nil
	})
}

/**
* buildIndex
* @return error
**/
func (s *FileStore) buildIndex() error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	idx := len(s.segments) - 1
	return s.rebuildIndex(idx)
}

/**
* sortedKeysLocked: Returns the ids of the index sorted asc/desc, windowed by
* offset and limit (limit <= 0 means no limit). Caller must hold indexMu.
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
* pinLocked: Returns the segment ref points at, registered as being read so a
* concurrent Compact()/Close() retires it without closing the file underneath.
* Caller must hold indexMu (read) and must call release() on the result.
* @param ref *RecordRef
* @return *segment, error
**/
func (s *FileStore) pinLocked(ref *RecordRef) (*segment, error) {
	if ref == nil || ref.segment < 0 || ref.segment >= len(s.segments) {
		return nil, errors.New(msg.MSG_CORRUPTED_RECORD)
	}

	seg := s.segments[ref.segment]
	seg.acquire()
	return seg, nil
}

/**
* RebuildIndexes
* @return error
**/
func (s *FileStore) rebuildIndexes() error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	s.index = make(map[string]*RecordRef)
	for i := range s.segments {
		if err := s.rebuildIndex(i); err != nil {
			return err
		}
	}

	return nil
}

/**
* Close
* @return error
**/
func (s *FileStore) Close() error {
	s.compactWg.Wait()

	// compactMu also waits for a Compact() started by Prune.
	s.compactMu.Lock()
	defer s.compactMu.Unlock()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	// Flush every segment and retire it: files nobody is reading close now, the
	// rest close when their last in-flight reader (Get/ForEach) releases them.
	var closeErr error
	for _, seg := range s.segments {
		if err := seg.Seal(); err != nil && closeErr == nil {
			closeErr = err
		}
		seg.retire()
	}

	return closeErr
}

/**
* Empty
* @return error
**/
func (s *FileStore) Empty() error {
	err := s.Close()
	if err != nil {
		return err
	}

	s.indexMu.Lock()
	s.index = make(map[string]*RecordRef)
	s.indexMu.Unlock()
	s.WAL.set(0)
	s.TombStones.set(0)
	s.Size.set(0)

	defer os.RemoveAll(s.Path)
	return nil
}

/**
* Sync
* @param id string, ref *RecordRef, ownerId string
**/
func (s *FileStore) Sync(id string, ref *RecordRef, ownerId string) {
	if s.ID == ownerId {
		return
	}
	s.setIndex(id, ref)
}

/**
* checkWrite: Validates that the store accepts writes and that id is usable
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
* put: Appends data for id to the log and points the index at it. Caller must
* hold writeMu, so the caller's existence check, the log append and the index
* update happen as one step and apply to the index in the same order as the log.
* @param id string, data []byte
* @return bool (true if id already existed), error
**/
func (s *FileStore) put(id string, data []byte) (bool, error) {
	ref, err := s.appendRecordLocked(0, id, data, Active)
	if err != nil {
		return false, err
	}

	exists := s.setIndex(id, ref)
	if exists {
		s.TombStones.inc()
	}

	return exists, nil
}

/**
* Insert: Stores data under id only if id does not exist yet
* @param id string, data []byte
* @return bool (true if inserted), error
**/
func (s *FileStore) Insert(id string, data []byte) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	s.writeMu.Lock()
	if _, exists := s.getIndex(id); exists {
		s.writeMu.Unlock()
		return false, nil
	}

	_, err := s.put(id, data)
	s.writeMu.Unlock()
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* Update: Replaces the data of id only if id exists and data is different
* @param id string, data []byte
* @return bool (true if updated), error
**/
func (s *FileStore) Update(id string, data []byte) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	s.writeMu.Lock()
	ref, exists := s.getIndex(id)
	if !exists {
		s.writeMu.Unlock()
		return false, nil
	}

	old, err := s.readPinned(ref, id)
	if err != nil {
		s.writeMu.Unlock()
		return false, err
	}
	if bytes.Equal(old, data) {
		s.writeMu.Unlock()
		return false, nil
	}

	_, err = s.put(id, data)
	s.writeMu.Unlock()
	if err != nil {
		return false, err
	}

	s.compactIfNeeded()

	return true, nil
}

/**
* Delete: Removes id only if it exists
* @param id string
* @return bool (true if deleted), error
**/
func (s *FileStore) Delete(id string) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	// The existence check, the tombstone append and the index removal run under
	// writeMu so no Insert/Update on the same id can slip in between them.
	s.writeMu.Lock()
	if _, exists := s.getIndex(id); !exists {
		s.writeMu.Unlock()
		return false, nil
	}

	if _, err := s.appendRecordLocked(0, id, nil, Deleted); err != nil {
		s.writeMu.Unlock()
		return false, err
	}

	s.deleteIndex(id)
	s.TombStones.inc()
	s.writeMu.Unlock()

	if s.isDebug {
		logs.Debug("deleted:", s.Path, ":total:", s.countIndex(), ":ID:", id)
	}

	s.compactIfNeeded()

	return true, nil
}

/**
* IsExist
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
* readPinned: Reads the record of id at ref without holding indexMu during the
* disk read; the segment is pinned so Compact() cannot close it meanwhile.
* An empty id reads the id from disk.
* @param ref *RecordRef, id string
* @return []byte, error
**/
func (s *FileStore) readPinned(ref *RecordRef, id string) ([]byte, error) {
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
* Read: Reads the record at ref
* @param ref *RecordRef
* @return []byte, error
**/
func (s *FileStore) Read(ref *RecordRef) ([]byte, error) {
	return s.readPinned(ref, "")
}

/**
* ReadHeader: Reads the header of the record at ref
* @param ref *RecordRef
* @return recordHeader, error
**/
func (s *FileStore) ReadHeader(ref *RecordRef) (recordHeader, error) {
	s.indexMu.RLock()
	seg, err := s.pinLocked(ref)
	s.indexMu.RUnlock()
	if err != nil {
		return recordHeader{}, err
	}
	defer seg.release()

	return seg.ReadHeader(ref)
}

/**
* Get: Returns the data stored under id
* @param id string
* @return []byte, bool (true if id exists), error
**/
func (s *FileStore) Get(id string) ([]byte, bool, error) {
	// Lookup and pin under one RLock so the ref and its segment belong to the
	// same layout; the disk read runs after releasing it.
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
* forEachItem: One record selected by ForEach, with its segment pinned
**/
type forEachItem struct {
	id  string
	ref *RecordRef
	seg *segment
}

/**
* forEachResult: Result of reading one forEachItem
**/
type forEachResult struct {
	data []byte
	err  error
}

/**
* ForEach: Calls fn for each record, in key order (asc/desc), windowed by offset
* and limit (limit <= 0 means no limit). Disk reads run concurrently on up to
* runtime.NumCPU() goroutines, but fn is always called sequentially, in order,
* from the calling goroutine, so it needs no locking of its own and returning
* false stops exactly after the current record.
* The set of records is fixed when the call starts: no lock is held while
* reading or while fn runs, so writes are not blocked and fn may call back into
* this FileStore (a concurrent Compact() defers closing files until the scan ends).
* @param fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *FileStore) ForEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
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
			seg.release() // one pin per segment is enough
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
	// Workers read items in parallel; each result goes to slot i%window. The
	// consumer (this goroutine) takes slots in order 0..n-1 and calls fn. sem
	// caps dispatched-but-unconsumed items at window, so memory stays bounded
	// and a slot is never reused before its previous result was consumed.
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

	// Stop the producer and wait for every goroutine before the deferred
	// release() of the pinned segments runs (also if fn panics).
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
* Keys: Returns the ids sorted asc/desc, windowed by offset and limit, without
* reading any record. Cheaper than ForEach when only ids are needed.
* @param asc bool, offset int, limit int
* @return []string
**/
func (s *FileStore) Keys(asc bool, offset, limit int) []string {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	return s.sortedKeysLocked(asc, offset, limit)
}

/**
* Prune
* @return error
**/
func (s *FileStore) Prune() error {
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
* Open
* @param pathData, pathWald, name string, isDebug bool, mode Mode
* @return *FileStore, error
**/
func Open(pathData, pathWald, name string, mode Mode) (*FileStore, error) {
	maxSegmentMG := envar.GetInt64("RELSEG_SIZE", 128)
	maxSegmentMG = maxSegmentMG * 1024 * 1024
	minThreshold := envar.GetInt("MIN_THRESHOLD_COMPACT", 1000)
	name = Normalize(name)
	fs := &FileStore{
		Name:                name,
		Path:                filepath.Join(pathData, "segments", name),
		PathSnapshot:        filepath.Join(pathWald, "snapshot", name),
		PathCompact:         filepath.Join(pathWald, "compact", name),
		MaxSegment:          maxSegmentMG,
		MinThresholdCompact: minThreshold,
		mode:                mode,
	}

	syncOnWrite := envar.GetBool("SYNC_ON_WRITE", true)
	fs.index = make(map[string]*RecordRef)
	fs.SyncOnWrite = syncOnWrite

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

	// If snapshot loaded: buildIndex covers only the last segment (snapshot has the rest).
	// If snapshot absent or corrupt: rebuildIndexes scans all segments from scratch.
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
