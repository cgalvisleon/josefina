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

type segment struct {
	file   *os.File
	offset int64
	size   int64
	name   string
}

/**
* ToJson
* @return et.Json
**/
func (s segment) ToJson() et.Json {
	return et.Json{
		"file":   s.file.Name(),
		"offset": s.offset,
		"size":   s.size,
		"name":   s.name,
	}
}

/**
* ToString
* @return string
 */
func (s segment) ToString() string {
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

const (
	packegeName     = "store"
	maxIdLen        = 65535
	fixedHeaderSize = 11
)

type recordHeader struct {
	DataLen uint32
	CRC     uint32
	IDLen   uint16
	Status  byte
}

/**
* RecordSize
* @return int64
**/
func (h *recordHeader) RecordSize() int64 {
	return int64(fixedHeaderSize) + int64(h.IDLen) + int64(h.DataLen)
}

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
	}

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

	if err := fs.tryLoadSnapshot(); err != nil {
		return nil, fmt.Errorf("tryLoadSnapshot: %w", err)
	}

	if err := fs.rebuildIndex(); err != nil {
		return nil, fmt.Errorf("rebuildIndex: %w", err)
	}

	return fs, nil
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

		seg := &segment{
			file: fd,
			size: size,
			name: name,
		}

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

	seg := &segment{
		file: fd,
		size: 0,
		name: name,
	}

	logs.Log(packegeName, "newSegment:", s.database, ":", s.name, ":", seg.ToString())
	s.segments = append(s.segments, seg)
	s.active = seg
	return nil
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

/*
*
* appendRecord
* @param id string, data []byte, status byte
* @return recordRef, error
*
 */
func (s *FileStore) appendRecord(id string, data []byte, status byte) (recordRef, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	ref, err := writeRecord(s.active, id, data, status)
	if err != nil {
		return recordRef{}, err
	}

	ref.segment = len(s.segments) - 1

	// idBytes := []byte(id)
	// idLen := len(idBytes)

	// if idLen == 0 || idLen > maxIdLen {
	// 	return recordRef{}, errors.New(MSG_INVALID_ID_LENGTH)
	// }

	// dataLen := len(data)
	// if dataLen > math.MaxUint32 {
	// 	return recordRef{}, errors.New(MSG_DATA_TOO_LARGE)
	// }

	// headerLen := 11 + idLen
	// recordSize := int64(headerLen) + int64(dataLen)

	// if s.active.size+recordSize > s.maxSegment {
	// 	if err := s.newSegment(); err != nil {
	// 		return recordRef{}, err
	// 	}
	// }

	// offset := s.active.size

	// header := make([]byte, headerLen)
	// putUint32(header[0:4], uint32(dataLen))
	// putUint32(header[4:8], checksum(data))
	// putUint16(header[8:10], uint16(idLen))
	// copy(header[10:10+idLen], idBytes)
	// header[10+idLen] = status

	// if _, err := s.active.Write(header); err != nil {
	// 	return recordRef{}, err
	// }

	// if len(data) > 0 {
	// 	if _, err := s.active.Write(data); err != nil {
	// 		return recordRef{}, err
	// 	}
	// }

	// s.active.size += recordSize

	if s.syncOnWrite {
		if err := s.active.Sync(); err != nil {
			return recordRef{}, err
		}
	}

	return ref, nil
	// return recordRef{
	// 	segment: len(s.segments) - 1,
	// 	offset:  offset,
	// 	length:  uint32(dataLen),
	// }, nil
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
		i := 0

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

			i++
			if status == Active {
				s.index[id] = recordRef{
					segment: segIndex,
					offset:  offset,
					length:  dataLen,
				}

				logs.Log(packegeName, "rebuildIndex:", i, ":", s.database, ":", s.name, ":ID:", id, ":ref:", s.index[id].ToString())
			} else if status == Deleted {
				logs.Log(packegeName, "rebuildIndex:deleted:", i, ":", s.database, ":", s.name, ":ID:", id)
				delete(s.index, id)
			}

			offset += int64(11) + int64(idLen) + int64(dataLen)
		}
	}

	return nil
}

/**
* tryLoadSnapshot
* @return error
**/
func (s *FileStore) tryLoadSnapshot() error {
	path := filepath.Join(s.dir_snapshot, "state.snap")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil // snapshot opcional
	}

	if len(data) < 10 {
		return errors.New("invalid snapshot")
	}

	// CRC check
	payload := data[:len(data)-4]
	storedCRC := getUint32(data[len(data)-4:])
	if checksum(payload) != storedCRC {
		return errors.New("snapshot corrupted")
	}

	buf := bytes.NewReader(payload)

	// ---- Header ----
	magic := make([]byte, 4)
	buf.Read(magic)
	if string(magic) != "SNAP" {
		return errors.New("invalid snapshot magic")
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

		var seg uint32
		var offset int64
		var length uint32

		binary.Read(buf, binary.BigEndian, &seg)
		binary.Read(buf, binary.BigEndian, &offset)
		binary.Read(buf, binary.BigEndian, &length)

		id := string(idBytes)
		s.index[id] = recordRef{
			segment: int(seg),
			offset:  offset,
			length:  length,
		}
	}

	return nil
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
* Put
* @param id string, value any
* @return error
**/
func (s *FileStore) Put(id string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ref, err := s.appendRecord(id, data, Active)
	if err != nil {
		return err
	}

	// índice: lock corto
	s.indexMu.Lock()
	s.index[id] = ref
	s.indexMu.Unlock()
	i := len(s.index)
	logs.Log(packegeName, "put:", i, ":", s.database, ":", s.name, ":ID:", id, ":ref:", ref.ToString())

	// snapshot async-safe
	if s.snapshotEvery > 0 {
		s.writesSinceSnapshot++
		if s.writesSinceSnapshot >= s.snapshotEvery {
			s.writesSinceSnapshot = 0
			// go
			s.createSnapshot()
		}
	}

	return nil
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

	header := make([]byte, 25)
	if _, err := seg.ReadAt(header, ref.offset); err != nil {
		return err
	}

	data := make([]byte, ref.length)
	if _, err := seg.ReadAt(data, ref.offset+25); err != nil {
		return err
	}

	if checksum(data) != getUint32(header[4:8]) {
		return errors.New("corrupted record")
	}

	return json.Unmarshal(data, dest)
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

		// Leer header mínimo
		fixed := make([]byte, 11)
		if _, err := seg.ReadAt(fixed, ref.offset); err != nil {
			return err
		}

		dataLen := getUint32(fixed[0:4])
		crcStored := getUint32(fixed[4:8])
		idLen := getUint16(fixed[8:10])
		// status := fixed[10] // no lo necesitas aquí

		// Validación básica
		if idLen == 0 {
			return errors.New("invalid record: zero idLen")
		}

		payloadOffset := ref.offset + int64(11+idLen)

		data := make([]byte, dataLen)
		if dataLen > 0 {
			if _, err := seg.ReadAt(data, payloadOffset); err != nil {
				return err
			}
			if checksum(data) != crcStored {
				return errors.New("corrupted record during iterate")
			}
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

		current = &segment{
			file: fd,
			size: 0,
			name: name,
		}
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
	s.indexMu.Unlock()

	return nil
}
