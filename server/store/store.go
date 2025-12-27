package store

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type Store interface {
	Put(id string, value any) error
	Get(id string, dest any) error
	Delete(id string) error
	Sync() error
	Close() error
}

type recordHeader struct {
	DataLen uint32
	CRC     uint32
	IDLen   uint16
	ID      string
	Status  byte
}

/**
* ToJson
* @return et.Json
**/
func (s *recordHeader) ToJson() et.Json {
	return et.Json{
		"data_len": s.DataLen,
		"crc":      s.CRC,
		"id_len":   s.IDLen,
		"id":       s.ID,
		"status":   s.Status,
	}
}

/**
* ToString
* @return string
**/
func (s *recordHeader) ToString() string {
	return s.ToJson().ToString()
}

/**
* RecordSize
* @return int64
**/
func (s *recordHeader) RecordSize() int64 {
	return int64(fixedHeaderSize) + int64(s.IDLen) + int64(s.DataLen)
}

type segment struct {
	file *os.File
	size int64
	name string
}

/**
* newSegment
* @param file *os.File, size int64, name string
* @return *segment
**/
func newSegment(file *os.File, size int64, name string) *segment {
	result := &segment{
		file: file,
		size: size,
		name: name,
	}

	return result
}

/**
* ToJson
* @return et.Json
**/
func (s *segment) ToJson() et.Json {
	return et.Json{
		"file": s.file.Name(),
		"size": s.size,
		"name": s.name,
	}
}

/**
* ToString
* @return string
 */
func (s *segment) ToString() string {
	return s.ToJson().ToString()
}

/**
* ReadAt
* @param b []byte, off int64
* @return int, error
**/
func (s *segment) ReadAt(b []byte, off int64) (int, error) {
	if s.file == nil {
		return 0, errors.New(MSG_FILE_IS_NIL)
	}
	return s.file.ReadAt(b, off)
}

/**
* Write
* @param b []byte
* @return int, error
**/
func (s *segment) Write(b []byte) (int, error) {
	if s.file == nil {
		return 0, errors.New(MSG_FILE_IS_NIL)
	}
	return s.file.Write(b)
}

/**
* Sync
* @return error
**/
func (s *segment) Sync() error {
	if s.file == nil {
		return errors.New(MSG_FILE_IS_NIL)
	}
	return s.file.Sync()
}

/**
* ReadHeader
* @param ref recordRef
* @return recordHeader, error
**/
func (s *segment) ReadHeader(ref recordRef) (recordHeader, error) {
	var header recordHeader
	buf := make([]byte, fixedHeaderSize)
	_, err := s.ReadAt(buf, ref.offset)
	if err != nil {
		return header, err
	}

	reader := bytes.NewReader(buf)
	binary.Read(reader, binary.BigEndian, &header.DataLen)
	binary.Read(reader, binary.BigEndian, &header.CRC)
	binary.Read(reader, binary.BigEndian, &header.IDLen)

	idBytes := make([]byte, header.IDLen)
	if _, err := s.ReadAt(idBytes, ref.offset+10); err != nil {
		return header, err
	}
	header.ID = string(idBytes)

	statusLen := int64(1)
	statusByte := make([]byte, statusLen)
	if _, err := s.ReadAt(statusByte, ref.offset+10+int64(header.IDLen)); err != nil {
		return header, err
	}
	header.Status = statusByte[0]

	return header, nil
}

/**
* ReadData
* @param ref recordRef
* @return []byte, error
**/
func (s *segment) ReadData(ref recordRef) ([]byte, error) {
	header, err := s.ReadHeader(ref)
	if err != nil {
		return nil, err
	}

	headerLen := fixedHeaderSize + int64(header.IDLen)
	data := make([]byte, header.DataLen)
	_, err = s.ReadAt(data, ref.offset+headerLen)
	if err != nil {
		return nil, err
	}

	if checksum(data) != header.CRC {
		return nil, errors.New(MSG_CORRUPTED_RECORD)
	}

	return data, nil
}

/**
* ReadObject
* @param ref recordRef, dest any
* @return error
**/
func (s *segment) ReadObject(ref recordRef, dest any) error {
	data, err := s.ReadData(ref)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

/**
* writeRecord
* @param seg *segment, id string, data []byte, status byte
* @return recordRef, error
**/
func writeRecord(seg *segment, id string, data []byte, status byte) (recordRef, error) {
	h, header, err := newRecordHeaderAt(id, data, status)
	if err != nil {
		return recordRef{}, err
	}

	offset := seg.size

	if _, err := seg.Write(header); err != nil {
		return recordRef{}, err
	}
	if len(data) > 0 {
		if _, err := seg.Write(data); err != nil {
			return recordRef{}, err
		}
	}

	seg.size += h.RecordSize()

	return recordRef{
		offset: offset,
		length: uint32(len(data)),
	}, nil
}

const (
	packegeName     = "store"
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
		return recordHeader{}, nil, errors.New(MSG_INVALID_ID_LENGTH)
	}

	dataLen := len(data)
	if dataLen > math.MaxUint32 {
		return recordHeader{}, nil, errors.New(MSG_DATA_TOO_LARGE)
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

type recordRef struct {
	segment int
	offset  int64
	length  uint32
}

/**
* ToJson
* @return et.Json
**/
func (s recordRef) ToJson() et.Json {
	return et.Json{
		"segment": s.segment,
		"offset":  s.offset,
		"length":  s.length,
	}
}

/**
* ToString
* @return string
 */
func (s recordRef) ToString() string {
	return s.ToJson().ToString()
}

type FileStore struct {
	writeMu             sync.Mutex   // SOLO WAL append
	indexMu             sync.RWMutex // índice en memoria
	database            string
	name                string
	dir                 string
	tombStones          int
	dir_segments        string
	dir_snapshot        string
	dir_compact         string
	maxSegment          int64
	segments            []*segment
	active              *segment
	index               map[string]recordRef
	syncOnWrite         bool
	snapshotEvery       uint64
	writesSinceSnapshot uint64
	wg                  sync.WaitGroup
	ch                  chan []byte
}

/**
* ToJson
* @return et.Json
**/
func (s *FileStore) ToJson() et.Json {
	return et.Json{
		"database":     s.database,
		"name":         s.name,
		"dir":          s.dir,
		"tomb_stones":  s.tombStones,
		"dir_segments": s.dir_segments,
		"dir_snapshot": s.dir_snapshot,
		"dir_compact":  s.dir_compact,
		"max_segment":  s.maxSegment,
		"segments":     len(s.segments),
		"active":       s.active != nil,
		"index_size":   len(s.index),
	}
}

/**
* String
* @return string
 */
func (s *FileStore) String() string {
	return s.ToJson().ToString()
}

/**
* loop processes the write queue
* @return void
 */
func (s *FileStore) loop() {
	defer s.wg.Done()
	for data := range s.ch {
		logs.Logf("store", "Processing data from queue: %v", len(data))
		// TODO: Implement actual processing logic
		_ = data
	}
}

/**
* Close
* @return void
**/
func (s *FileStore) Close() {
	close(s.ch)
	s.wg.Wait()
}

/**
* loadSegments
* @return error
**/
func (s *FileStore) loadSegments() error {
	files, err := os.ReadDir(s.dir_segments)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, f := range files {
		name := f.Name()
		path := filepath.Join(s.dir_segments, name)
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
		logs.Log(packegeName, "loadSegments:", s.database, ":", s.name, ":", seg.ToString())
		s.segments = append(s.segments, seg)
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
	path := filepath.Join(s.dir_segments, name)

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	seg := newSegment(fd, 0, name)
	logs.Log(packegeName, "newSegment:", s.database, ":", s.name, ":", seg.ToString())
	s.segments = append(s.segments, seg)
	s.active = seg
	return nil
}

/**
* appendRecord
* @param id string, data []byte, status byte
* @return recordRef, error
**/
func (s *FileStore) appendRecord(id string, data []byte, status byte) (recordRef, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	ref, err := writeRecord(s.active, id, data, status)
	if err != nil {
		return recordRef{}, err
	}

	ref.segment = len(s.segments) - 1

	if s.syncOnWrite {
		if err := s.active.Sync(); err != nil {
			return recordRef{}, err
		}
	}

	return ref, nil
}

/**
* putIndex
* @param id string, segIndex int, offset int64, dataLen uint32
* @return error
**/
func (s *FileStore) putIndex(id string, segIndex int, offset int64, dataLen uint32) error {
	ref := recordRef{
		segment: segIndex,
		offset:  offset,
		length:  dataLen,
	}
	s.index[id] = ref
	logs.Log(packegeName, "putIndex:", segIndex, ":", s.database, ":", s.name, ":ID:", id, ":ref:", s.index[id].ToString())
	return nil
}

/**
* deleteIndex
* @param id string
* @return error
**/
func (s *FileStore) deleteIndex(id string) {
	delete(s.index, id)
	logs.Log(packegeName, "deleteIndex:", s.database, ":", s.name, ":ID:", id)
}

/**
* tryLoadSnapshot
* @return bool, error
**/
func (s *FileStore) tryLoadSnapshot() (bool, error) {
	path := filepath.Join(s.dir_snapshot, "state.snap")

	data, err := os.ReadFile(path)
	if err != nil {
		return false, nil // snapshot opcional
	}

	if len(data) < 10 {
		return false, errors.New("invalid snapshot")
	}

	// CRC check
	payload := data[:len(data)-4]
	storedCRC := getUint32(data[len(data)-4:])
	if checksum(payload) != storedCRC {
		return false, errors.New("snapshot corrupted")
	}

	buf := bytes.NewReader(payload)

	// ---- Header ----
	magic := make([]byte, 4)
	buf.Read(magic)
	if string(magic) != "SNAP" {
		return false, errors.New("invalid snapshot magic")
	}

	var version uint16
	binary.Read(buf, binary.BigEndian, &version)

	var count uint64
	binary.Read(buf, binary.BigEndian, &count)

	// ---- Entries ----
	s.index = make(map[string]recordRef, count)

	for i := uint64(0); i < count; i++ {
		var idLen uint16
		binary.Read(buf, binary.BigEndian, &idLen)

		idBytes := make([]byte, idLen)
		buf.Read(idBytes)

		var segIndex uint32
		var offset int64
		var dataLen uint32

		binary.Read(buf, binary.BigEndian, &segIndex)
		binary.Read(buf, binary.BigEndian, &offset)
		binary.Read(buf, binary.BigEndian, &dataLen)

		id := string(idBytes)
		s.putIndex(id, int(segIndex), offset, dataLen)
	}

	return true, nil
}

/**
* createSnapshot
* @return error
**/
func (s *FileStore) createSnapshot() error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	path := filepath.Join(s.dir_snapshot, "state.snap")
	tmp := path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := bytes.NewBuffer(nil)

	// ---- Header ----
	buf.WriteString("SNAP")
	binary.Write(buf, binary.BigEndian, uint16(1))
	binary.Write(buf, binary.BigEndian, uint64(len(s.index)))

	// ---- Entries ----
	for id, ref := range s.index {
		idBytes := []byte(id)
		binary.Write(buf, binary.BigEndian, uint16(len(idBytes)))
		buf.Write(idBytes)
		binary.Write(buf, binary.BigEndian, uint32(ref.segment))
		binary.Write(buf, binary.BigEndian, ref.offset)
		binary.Write(buf, binary.BigEndian, ref.length)
		logs.Log(packegeName, "snapshot:", s.database, ":", s.name, ":ID:", id, "seg:", ref.segment, ":offset:", ref.offset, ":len:", ref.length)
	}

	// ---- CRC ----
	crc := checksum(buf.Bytes())
	binary.Write(buf, binary.BigEndian, crc)

	if _, err := f.Write(buf.Bytes()); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	// atomic swap
	return os.Rename(tmp, path)
}

/**
* flushSnapshot
* @return
**/
func (s *FileStore) flushSnapshot() {
	if s.snapshotEvery > 0 {
		s.writesSinceSnapshot++
		if s.writesSinceSnapshot >= s.snapshotEvery {
			s.writesSinceSnapshot = 0
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.createSnapshot()
			}()
		}
	}

	n := len(s.index)
	threshold := int(float64(n) * 0.1) // 10% del tamaño del índice
	if s.tombStones > threshold {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.Compact()
		}()
	}

}

/**
* rebuildIndex
* @return error
**/
func (s *FileStore) rebuildIndex() error {
	if len(s.index) == 0 {
		s.index = make(map[string]recordRef)
	}

	for segIndex, seg := range s.segments {
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
				s.putIndex(id, segIndex, offset, dataLen)
			} else if status == Deleted {
				s.deleteIndex(id)
			}

			offset += int64(11) + int64(idLen) + int64(dataLen)
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
	if id == "" {
		return id, fmt.Errorf("id cannot be empty")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return id, err
	}

	ref, err := s.appendRecord(id, data, Active)
	if err != nil {
		return id, err
	}

	// índice: lock corto
	s.indexMu.Lock()
	_, exists := s.index[id]
	if exists {
		s.tombStones++
	}
	s.index[id] = ref
	s.indexMu.Unlock()
	i := len(s.index)
	logs.Log(packegeName, "putData:", i, ":", s.database, ":", s.name, ":ID:", id, ":ref:", ref.ToString())

	// snapshot async-safe
	s.flushSnapshot()

	return id, nil
}

/**
* Delete
* @param id string
* @return error, bool
**/
func (s *FileStore) Delete(id string) (error, bool) {
	// Verificamos si existe
	s.indexMu.RLock()
	_, exists := s.index[id]
	if exists {
		s.tombStones++
	}
	s.indexMu.RUnlock()

	if !exists {
		return nil, false
	}

	// Tombstone en WAL
	if _, err := s.appendRecord(id, nil, Deleted); err != nil {
		return logs.Error(err), true
	}

	// Log the deletion
	logs.Log(packegeName, "deleted", s.database, ":", s.name, ":ID:", id)

	// Removemos del índice
	s.indexMu.Lock()
	delete(s.index, id)
	s.indexMu.Unlock()

	// snapshot async-safe
	s.flushSnapshot()

	return nil, true
}

/**
* Get
* @param id string, dest any
* @return error
**/
func (s *FileStore) Get(id string, dest any) error {
	s.indexMu.RLock()
	ref, ok := s.index[id]
	s.indexMu.RUnlock()

	if !ok {
		return errors.New("not found")
	}

	seg := s.segments[ref.segment]
	err := seg.ReadObject(ref, dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* Iterate
* @param fn func(id string, data []byte) bool
* @return error
**/
func (s *FileStore) Iterate(fn func(id string, data []byte) bool) error {
	s.indexMu.RLock()
	indexResult := make(map[string]recordRef, len(s.index))
	for k, v := range s.index {
		indexResult[k] = v
	}
	s.indexMu.RUnlock()

	// 2. Orden determinista (opcional pero recomendado)
	keys := make([]string, 0, len(indexResult))
	for k := range indexResult {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. Iterar registros vivos
	for _, id := range keys {
		ref := indexResult[id]
		seg := s.segments[ref.segment]

		data, err := seg.ReadData(ref)
		if err != nil {
			return err
		}

		if !fn(id, data) {
			break
		}
	}

	return nil
}

/**
* Compact
* @return error
**/
func (s *FileStore) Compact() error {
	// Snapshot estable del índice
	s.indexMu.RLock()
	indexCopy := make(map[string]recordRef, len(s.index))
	for k, v := range s.index {
		indexCopy[k] = v
	}
	s.indexMu.RUnlock()

	// Orden determinista
	keys := make([]string, 0, len(indexCopy))
	for k := range indexCopy {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Directorio temporal
	tmpDir := filepath.Join(s.dir_compact, "segments.tmp")
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}

	var (
		newSegments []*segment
		current     *segment
	)

	createSegment := func() error {
		name := fmt.Sprintf("segment-%06d.dat", len(newSegments)+1)
		path := filepath.Join(tmpDir, name)

		fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return err
		}

		current = newSegment(fd, 0, name)
		newSegments = append(newSegments, current)
		return nil
	}

	if err := createSegment(); err != nil {
		return err
	}

	newIndex := make(map[string]recordRef, len(indexCopy))

	for _, id := range keys {
		ref := indexCopy[id]
		oldSeg := s.segments[ref.segment]

		// Leer header real
		fixed := make([]byte, fixedHeaderSize)
		if _, err := oldSeg.ReadAt(fixed, ref.offset); err != nil {
			return err
		}

		idLen := getUint16(fixed[8:10])
		payloadOffset := ref.offset + int64(fixedHeaderSize+idLen)

		data := make([]byte, ref.length)
		if ref.length > 0 {
			if _, err := oldSeg.ReadAt(data, payloadOffset); err != nil {
				return err
			}
		}

		// Rotar segmento si es necesario
		recordSize := int64(fixedHeaderSize) + int64(idLen) + int64(len(data))
		if current.size+recordSize > s.maxSegment {
			if err := createSegment(); err != nil {
				return err
			}
		}

		newRef, err := writeRecord(current, id, data, Active)
		if err != nil {
			return err
		}
		newRef.segment = len(newSegments) - 1
		newIndex[id] = newRef
		logs.Log(packegeName, "compacted:", s.database, ":", s.name, ":ID:", id, ":segment:", newRef.segment, ":offset:", newRef.offset, ":size:", newRef.length)
	}

	// Swap atómico
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	for _, seg := range s.segments {
		seg.file.Close()
	}

	oldDir := filepath.Join(s.dir, "segments.old")
	os.RemoveAll(oldDir)

	if err := os.Rename(s.dir_segments, oldDir); err != nil {
		return err
	}
	if err := os.Rename(tmpDir, s.dir_segments); err != nil {
		return err
	}

	// Activar nuevos segmentos
	s.indexMu.Lock()
	s.index = newIndex
	s.segments = newSegments
	s.active = newSegments[len(newSegments)-1]
	s.tombStones = 0
	s.indexMu.Unlock()

	return nil
}

/**
* normalize
* @param input string
* @return string
**/
func normalize(input string) string {
	// 1. Quitar espacios al inicio y final
	s := strings.TrimSpace(input)

	// 2. Reemplazar uno o más espacios por _
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "_")

	// 3. Eliminar todo lo que no sea letra, número o _
	s = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(s, "")

	// 4. Garantizar que no empiece con número
	s = regexp.MustCompile(`^[0-9]+`).ReplaceAllString(s, "")

	return s
}

/**
* Open
* @param dir, database, name string, maxSegmentBytes int64, syncOnWrite bool, snapshotEvery uint64
* @return *FileStore, error
**/
func Open(dir, database, name string, maxSegmentBytes int64, syncOnWrite bool, snapshotEvery uint64) (*FileStore, error) {
	if maxSegmentBytes < 1 {
		return nil, errors.New(MSG_MAX_SEGMENT_BYTES)
	}

	maxSegmentBytes = maxSegmentBytes * 1024 * 1024
	name = normalize(name)
	fs := &FileStore{
		database:      database,
		name:          name,
		dir:           dir,
		dir_segments:  filepath.Join(dir, database, name, "segments"),
		dir_snapshot:  filepath.Join(dir, database, name, "snapshot"),
		dir_compact:   filepath.Join(dir, database, name, "compact"),
		maxSegment:    maxSegmentBytes,
		syncOnWrite:   syncOnWrite,
		index:         make(map[string]recordRef),
		snapshotEvery: snapshotEvery,
		wg:            sync.WaitGroup{},
		ch:            make(chan []byte, 0),
	}
	fs.wg.Add(1)
	go fs.loop()

	if err := os.MkdirAll(fs.dir_segments, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fs.dir_snapshot, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fs.dir_compact, 0755); err != nil {
		return nil, err
	}

	if err := fs.loadSegments(); err != nil {
		return nil, fmt.Errorf("loadSegments: %w", err)
	}

	loaded, err := fs.tryLoadSnapshot()
	if err != nil {
		return nil, fmt.Errorf("tryLoadSnapshot: %w", err)
	}

	if !loaded {
		if err := fs.rebuildIndex(); err != nil {
			return nil, fmt.Errorf("rebuildIndex: %w", err)
		}
	}

	return fs, nil
}
