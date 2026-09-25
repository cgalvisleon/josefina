package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/josefina/internal/msg"
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
	file      *os.File
	size      int64
	name      string
	readOnly  bool
	writeErr  error
	errMu     sync.Mutex
	closeOnce sync.Once
}

/**
* newSegment
* @param file *os.File, size int64, name string
* @return *segment
**/
func newSegment(file *os.File, size int64, name string) *segment {
	return &segment{
		file:     file,
		size:     size,
		name:     name,
		readOnly: false,
	}
}

/**
* newReadOnlySegment creates a segment that rejects writes.
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
* Seal: Flushes to disk and marks the segment read-only, keeping the file open
* so records already indexed in it can still be read.
* @return error
**/
func (s *segment) Seal() error {
	if s.readOnly {
		return nil
	}

	if err := s.WriteError(); err != nil {
		return err
	}
	if err := s.file.Sync(); err != nil {
		return err
	}

	s.readOnly = true
	return nil
}

/**
* Close
* @return error
**/
func (s *segment) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		if s.file == nil {
			return
		}
		if s.readOnly {
			closeErr = s.file.Close()
			return
		}
		if err := s.Sync(); err != nil {
			closeErr = err
			return
		}
		closeErr = s.file.Close()
	})

	return closeErr
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
* Write: Writes b to the file synchronously, so a record is on the file (page
* cache) before its RecordRef is returned and published in the index. The first
* error is sticky: once a write fails, offsets can no longer be trusted, so every
* later write returns that same error.
* @param b []byte
* @return error
**/
func (s *segment) Write(b []byte) error {
	if s.file == nil {
		return errors.New(msg.MSG_FILE_IS_NIL)
	}
	if s.readOnly {
		return errors.New(msg.MSG_STORE_IS_READ_ONLY)
	}
	if err := s.WriteError(); err != nil {
		return err
	}

	if _, err := s.file.Write(b); err != nil {
		s.errMu.Lock()
		s.writeErr = err
		s.errMu.Unlock()
		return err
	}

	return nil
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

	if err := s.Write(header); err != nil {
		return nil, err
	}
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

	// Header and payload go out in a single write so a reader never sees a
	// header without its data.
	offset := s.size
	record := make([]byte, 0, len(header)+len(data))
	record = append(record, header...)
	record = append(record, data...)
	if err := s.Write(record); err != nil {
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
func (s *segment) Read(ref *RecordRef) ([]byte, error) {
	result, err := s.read(ref)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* scan: Walks the records of the segment starting at offset, calling fn for each
* one in write order. Layout: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1][Data].
* Stops quietly at the end of the file or at the first corrupt/partial record (a
* torn tail write); only an I/O error reading a header is returned. fn returning
* an error stops the scan and returns it.
* @param offset int64, fn func(offset int64, h recordHeader, data []byte) error
* @return error
**/
func (s *segment) scan(offset int64, fn func(offset int64, h recordHeader, data []byte) error) error {
	for {
		fixed := make([]byte, fixedHeaderSize)
		n, err := s.ReadAt(fixed, offset)
		if err != nil {
			if errors.Is(err, io.EOF) || n < len(fixed) {
				return nil
			}
			return err
		}

		h := recordHeader{
			LSN:     binary.BigEndian.Uint64(fixed[0:8]),
			DataLen: getUint32(fixed[8:12]),
			CRC:     getUint32(fixed[12:16]),
			IDLen:   getUint16(fixed[16:18]),
		}
		if h.IDLen == 0 || h.IDLen > maxIdLen {
			return nil // corrupción → parar seguro
		}

		idBytes := make([]byte, h.IDLen)
		if _, err := s.ReadAt(idBytes, offset+18); err != nil {
			return nil
		}
		h.ID = string(idBytes)

		statusByte := make([]byte, 1)
		if _, err := s.ReadAt(statusByte, offset+18+int64(h.IDLen)); err != nil {
			return nil
		}
		h.Status = statusByte[0]

		var data []byte
		if h.DataLen > 0 {
			data = make([]byte, h.DataLen)
			if _, err := s.ReadAt(data, offset+18+int64(h.IDLen)+1); err != nil {
				return nil
			}
			if checksum(data) != h.CRC {
				return nil // registro corrupto → detener el escaneo
			}
		}

		if err := fn(offset, h, data); err != nil {
			return err
		}

		offset += h.RecordSize()
	}
}
