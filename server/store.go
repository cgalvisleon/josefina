package server

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type RecordStatus byte

const (
	Active  RecordStatus = 1
	Deleted RecordStatus = 0
)

type Config struct {
	SegmentMaxBytes int64 // ej 100 * 1024 * 1024
}

type Store struct {
	mu    sync.RWMutex
	file  *os.File
	index map[uint64]int64 // id -> offset
	path  string
}

/**
* OpenStore
* @param path string
* @return *Store, error
**/
func OpenStore(path string) (*Store, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	s := &Store{
		file:  f,
		path:  path,
		index: make(map[uint64]int64),
	}

	if err := s.rebuildIndex(); err != nil {
		return nil, err
	}

	return s, nil
}

/**
* rebuildIndex
* @return error
**/
func (s *Store) rebuildIndex() error {
	offset := int64(0)

	for {
		header := make([]byte, 13) // len(4) + id(8) + status(1)
		_, err := s.file.ReadAt(header, offset)
		if err != nil {
			return nil
		}

		length := binary.BigEndian.Uint32(header[0:4])
		id := binary.BigEndian.Uint64(header[4:12])
		status := RecordStatus(header[12])

		if status == Active {
			s.index[id] = offset
		} else {
			delete(s.index, id)
		}

		offset += int64(13 + length)
	}
}

/**
* Put
* @param id uint64, value any
* @return error
**/
func (s *Store) Put(id uint64, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	offset, _ := s.file.Seek(0, os.SEEK_END)

	buf := make([]byte, 13+len(data))
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(data)))
	binary.BigEndian.PutUint64(buf[4:12], id)
	buf[12] = byte(Active)
	copy(buf[13:], data)

	if _, err := s.file.Write(buf); err != nil {
		return err
	}

	s.index[id] = offset
	return nil
}

/**
* Get
* @param id uint64, dest any
* @return error
**/
func (s *Store) Get(id uint64, dest any) error {
	s.mu.RLock()
	offset, ok := s.index[id]
	s.mu.RUnlock()

	if !ok {
		return errors.New("not found")
	}

	header := make([]byte, 13)
	_, err := s.file.ReadAt(header, offset)
	if err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(header[0:4])

	data := make([]byte, length)
	_, err = s.file.ReadAt(data, offset+13)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

/**
* Delete
* @param id uint64
* @return error
**/
func (s *Store) Delete(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	offset, ok := s.index[id]
	if !ok {
		return nil
	}

	_, err := s.file.WriteAt([]byte{byte(Deleted)}, offset+12)
	if err != nil {
		return err
	}

	delete(s.index, id)
	return nil
}
