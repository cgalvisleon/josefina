package store

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type recordHeader struct {
	LSN     uint64 `json:"lsn"`
	DataLen uint32 `json:"data_len"`
	CRC     uint32 `json:"crc"`
	IDLen   uint16 `json:"id_len"`
	ID      string `json:"id"`
	Status  byte   `json:"status"`
}

/**
* HeaderSize
* @return int64
**/
func (s *recordHeader) HeaderSize() int64 {
	return int64(fixedHeaderSize) + int64(s.IDLen)
}

/**
* RecordSize
* @return int64
**/
func (s *recordHeader) RecordSize() int64 {
	return s.HeaderSize() + int64(s.DataLen)
}

type RecordRef struct {
	segment int
	offset  int64
	length  uint32
}

/**
* ToJson
* @return et.Json
**/
func (s *RecordRef) ToJson() et.Json {
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
func (s *RecordRef) ToString() string {
	return s.ToJson().ToString()
}

type segment struct {
	file     *os.File
	size     int64
	name     string
	readOnly bool
	ch       chan []byte
	wg       sync.WaitGroup
	writeErr error
	errMu    sync.Mutex
}

/**
* newSegment
* @param file *os.File, size int64, name string
* @return *segment
**/
func newSegment(file *os.File, size int64, name string) *segment {
	result := &segment{
		file:     file,
		size:     size,
		name:     name,
		readOnly: false,
		ch:       make(chan []byte),
		writeErr: nil,
		errMu:    sync.Mutex{},
	}

	result.wg.Add(1)
	go result.loop()
	return result
}

/**
* newReadOnlySegment creates a segment without a write goroutine.
* Safe to use when the underlying file is opened O_RDONLY.
* @param file *os.File, size int64, name string
* @return *segment
**/
func newReadOnlySegment(file *os.File, size int64, name string) *segment {
	return &segment{
		file:     file,
		size:     size,
		name:     name,
		readOnly: true,
	}
}

/**
* loop
* @return void
**/
func (s *segment) loop() error {
	defer s.wg.Done()
	for data := range s.ch {
		_, err := s.file.Write(data)
		if err != nil {
			s.errMu.Lock()
			s.writeErr = err
			s.errMu.Unlock()
			for range s.ch {
			}
			return err
		}
	}

	return nil
}

/**
* Sync
* @return error
**/
func (s *segment) Sync() error {
	if s.readOnly {
		return nil
	}
	if s.file == nil {
		return errors.New(msg.MSG_FILE_IS_NIL)
	}
	return s.file.Sync()
}

/**
* Close
* @return error
**/
func (s *segment) Close() error {
	if s.readOnly {
		if s.file != nil {
			return s.file.Close()
		}
		return nil
	}
	close(s.ch)
	s.wg.Wait()
	if err := s.Sync(); err != nil {
		return err
	}
	return s.file.Close()
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
		return 0, errors.New(msg.MSG_FILE_IS_NIL)
	}
	return s.file.ReadAt(b, off)
}

/**
* Write
* @param b []byte
**/
func (s *segment) Write(b []byte) {
	if s.file == nil || s.readOnly {
		return
	}
	s.ch <- b
}

/**
* WriteHeader
* @param id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *segment) WriteHeader(lsn uint64, id string, data []byte, status byte) (*RecordRef, error) {
	h, header, err := newRecordHeaderAt(lsn, id, data, status)
	if err != nil {
		return nil, err
	}

	offset := s.size

	s.Write(header)
	s.size += h.HeaderSize()

	return &RecordRef{
		offset: offset,
		length: h.DataLen,
	}, nil
}

/**
* WriteError
* @return error
**/
func (s *segment) WriteError() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.writeErr
}

/**
* WriteRecord
* @param seg *segment, id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *segment) WriteRecord(lsn uint64, id string, data []byte, status byte) (*RecordRef, error) {
	h, header, err := newRecordHeaderAt(lsn, id, data, status)
	if err != nil {
		return nil, err
	}

	offset := s.size
	s.Write(header)
	if len(data) > 0 {
		s.Write(data)
	}
	if err := s.WriteError(); err != nil {
		return nil, err
	}

	s.size += h.RecordSize()
	return &RecordRef{
		offset: offset,
		length: h.DataLen,
	}, nil
}

/**
* ReadHeader
* @param ref *RecordRef
* @return recordHeader, error
**/
func (s *segment) ReadHeader(ref *RecordRef) (recordHeader, error) {
	var header recordHeader
	buf := make([]byte, fixedHeaderSize)
	_, err := s.ReadAt(buf, ref.offset)
	if err != nil {
		return header, err
	}

	// Layout: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1]
	reader := bytes.NewReader(buf)
	if err := binary.Read(reader, binary.BigEndian, &header.LSN); err != nil {
		return header, fmt.Errorf("read LSN: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.DataLen); err != nil {
		return header, fmt.Errorf("read DataLen: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.CRC); err != nil {
		return header, fmt.Errorf("read CRC: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.IDLen); err != nil {
		return header, fmt.Errorf("read IDLen: %w", err)
	}

	idBytes := make([]byte, header.IDLen)
	if _, err := s.ReadAt(idBytes, ref.offset+18); err != nil {
		return header, err
	}
	header.ID = string(idBytes)

	statusByte := make([]byte, 1)
	if _, err := s.ReadAt(statusByte, ref.offset+18+int64(header.IDLen)); err != nil {
		return header, err
	}
	header.Status = statusByte[0]

	return header, nil
}

/**
* Read
* @param ref *RecordRef
* @return []byte, error
**/
func (s *segment) read(ref *RecordRef) ([]byte, error) {
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
		return nil, errors.New(msg.MSG_CORRUPTED_RECORD)
	}

	return data, nil
}

/**
* Read
* @param ref *RecordRef, dest any
* @return error
**/
func (s *segment) Read(ref *RecordRef, dest any) error {
	data, err := s.read(ref)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}
