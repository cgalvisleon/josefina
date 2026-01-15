package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
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
	Metrics      map[string]int64      `json:"metrics"`
	Workers      int                   `json:"workers"`
	IsDebug      bool                  `json:"-"`
	writeMu      sync.Mutex            `json:"-"` // SOLO WAL append
	indexMu      sync.RWMutex          `json:"-"` // índice en memoria
	segments     []*segment            `json:"-"` // segmentos de datos
	active       *segment              `json:"-"` // segmento activo para escritura
	index        map[string]*recordRef `json:"-"` // índice en memoria
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
* Close
* @return void
**/
func (s *FileStore) Close() {
	if s.active != nil {
		s.active.Close()
	}
}

/**
* Debug
* Enable debug mode for this store
**/
func (s *FileStore) Debug() {
	s.IsDebug = true
}

/**
* UseMemory
* @return float64
 */
func (s *FileStore) UseMemory() float64 {
	s.indexMu.RLock()
	n := len(s.index)
	s.indexMu.RUnlock()

	result := float64(n) * 96 // estimación simple
	if result < 1024 {
		return 0.001 // evitar valores muy pequeños
	}

	return result / 1024 / 1024 // devolver en MB
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
		logs.Log(packageName, "load:segments:", s.Path, ":", s.Name, ":", seg.ToString())
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
		s.active.Close()
	}
	s.active = seg
	logs.Log(packageName, "new:segment:", s.Path, ":", s.Name, ":", seg.ToString())
	return nil
}

/**
* appendRecord
* @param id string, data []byte, status byte
* @return *recordRef, error
**/
func (s *FileStore) appendRecord(id string, data []byte, status byte) (*recordRef, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	recordSize := int64(len(id)) + int64(len(data)) + 11
	currentSize := s.active.size
	totalSize := currentSize + recordSize
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
	ref := &recordRef{
		segment: segIndex,
		offset:  offset,
		length:  dataLen,
	}
	s.index[id] = ref
	return nil
}

/**
* deleteIndex
* @param id string
**/
func (s *FileStore) deleteIndex(id string) {
	delete(s.index, id)
}

/**
* rebuildIndex
* @param segIndex int
* @return error
**/
func (s *FileStore) rebuildIndex(segIndex int) error {
	if len(s.index) == 0 {
		s.index = make(map[string]*recordRef)
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
* RebuildIndexes
* @return error
**/
func (s *FileStore) RebuildIndexes() error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	tag := "rebuild_indexes"
	s.metricStart(tag)
	defer s.metricEnd(tag, "completed")

	s.index = make(map[string]*recordRef)
	for i := range s.segments {
		if err := s.rebuildIndex(i); err != nil {
			return err
		}
	}

	return nil
}

/**
* Put
* @param id string, value any
* @return string, error
**/
func (s *FileStore) Put(id string, value any) (string, error) {
	atomic.AddUint64(storeCallsMap["put"], 1)
	tag := "put"
	s.metricStart(tag)
	defer s.metricEnd(tag, "completed")

	if id == "" {
		return id, errors.New(msg.MSG_ID_IS_REQUIRED)
	}

	data, err := json.Marshal(value)
	if err != nil {
		return id, err
	}

	ref, err := s.appendRecord(id, data, Active)
	if err != nil {
		return id, err
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

	if s.IsDebug {
		i := len(s.index)
		logs.Log(packageName, "put:", s.Path, ":", s.Name, ":total:", i, ":ID:", id, ":ref:", ref.ToString())
	}

	return id, nil
}

/**
* Delete
* @param id string
* @return bool, error
**/
func (s *FileStore) Delete(id string) (bool, error) {
	atomic.AddUint64(storeCallsMap["delete"], 1)
	tag := "delete"
	s.metricStart(tag)
	defer s.metricEnd(tag, "completed")

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

	if s.IsDebug {
		i := len(s.index)
		logs.Log(packageName, "deleted", s.Path, ":", s.Name, ":total:", i, ":ID:", id)
	}

	return true, nil
}

/**
* Get
* @param id string, dest any
* @return error
**/
func (s *FileStore) Get(id string, dest any) error {
	atomic.AddUint64(storeCallsMap["get"], 1)
	tag := "get"
	s.metricStart(tag)
	defer s.metricEnd(tag, "completed")

	s.indexMu.RLock()
	ref, existed := s.index[id]
	s.indexMu.RUnlock()

	if !existed {
		return errors.New("not found")
	}

	seg := s.segments[ref.segment]
	err := seg.Read(ref, dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* Iterate
* @param fn func(id string, data []byte) bool, workers int
* @return error
**/
func (s *FileStore) Iterate(fn func(id string, data []byte) bool, workers int) error {
	tag := "iterate"
	s.metricStart(tag)

	// 1. Seleccionar todos los IDs
	keys := make([]string, 0)
	indexResult := make(map[string]*recordRef, len(s.index))
	s.indexMu.RLock()
	s.Workers = workers
	for k, v := range s.index {
		keys = append(keys, k)
		indexResult[k] = v
	}
	s.indexMu.RUnlock()

	// 2. Orden determinista (opcional pero recomendado)
	s.metricSegment(tag, "load")
	sort.Strings(keys)
	s.metricSegment(tag, "sort")

	parts := chunkKeys(keys, s.Workers) // workers workers para paralelizar
	s.metricSegment(tag, "chunk")

	// 3. Procesar en paralelo
	n := 0
	var wg sync.WaitGroup
	for _, part := range parts {
		wg.Add(1)

		go func(keys []string) {
			defer wg.Done()

			for _, id := range keys {
				ref := indexResult[id]
				seg := s.segments[ref.segment]

				data, err := seg.read(ref)
				if err != nil {
					return
				}

				if !fn(id, data) {
					break
				}

				n++
			}
		}(part)
	}

	wg.Wait()
	msg := fmt.Sprintf("completed:total:%d:workers:%d", n, s.Workers)
	s.metricEnd(tag, msg)

	return nil
}

/**
* Prune
* @return error
**/
func (s *FileStore) Prune() error {
	tag := "prune"
	s.metricStart(tag)
	defer s.metricEnd(tag, "completed")

	err := s.Compact()
	if err != nil {
		return err
	}

	err = s.RebuildIndexes()
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
	s.index = make(map[string]*recordRef)
	s.WAL = 0
	s.TombStones = 0

	return nil
}

/**
* Open
* @param path, name string, debug bool
* @return *FileStore, error
**/
func Open(path, name string, debug bool) (*FileStore, error) {
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
		Metrics:      make(map[string]int64),
	}

	syncOnWrite := envar.GetBool("SYNC_ON_WRITE", true)
	fs.index = make(map[string]*recordRef)
	fs.IsDebug = debug
	fs.SyncOnWrite = syncOnWrite
	tag := "store_open"
	fs.metricStart(tag)

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

	fs.metricEnd(tag, fmt.Sprintf("total:%d:completed", len(fs.index)))
	go fs.logMetrics()
	return fs, nil
}
