package store

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type recordHeader struct {
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
	file *os.File
	size int64
	name string
	ch   chan []byte
	wg   sync.WaitGroup
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
		ch:   make(chan []byte),
	}

	result.wg.Add(1)
	go result.loop()
	return result
}

/**
* loop
* @return void
**/
func (s *segment) loop() {
	defer s.wg.Done()
	for data := range s.ch {
		s.file.Write(data)
	}
}

/**
* Sync
* @return error
**/
func (s *segment) Sync() error {
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
	close(s.ch)
	s.wg.Wait()
	err := s.Sync()
	if err != nil {
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
	if s.file == nil {
		return
	}
	s.ch <- b
}

/**
* WriteHeader
* @param id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *segment) WriteHeader(id string, data []byte, status byte) (*RecordRef, error) {
	h, header, err := newRecordHeaderAt(id, data, status)
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
* WriteRecord
* @param seg *segment, id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *segment) WriteRecord(id string, data []byte, status byte) (*RecordRef, error) {
	h, header, err := newRecordHeaderAt(id, data, status)
	if err != nil {
		return nil, err
	}

	offset := s.size

	s.Write(header)
	if len(data) > 0 {
		s.Write(data)
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
