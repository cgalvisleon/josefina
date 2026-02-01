package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

const (
	packageName     = "store"
	maxIdLen        = 65535
	fixedHeaderSize = 11
)

/**
* newRecordHeaderAt
* @param id string, data []byte, status byte
* @return recordHeader, []byte, error
**/
func newRecordHeaderAt(id string, data []byte, status byte) (recordHeader, []byte, error) {
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
		DataLen: uint32(dataLen),
		CRC:     checksum(data),
		IDLen:   uint16(idLen),
		Status:  status,
	}

	header := make([]byte, headerLen)
	putUint32(header[0:4], result.DataLen)
	putUint32(header[4:8], result.CRC)
	putUint16(header[8:10], result.IDLen)
	copy(header[10:10+idLen], idBytes)
	header[10+idLen] = status

	return result, header, nil
}

type mode int

const (
	modeRead mode = iota
	modeWrite
)

type Putfn func(string, []byte)
type Deletefn func(string)

type FileStore struct {
	Name         string                `json:"name"`
	Path         string                `json:"path"`
	WAL          uint64                `json:"wal"` // Write-ahead log counter
	TombStones   int                   `json:"tomb_stones"`
	PathSegments string                `json:"path_segments"`
	PathSnapshot string                `json:"path_snapshot"`
	PathCompact  string                `json:"path_compact"`
	MaxSegment   int64                 `json:"max_segment"`
	SyncOnWrite  bool                  `json:"sync_on_write"`
	Size         int64                 `json:"size"`
	isDebug      bool                  `json:"-"`
	writeMu      sync.Mutex            `json:"-"` // SOLO WAL append
	indexMu      sync.RWMutex          `json:"-"` // índice en memoria
	segments     []*segment            `json:"-"` // segmentos de datos
	active       *segment              `json:"-"` // segmento activo para escritura
	index        map[string]*RecordRef `json:"-"` // índice en memoria
	keys         []string              `json:"-"` // claves en memoria
	mode         mode                  `json:"-"` // modo de operación
	onPut        []Putfn               `json:"-"` // función de escritura
	onDelete     []Deletefn            `json:"-"` // función de eliminación
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
* Count
* @return int
**/
func (s *FileStore) Count() int {
	return len(s.index)
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

		fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return err
		}

		size := st.Size()
		if _, err := fd.Seek(size, io.SeekStart); err != nil {
			return err
		}

		seg := newSegment(fd, size, name)
		s.segments = append(s.segments, seg)
		s.Size += size
		if s.isDebug {
			logs.Log(packageName, "load:segments:", s.Path, ":", s.Name, ":", seg.ToString())
		}
	}

	if len(s.segments) == 0 {
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
* appendRecord
* @param id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *FileStore) appendRecord(id string, data []byte, status byte) (*RecordRef, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	recordSize := int64(len(id)) + int64(len(data)) + 11
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

	ref, err := s.active.WriteRecord(id, data, status)
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
	if s.TombStones > threshold {
		go s.Compact()
	}

	return ref, nil
}

/**
* setIndex
* @param id string, segIndex int, offset int64, dataLen uint32
* @return error
**/
func (s *FileStore) setIndex(id string, segIndex int, offset int64, dataLen uint32) error {
	ref := &RecordRef{
		segment: segIndex,
		offset:  offset,
		length:  dataLen,
	}
	s.index[id] = ref
	s.keys = append(s.keys, id)
	return nil
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
		// Leer header mínimo
		fixed := make([]byte, 11)
		n, err := seg.ReadAt(fixed, offset)
		if err != nil {
			if errors.Is(err, io.EOF) || n < len(fixed) {
				break
			}
			return err
		}

		dataLen := getUint32(fixed[0:4])
		crcStored := getUint32(fixed[4:8])
		idLen := getUint16(fixed[8:10])

		if idLen == 0 || idLen > maxIdLen {
			break // corrupción → paro seguro
		}

		// Leer ID
		idBytes := make([]byte, idLen)
		if _, err := seg.ReadAt(idBytes, offset+10); err != nil {
			break
		}
		id := string(idBytes)

		// Leer status
		statusLen := int64(1)
		statusByte := make([]byte, statusLen)
		if _, err := seg.ReadAt(statusByte, offset+10+int64(idLen)); err != nil {
			break
		}
		status := statusByte[0]

		// Leer payload
		data := make([]byte, dataLen)
		if dataLen > 0 {
			if _, err := seg.ReadAt(data, offset+10+int64(idLen)+statusLen); err != nil {
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

		offset += int64(11) + int64(idLen) + int64(dataLen)
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
	n := len(s.index)
	keys := make([]string, 0)
	indexResult := make(map[string]*RecordRef, 0)
	if offset >= n {
		return indexResult, keys
	}

	if limit <= 0 {
		limit = n
	}

	s.indexMu.RLock()
	nKeys := len(s.keys)
	if n != nKeys {
		s.keys = make([]string, 0)
		for k := range s.index {
			s.keys = append(s.keys, k)
		}
		sort.Strings(keys)
	}

	if asc {
		sort.Strings(s.keys)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(s.keys)))
	}
	i := 0
	for {
		k := s.keys[offset]
		v := s.index[k]
		indexResult[k] = v
		keys = append(keys, k)
		offset++
		i++
		if i >= limit {
			break
		}
	}
	s.indexMu.RUnlock()

	return indexResult, keys
}

/**
* Close
* @return error
**/
func (s *FileStore) Close() error {
	if s.active == nil {
		return nil
	}

	err := s.active.Close()
	if err != nil {
		return err
	}

	return nil
}

/**
* Keys
* @param asc bool, offset int, limit int
* @return []string
**/
func (s *FileStore) Keys(asc bool, offset, limit int) []string {
	n := len(s.keys)
	result := make([]string, 0)
	if offset >= n {
		return result
	}

	if limit <= 0 {
		limit = n
	}

	if asc {
		sort.Strings(s.keys)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(s.keys)))
	}
	i := 0
	for {
		k := s.keys[offset]
		result = append(result, k)
		offset++
		i++
		if i >= limit {
			break
		}
	}

	return result
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
* OnPut
* @param fn func(string, []byte)
**/
func (s *FileStore) OnPut(fn func(string, []byte)) {
	s.onPut = append(s.onPut, fn)
}

/**
* OnDelete
* @param fn func(string)
**/
func (s *FileStore) OnDelete(fn func(string)) {
	s.onDelete = append(s.onDelete, fn)
}

/**
* Put
* @param id string, value any
* @return error
**/
func (s *FileStore) Put(id string, value any) error {
	if id == "" {
		return errors.New(msg.MSG_ID_IS_REQUIRED)
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ref, err := s.appendRecord(id, data, Active)
	if err != nil {
		return err
	}

	s.indexMu.Lock()
	_, exists := s.index[id]
	if exists {
		s.TombStones++
	} else {
		s.WAL++
	}
	s.index[id] = ref
	s.indexMu.Unlock()

	for _, fn := range s.onPut {
		fn(id, data)
	}

	if s.isDebug {
		i := len(s.index)
		logs.Debug("put:", s.Path, ":", s.Name, ":total:", i, ":ID:", id, ":ref:", ref.ToString())
	}

	return nil
}

/**
* Delete
* @param id string
* @return bool, error
**/
func (s *FileStore) Delete(id string) (bool, error) {
	s.indexMu.RLock()
	_, exists := s.index[id]
	if exists {
		s.TombStones++
	}
	s.indexMu.RUnlock()

	if !exists {
		return false, nil
	}

	if _, err := s.appendRecord(id, nil, Deleted); err != nil {
		return false, logs.Error(err)
	}

	s.indexMu.Lock()
	s.deleteIndex(id)
	s.indexMu.Unlock()

	for _, fn := range s.onDelete {
		fn(id)
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
	s.indexMu.RLock()
	_, existed := s.index[id]
	s.indexMu.RUnlock()

	return existed
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

	seg := s.segments[ref.segment]
	err := seg.Read(ref, dest)
	if err != nil {
		return existed, err
	}

	return existed, nil
}

/**
* Iterate
* @param fn func(id string, data []byte) bool, asc bool, offset, limit, workers int
* @return error
**/
func (s *FileStore) Iterate(fn func(id string, data []byte) (bool, error), asc bool, offset, limit, workers int) error {
	// 1. Seleccionar IDs
	index, keys := s.getRecords(asc, offset, limit)

	if workers <= 0 {
		workers = 1
	}

	// 2) Worker pool
	jobs := make(chan string, 1024)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		mErr    error
		total   int64
	)

	setErr := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			mErr = err
			cancel() // cortar todos
		})
	}

	// 3) Lanzar workers
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
						// si esto puede pasar, es inconsistencia del índice
						// define si debe ser error o skip
						continue
					}

					seg := s.segments[ref.segment]

					data, err := seg.read(ref)
					if err != nil {
						setErr(err)
						return
					}

					cont, err := fn(id, data)
					if err != nil {
						setErr(err)
						return
					}

					atomic.AddInt64(&total, 1)

					if !cont {
						// detener todo si el callback pide parar
						cancel()
						return
					}
				}
			}
		}()
	}

	// 4) Enviar jobs (producer)
	for _, id := range keys {
		select {
		case <-ctx.Done():
			break
		case jobs <- id:
		}
	}

	close(jobs)
	wg.Wait()

	return mErr
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
* Empty
* @return error
**/
func (s *FileStore) Empty() error {
	s.index = make(map[string]*RecordRef)
	s.WAL = 0
	s.TombStones = 0

	return nil
}

/**
* open
* @param path, name string,
* @return *FileStore, error
**/
func open(path, name string, isDebug bool, mode mode) (*FileStore, error) {
	maxSegmentMG := envar.GetInt64("RELSEG_SIZE", 128)
	maxSegmentMG = maxSegmentMG * 1024 * 1024
	name = utility.Normalize(name)
	fs := &FileStore{
		Name:         name,
		Path:         filepath.Join(path),
		PathSegments: filepath.Join(path, name, "segments"),
		PathSnapshot: filepath.Join(path, name, "snapshot"),
		PathCompact:  filepath.Join(path, name, "compact"),
		MaxSegment:   maxSegmentMG,
		isDebug:      isDebug,
		mode:         mode,
		onPut:        make([]Putfn, 0),
		onDelete:     make([]Deletefn, 0),
	}

	syncOnWrite := envar.GetBool("SYNC_ON_WRITE", true)
	fs.index = make(map[string]*RecordRef)
	fs.keys = make([]string, 0)
	fs.SyncOnWrite = syncOnWrite

	if err := os.MkdirAll(fs.PathSegments, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fs.PathSnapshot, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fs.PathCompact, 0755); err != nil {
		return nil, err
	}

	if err := fs.loadSegments(); err != nil {
		return nil, fmt.Errorf("loadSegments: %w", err)
	}
	if err := fs.tryLoadSnapshot(); err != nil {
		return nil, fmt.Errorf("tryLoadSnapshot: %w", err)
	}
	if err := fs.buildIndex(); err != nil {
		return nil, fmt.Errorf("buildIndex: %w", err)
	}

	return fs, nil
}

/**
* Open
* @param path, name string,
* @return *FileStore, error
**/
func Open(path, name string, isDebug bool) (*FileStore, error) {
	return open(path, name, isDebug, modeWrite)
}

/**
* ReadOnly
* @param path, name string,
* @return *FileStore, error
**/
func ReadOnly(path, name string, isDebug bool) (*FileStore, error) {
	return open(path, name, isDebug, modeRead)
}
