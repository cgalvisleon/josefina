package store

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

const (
	packageName       = "store"
	maxIdLen          = 65535
	fixedHeaderSize   = 19  // LSN(8) + DataLen(4) + CRC(4) + IDLen(2) + Status(1)
	workerThreshold   = 128 // below this, sequential is faster than goroutine pool
	workerRecordRatio = 128 // one worker per this many records
)

/**
* optimalWorkers: Returns the worker count for a parallel ForEach.
* For I/O-bound segment reads: scales up to 2×GOMAXPROCS capped by one
* worker per workerRecordRatio records. Returns 1 for small sets, which
* triggers the sequential fast path (no goroutines or channels).
* @param total int
* @return int
**/
func optimalWorkers(total int) int {
	if total <= workerThreshold {
		return 1
	}
	max := runtime.GOMAXPROCS(0) * 2
	w := (total + workerRecordRatio - 1) / workerRecordRatio
	if w > max {
		w = max
	}
	return w
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

type SetIndexFn func(*FileStore, string, *RecordRef)
type Putfn func(*FileStore, string, []byte)
type Deletefn func(*FileStore, string)

type FileStore struct {
	ID                  string                `json:"id"`
	Name                string                `json:"name"`
	Path                string                `json:"path"`
	WAL                 uint64                `json:"wal"`
	TombStones          int                   `json:"tomb_stones"`
	PathSegments        string                `json:"path_segments"`
	PathSnapshot        string                `json:"path_snapshot"`
	PathCompact         string                `json:"path_compact"`
	MaxSegment          int64                 `json:"max_segment"`
	SyncOnWrite         bool                  `json:"sync_on_write"`
	Size                int64                 `json:"size"`
	MinThresholdCompact int                   `json:"min_threshold_compact"`
	writeMu             sync.Mutex            `json:"-"` // SOLO WAL append
	indexMu             sync.RWMutex          `json:"-"` // índice en memoria
	segments            []*segment            `json:"-"` // segmentos de datos
	active              *segment              `json:"-"` // segmento activo para escritura
	index               map[string]*RecordRef `json:"-"` // índice en memoria
	keys                []string              `json:"-"` // claves en memoria
	mode                Mode                  `json:"-"` // modo de operación
	onSetIndex          []SetIndexFn          `json:"-"` // función de actualización de índice
	onPut               []Putfn               `json:"-"` // función de escritura
	onDelete            []Deletefn            `json:"-"` // función de eliminación
	compacting          int32                 `json:"-"` // 0 = idle, 1 = running
	compactWg           sync.WaitGroup        `json:"-"` // espera que termine la goroutine de compaction
	isDebug             bool                  `json:"-"`
}

/**
* Serialize
* @return []byte, error
**/
func (s *FileStore) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *FileStore) ToJson() et.Json {
	bt, err := s.serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
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
	s.indexMu.RLock()
	n := len(s.index)
	s.indexMu.RUnlock()
	return n
}

/**
* OnIndex
* @param fn SetIndexFn
**/
func (s *FileStore) OnIndex(fn SetIndexFn) {
	s.onSetIndex = append(s.onSetIndex, fn)
}

/**
* OnPut
* @param fn Putfn
**/
func (s *FileStore) OnPut(fn Putfn) {
	s.onPut = append(s.onPut, fn)
}

/**
* OnDelete
* @param fn Deletefn
**/
func (s *FileStore) OnDelete(fn Deletefn) {
	s.onDelete = append(s.onDelete, fn)
}

/**
* loadSegments
* @return error
**/
func (s *FileStore) loadSegments() error {
	files, err := os.ReadDir(s.PathSegments)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, f := range files {
		name := f.Name()
		path := filepath.Join(s.PathSegments, name)
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
		s.Size += size
		if s.isDebug {
			logs.Log(packageName, "load:segments:", s.Path, ":", s.Name, ":", seg.ToString())
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
	name := fmt.Sprintf("segment-%06d.dat", len(s.segments)+1)
	path := filepath.Join(s.PathSegments, name)

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	seg := newSegment(fd, 0, name)
	s.segments = append(s.segments, seg)
	if s.active != nil {
		err := s.active.Close()
		if err != nil {
			return err
		}
	}
	s.active = seg
	if s.isDebug {
		logs.Log(packageName, "new:segment:", s.Path, ":", s.Name, ":", seg.ToString())
	}

	return nil
}

/**
* appendRecordAt writes a record with a caller-supplied LSN (replication path).
* It does not check ReadOnly and does not auto-increment s.WAL.
* If lsn > s.WAL the local counter is advanced to stay in sync with the leader.
* @param lsn uint64, id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *FileStore) appendRecordAt(lsn uint64, id string, data []byte, status byte) (*RecordRef, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	recordSize := int64(fixedHeaderSize) + int64(len(id)) + int64(len(data))
	currentSize := s.active.size
	totalSize := currentSize + recordSize
	s.Size += recordSize
	if totalSize > s.MaxSegment {
		if err := s.newSegment(); err != nil {
			return nil, err
		}
		if err := s.CreateSnapshot(); err != nil {
			return nil, err
		}
	}

	ref, err := s.active.WriteRecord(lsn, id, data, status)
	if err != nil {
		return nil, err
	}
	ref.segment = len(s.segments) - 1
	if s.SyncOnWrite {
		if err := s.active.Sync(); err != nil {
			return nil, err
		}
	}

	if lsn > s.WAL {
		s.WAL = lsn
	}

	return ref, nil
}

/**
* appendRecord
* @param id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *FileStore) appendRecord(id string, data []byte, status byte) (*RecordRef, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	recordSize := int64(fixedHeaderSize) + int64(len(id)) + int64(len(data))
	currentSize := s.active.size
	totalSize := currentSize + recordSize
	s.Size += recordSize
	if totalSize > s.MaxSegment {
		if err := s.newSegment(); err != nil {
			return nil, err
		}

		if err := s.CreateSnapshot(); err != nil {
			return nil, err
		}
	}

	s.WAL++
	ref, err := s.active.WriteRecord(s.WAL, id, data, status)
	if err != nil {
		return nil, err
	}
	ref.segment = len(s.segments) - 1
	if s.SyncOnWrite {
		if err := s.active.Sync(); err != nil {
			return nil, err
		}
	}

	n := len(s.index)
	threshold := int(float64(n) * 0.1) // 10% del tamaño del índice
	if threshold < s.MinThresholdCompact {
		threshold = s.MinThresholdCompact
	}
	if s.TombStones > threshold {
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

	return ref, nil
}

/**
* setIndex
* @param id string, segIndex int, offset int64, dataLen uint32
**/
func (s *FileStore) setIndex(id string, segIndex int, offset int64, dataLen uint32) {
	ref := &RecordRef{
		segment: segIndex,
		offset:  offset,
		length:  dataLen,
	}
	if _, exists := s.index[id]; !exists {
		s.keys = append(s.keys, id)
	}
	s.index[id] = ref
	for _, fn := range s.onSetIndex {
		fn(s, id, ref)
	}
}

/**
* putIndex
* @param id string, ref *RecordRef
**/
func (s *FileStore) putIndex(id string, ref *RecordRef) {
	if _, exists := s.index[id]; !exists {
		s.keys = append(s.keys, id)
	}
	s.index[id] = ref
	if s.isDebug {
		logs.Debug("put:", s.Path, ":", s.Name, ":lsn:", s.WAL, ":ID:", id, ":ref:", ref.ToString())
	}
}

/**
* deleteIndex
* @param id string
**/
func (s *FileStore) deleteIndex(id string) {
	delete(s.index, id)
	idx := slices.Index(s.keys, id)
	if idx != -1 {
		s.keys = append(s.keys[:idx], s.keys[idx+1:]...)
	}
}

/**
* rebuildIndex
* @param segIndex int
* @return error
**/
func (s *FileStore) rebuildIndex(segIndex int) error {
	if len(s.index) == 0 {
		s.index = make(map[string]*RecordRef)
		s.keys = make([]string, 0)
	}

	seg := s.segments[segIndex]
	offset := int64(0)
	for {
		// Layout: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1]
		fixed := make([]byte, fixedHeaderSize)
		n, err := seg.ReadAt(fixed, offset)
		if err != nil {
			if errors.Is(err, io.EOF) || n < len(fixed) {
				break
			}
			return err
		}

		lsn := binary.BigEndian.Uint64(fixed[0:8])
		dataLen := getUint32(fixed[8:12])
		crcStored := getUint32(fixed[12:16])
		idLen := getUint16(fixed[16:18])

		if idLen == 0 || idLen > maxIdLen {
			break // corrupción → paro seguro
		}

		// Leer ID
		idBytes := make([]byte, idLen)
		if _, err := seg.ReadAt(idBytes, offset+18); err != nil {
			break
		}
		id := string(idBytes)

		// Leer status
		statusByte := make([]byte, 1)
		if _, err := seg.ReadAt(statusByte, offset+18+int64(idLen)); err != nil {
			break
		}
		status := statusByte[0]

		// Leer payload
		data := make([]byte, dataLen)
		if dataLen > 0 {
			if _, err := seg.ReadAt(data, offset+18+int64(idLen)+1); err != nil {
				break
			}
			if checksum(data) != crcStored {
				break
			}
		}

		if status == Active {
			s.setIndex(id, segIndex, offset, dataLen)
		} else if status == Deleted {
			s.deleteIndex(id)
		}

		// Restaurar WAL al LSN más alto visto en disco
		if lsn > s.WAL {
			s.WAL = lsn
		}

		offset += int64(fixedHeaderSize) + int64(idLen) + int64(dataLen)
	}

	return nil
}

/**
* buildIndex
* @return error
**/
func (s *FileStore) buildIndex() error {
	idx := len(s.segments) - 1
	return s.rebuildIndex(idx)
}

/**
* getRecords
* @param asc bool, offset int, limit int
* @return map[string]*RecordRef, []string
**/
func (s *FileStore) getRecords(asc bool, offset, limit int) (map[string]*RecordRef, []string) {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	n := len(s.index)
	keys := make([]string, 0)
	indexResult := make(map[string]*RecordRef, 0)
	if offset >= n {
		return indexResult, keys
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

	return indexResult, keys
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

	s.indexMu.RLock()
	active := s.active
	s.indexMu.RUnlock()

	if active == nil {
		return nil
	}
	return active.Close()
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

	s.index = make(map[string]*RecordRef)
	s.WAL = 0
	s.TombStones = 0

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
	s.indexMu.Lock()
	s.putIndex(id, ref)
	s.indexMu.Unlock()
}

/**
* Put
* @param id string, value any
* @return error, bool
**/
func (s *FileStore) Put(id string, value any) (bool, error) {
	if s.mode == ReadOnly {
		return false, errors.New(msg.MSG_STORE_IS_READ_ONLY)
	}

	if id == "" {
		return false, errors.New(msg.MSG_ID_IS_REQUIRED)
	}

	bt, ok := value.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(value)
		if err != nil {
			return false, err
		}
	}

	ref, err := s.appendRecord(id, bt, Active)
	if err != nil {
		return false, err
	}

	s.indexMu.Lock()
	_, exists := s.index[id]
	if exists {
		s.TombStones++
	}
	s.putIndex(id, ref)
	s.indexMu.Unlock()

	for _, fn := range s.onPut {
		fn(s, id, bt)
	}

	return exists, nil
}

/**
* Read
* @param ref *RecordRef, dest any
* @return bool, error
**/
func (s *FileStore) Read(ref *RecordRef, dest any) (bool, error) {
	seg := s.segments[ref.segment]
	err := seg.Read(ref, dest)
	if err != nil {
		return false, err
	}
	return true, nil
}

/**
* ReadHeader
* @param ref *RecordRef
* @return recordHeader, error
**/
func (s *FileStore) ReadHeader(ref *RecordRef) (recordHeader, error) {
	seg := s.segments[ref.segment]
	return seg.ReadHeader(ref)
}

/**
* Get
* @param id string, dest any
* @return bool, error
**/
func (s *FileStore) Get(id string, dest any) (bool, error) {
	s.indexMu.RLock()
	ref, existed := s.index[id]
	s.indexMu.RUnlock()

	if !existed {
		return false, nil
	}

	return s.Read(ref, dest)
}

/**
* Delete
* @param id string
* @return bool, error
**/
func (s *FileStore) Delete(id string) (bool, error) {
	if s.mode == ReadOnly {
		return false, errors.New(msg.MSG_STORE_IS_READ_ONLY)
	}

	s.indexMu.RLock()
	_, exists := s.index[id]
	s.indexMu.RUnlock()

	if !exists {
		return false, nil
	}

	if _, err := s.appendRecord(id, nil, Deleted); err != nil {
		return false, logs.Error(err)
	}

	s.indexMu.Lock()
	s.TombStones++
	s.deleteIndex(id)
	s.indexMu.Unlock()

	for _, fn := range s.onDelete {
		fn(s, id)
	}

	if s.isDebug {
		i := len(s.index)
		logs.Debug("deleted:", s.Path, ":", s.Name, ":total:", i, ":ID:", id)
	}

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

	s.indexMu.RLock()
	_, existed := s.index[id]
	s.indexMu.RUnlock()

	return existed
}

/**
* ForEach
* @param fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *FileStore) ForEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	index, keys := s.getRecords(asc, offset, limit)
	s.indexMu.RLock()
	segs := s.segments
	s.indexMu.RUnlock()

	workers := optimalWorkers(len(keys))

	// Sequential fast path: avoids goroutine and channel overhead for small sets.
	if workers == 1 {
		for _, id := range keys {
			ref, ok := index[id]
			if !ok {
				continue
			}
			data, err := segs[ref.segment].read(ref)
			if err != nil {
				return err
			}
			cont, err := fn(id, data)
			if err != nil {
				return err
			}
			if !cont {
				return nil
			}
		}
		return nil
	}

	// Parallel path: worker pool for I/O-bound segment reads.
	bufSize := workers * 4
	if bufSize > len(keys) {
		bufSize = len(keys)
	}
	jobs := make(chan string, bufSize)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	wg.Add(workers)
	for w := 0; w < workers; w++ {
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

producerLoop:
	for _, id := range keys {
		select {
		case <-ctx.Done():
			break producerLoop
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
	_, keys := s.getRecords(asc, offset, limit)
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
* @param path, name string, isDebug bool, mode Mode
* @return *FileStore, error
**/
func Open(path, name string, mode Mode) (*FileStore, error) {
	maxSegmentMG := envar.GetInt64("RELSEG_SIZE", 128)
	maxSegmentMG = maxSegmentMG * 1024 * 1024
	minThreshold := envar.GetInt("MIN_THRESHOLD_COMPACT", 1000)
	name = utility.Normalize(name)
	fs := &FileStore{
		Name:                name,
		Path:                filepath.Join(path, name),
		PathSegments:        filepath.Join(path, name, "segments"),
		PathSnapshot:        filepath.Join(path, name, "snapshot"),
		PathCompact:         filepath.Join(path, name, "compact"),
		MaxSegment:          maxSegmentMG,
		MinThresholdCompact: minThreshold,
		mode:                mode,
		onSetIndex:          make([]SetIndexFn, 0),
		onPut:               make([]Putfn, 0),
		onDelete:            make([]Deletefn, 0),
	}

	syncOnWrite := envar.GetBool("SYNC_ON_WRITE", true)
	fs.index = make(map[string]*RecordRef)
	fs.keys = make([]string, 0)
	fs.SyncOnWrite = syncOnWrite

	if mode == ReadOnly {
		if _, err := os.Stat(fs.PathSegments); os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: store not found at %s", errors.New(msg.MSG_STORE_NOT_FOUND), fs.PathSegments)
		}
	} else {
		if err := os.MkdirAll(fs.PathSegments, 0755); err != nil {
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
