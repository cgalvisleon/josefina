package store

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	keys                []string              `json:"-"` // claves en memoria
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
	if !exists {
		s.keys = append(s.keys, id)
	}
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
	idx := slices.Index(s.keys, id)
	if idx != -1 {
		s.keys = append(s.keys[:idx], s.keys[idx+1:]...)
	}
	return true
}

/**
* rebuildIndex: Replays a segment into the index. Caller must hold indexMu.
* @param segIndex int
* @return error
**/
func (s *FileStore) rebuildIndex(segIndex int) error {
	if len(s.index) == 0 {
		s.index = make(map[string]*RecordRef)
		s.keys = make([]string, 0)
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
* getRecords
* @param asc bool, offset int, limit int
* @return map[string]*RecordRef, []string, []*segment
**/
func (s *FileStore) getRecords(asc bool, offset, limit int) (map[string]*RecordRef, []string, []*segment) {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	return s.getRecordsLocked(asc, offset, limit)
}

/**
* getRecordsLocked: Same as getRecords but assumes the caller already holds indexMu
* (read or write). ForEach uses this to keep the lock held across the whole scan,
* including the segment reads, so Compact() cannot close/swap segments underneath it.
* @param asc bool, offset int, limit int
* @return map[string]*RecordRef, []string, []*segment
**/
func (s *FileStore) getRecordsLocked(asc bool, offset, limit int) (map[string]*RecordRef, []string, []*segment) {
	segs := s.segments

	n := len(s.index)
	keys := make([]string, 0)
	indexResult := make(map[string]*RecordRef, 0)
	if offset >= n {
		return indexResult, keys, segs
	}

	if limit <= 0 {
		limit = n
	}

	keysCopy := make([]string, len(s.keys))
	copy(keysCopy, s.keys)

	if asc {
		sort.Strings(keysCopy)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(keysCopy)))
	}

	i := 0
	for {
		if offset >= len(keysCopy) {
			break
		}
		k := keysCopy[offset]
		v, ok := s.index[k]
		if ok {
			indexResult[k] = v
			keys = append(keys, k)
		}
		offset++
		i++
		if i >= limit {
			break
		}
	}

	return indexResult, keys, segs
}

/**
* RebuildIndexes
* @return error
**/
func (s *FileStore) rebuildIndexes() error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	s.index = make(map[string]*RecordRef)
	s.keys = make([]string, 0)
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

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	// Close every segment, not just the active one: sealed segments keep their
	// file open for reads, and segments loaded read-write run a writer goroutine.
	var closeErr error
	for _, seg := range s.segments {
		if err := seg.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
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
	s.keys = make([]string, 0)
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

	old, err := s.Read(ref)
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
* readSegment: Reads ref from its segment. Caller must hold indexMu (read or write)
* for the whole call — Compact() takes indexMu.Lock() before closing old segments,
* so holding the lock here prevents reading from a segment mid-close/after-swap.
* @param ref *RecordRef
* @return []byte, error
**/
func (s *FileStore) readSegment(ref *RecordRef) ([]byte, error) {
	seg := s.segments[ref.segment]
	return seg.Read(ref)
}

/**
* Read
* @param ref *RecordRef
* @return bool, error
**/
func (s *FileStore) Read(ref *RecordRef) ([]byte, error) {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	result, err := s.readSegment(ref)
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
* ReadHeader
* @param ref *RecordRef
* @return recordHeader, error
**/
func (s *FileStore) ReadHeader(ref *RecordRef) (recordHeader, error) {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	seg := s.segments[ref.segment]
	return seg.ReadHeader(ref)
}

/**
* Get: Returns the data stored under id
* @param id string
* @return []byte, bool (true if id exists), error
**/
func (s *FileStore) Get(id string) ([]byte, bool, error) {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	ref, exists := s.index[id]
	if !exists {
		return nil, false, nil
	}

	result, err := s.readSegment(ref)
	if err != nil {
		return nil, false, err
	}

	return result, true, nil
}

/**
* ForEach: Iterates over records, holding indexMu.RLock() for the whole scan
* (including fn and the segment reads) so Compact() cannot close/swap segments
* underneath it. fn must not call back into this same FileStore (Get/Put/Delete/
* Read/ForEach), or it can deadlock against a concurrent Compact().
* @param fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *FileStore) ForEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	index, keys, segs := s.getRecordsLocked(asc, offset, limit)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan string)

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		mErr    error
	)

	setErr := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			mErr = err
			cancel()
		})
	}

	workers := runtime.NumCPU()

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {

				case <-ctx.Done():
					return

				case id, ok := <-jobs:
					if !ok {
						return
					}

					ref, ok := index[id]
					if !ok {
						continue
					}

					data, err := segs[ref.segment].read(ref)
					if err != nil {
						setErr(err)
						return
					}

					cont, err := fn(id, data)
					if err != nil {
						setErr(err)
						return
					}

					if !cont {
						cancel()
						return
					}
				}
			}
		}()
	}

Producer:
	for _, id := range keys {

		select {

		case <-ctx.Done():
			break Producer

		case jobs <- id:
		}
	}

	close(jobs)

	wg.Wait()

	return mErr
}

/**
* Keys: Returns a sorted key snapshot without spawning a worker pool.
* Suitable for cursor creation; cheaper than ForEach when only IDs are needed.
* @param asc bool, offset int, limit int
* @return []string
**/
func (s *FileStore) Keys(asc bool, offset, limit int) []string {
	_, keys, _ := s.getRecords(asc, offset, limit)
	return keys
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
	fs.keys = make([]string, 0)
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
