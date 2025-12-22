package store

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
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

type FileStore struct {
	writeMu sync.Mutex   // SOLO WAL append
	indexMu sync.RWMutex // índice en memoria

	dir        string
	maxSegment int64

	segments []*segment
	active   *segment

	index map[string]recordRef

	syncOnWrite bool

	snapshotEvery       uint64
	writesSinceSnapshot uint64
}

type recordRef struct {
	segment int
	offset  int64
	length  uint32
}

/**
* Open
* @param dir string, maxSegmentBytes int64, syncOnWrite bool, snapshotEvery uint64
* @return *FileStore, error
**/
func Open(dir string, maxSegmentBytes int64, syncOnWrite bool, snapshotEvery uint64) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Join(dir, "segments"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "snapshot"), 0755); err != nil {
		return nil, err
	}

	fs := &FileStore{
		dir:           dir,
		maxSegment:    maxSegmentBytes,
		syncOnWrite:   syncOnWrite,
		index:         make(map[string]recordRef),
		snapshotEvery: snapshotEvery,
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
* appendRecord
* @param id string, data []byte, status byte
* @return recordRef, error
**/
func (s *FileStore) appendRecord(id string, data []byte, status byte) (recordRef, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	recordSize := int64(25 + len(data))
	if s.active.size+recordSize > s.maxSegment {
		if err := s.newSegment(); err != nil {
			return recordRef{}, err
		}
	}

	header := make([]byte, 25)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(data)))
	binary.BigEndian.PutUint32(header[4:8], checksum(data))

	copy(header[8:24], []byte(id)) // luego lo mejoramos
	header[24] = status

	offset := s.active.size

	if _, err := s.active.file.Write(header); err != nil {
		return recordRef{}, err
	}
	if len(data) > 0 {
		if _, err := s.active.file.Write(data); err != nil {
			return recordRef{}, err
		}
	}

	s.active.size += int64(len(header) + len(data))

	if s.syncOnWrite {
		if err := s.active.file.Sync(); err != nil {
			return recordRef{}, err
		}
	}

	return recordRef{
		segment: len(s.segments) - 1,
		offset:  offset,
		length:  uint32(len(data)),
	}, nil
}

/**
* loadSegments
* @return error
**/
func (s *FileStore) loadSegments() error {
	segDir := filepath.Join(s.dir, "segments")

	files, err := os.ReadDir(segDir)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, f := range files {
		path := filepath.Join(segDir, f.Name())
		st, _ := os.Stat(path)

		fd, err := os.OpenFile(path, os.O_RDWR, 0644)
		if err != nil {
			return err
		}

		seg := &segment{
			file: fd,
			size: st.Size(),
			name: f.Name(),
		}

		s.segments = append(s.segments, seg)
	}

	if len(s.segments) == 0 {
		return s.newSegment()
	}

	s.active = s.segments[len(s.segments)-1]
	return nil
}

/**
* rebuildIndex
* @return error
**/
func (s *FileStore) rebuildIndex() error {
	s.index = make(map[string]recordRef)

	for segIndex, seg := range s.segments {
		offset := int64(0)

		for {
			header := make([]byte, 25)

			n, err := seg.file.ReadAt(header, offset)
			if err != nil {
				if errors.Is(err, io.EOF) || n == 0 {
					break // finished segment
				}

				// archivo pudo quedar truncado → parar seguro
				if n < len(header) {
					break
				}

				return err
			}

			length := binary.BigEndian.Uint32(header[0:4])
			crcStored := binary.BigEndian.Uint32(header[4:8])

			var id string
			id = string(header[8:24])

			status := header[24]

			// leer payload
			data := make([]byte, length)
			n, err = seg.file.ReadAt(data, offset+25)
			if err != nil {
				break // truncado → paramos seguro
			}

			// validar CRC
			if checksum(data) != crcStored {
				// registro dañado -> lo ignoramos y paramos
				break
			}

			if status == Active {
				s.index[id] = recordRef{
					segment: segIndex,
					offset:  offset,
					length:  length,
				}
			} else {
				delete(s.index, id)
			}

			offset += int64(25 + length)
		}
	}

	return nil
}

/**
* newSegment
* @return error
**/
func (s *FileStore) newSegment() error {
	name := fmt.Sprintf("segment-%06d.dat", len(s.segments)+1)
	path := filepath.Join(s.dir, "segments", name)

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	seg := &segment{
		file: fd,
		size: 0,
		name: name,
	}

	s.segments = append(s.segments, seg)
	s.active = seg
	return nil
}

/**
* createSnapshot
* @return error
**/
func (s *FileStore) createSnapshot() error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	path := filepath.Join(s.dir, "snapshot", "state.snap")
	tmp := path + ".tmp"

	if err := os.MkdirAll(filepath.Join(s.dir, "snapshot"), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := bytes.NewBuffer(nil)

	// version
	binary.Write(buf, binary.BigEndian, uint16(1))

	// record count
	count := uint64(len(s.index))
	binary.Write(buf, binary.BigEndian, count)

	// entries
	for id, ref := range s.index {
		buf.WriteString(id)
		binary.Write(buf, binary.BigEndian, uint32(ref.segment))
		binary.Write(buf, binary.BigEndian, ref.offset)
		binary.Write(buf, binary.BigEndian, ref.length)
	}

	// crc
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
* tryLoadSnapshot
* @return error
**/
func (s *FileStore) tryLoadSnapshot() error {
	path := filepath.Join(s.dir, "snapshot", "state.snap")

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	buf := bytes.NewReader(data)

	var version uint16
	binary.Read(buf, binary.BigEndian, &version)

	var count uint64
	binary.Read(buf, binary.BigEndian, &count)

	// all but last 4 bytes
	payload := data[:len(data)-4]
	storedCRC := binary.BigEndian.Uint32(data[len(data)-4:])

	if checksum(payload) != storedCRC {
		return errors.New("snapshot corrupted")
	}

	for i := uint64(0); i < count; i++ {
		var idBytes [16]byte
		buf.Read(idBytes[:])

		var seg uint32
		binary.Read(buf, binary.BigEndian, &seg)

		var offset int64
		binary.Read(buf, binary.BigEndian, &offset)

		var length uint32
		binary.Read(buf, binary.BigEndian, &length)

		var id string
		id = string(idBytes[:])

		s.index[id] = recordRef{
			segment: int(seg),
			offset:  offset,
			length:  length,
		}
	}

	return nil
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

	// snapshot async-safe
	if s.snapshotEvery > 0 {
		s.writesSinceSnapshot++
		if s.writesSinceSnapshot >= s.snapshotEvery {
			s.writesSinceSnapshot = 0
			go s.createSnapshot()
		}
	}

	return nil
}

/**
* Delete
* @param id string
* @return error
**/
func (s *FileStore) Delete(id string) error {
	// Verificamos si existe
	s.indexMu.RLock()
	_, exists := s.index[id]
	s.indexMu.RUnlock()

	if !exists {
		return nil
	}

	// Tombstone en WAL
	if _, err := s.appendRecord(id, nil, Deleted); err != nil {
		return err
	}

	// Removemos del índice
	s.indexMu.Lock()
	delete(s.index, id)
	s.indexMu.Unlock()

	return nil
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
	if _, err := seg.file.ReadAt(header, ref.offset); err != nil {
		return err
	}

	data := make([]byte, ref.length)
	if _, err := seg.file.ReadAt(data, ref.offset+25); err != nil {
		return err
	}

	if checksum(data) != binary.BigEndian.Uint32(header[4:8]) {
		return errors.New("corrupted record")
	}

	return json.Unmarshal(data, dest)
}
