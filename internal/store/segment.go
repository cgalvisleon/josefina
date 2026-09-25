package store

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"

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
	readers   atomic.Int32 // lectores en curso (Get/Read/ForEach) que no sostienen indexMu
	retired   atomic.Bool  // fuera del store (compaction/close): se cierra con el último lector
}

/**
* acquire: Registers a reader. Only call it while the segment is still part of
* the store (under indexMu), so it can never race with retire().
**/
func (s *segment) acquire() {
	s.readers.Add(1)
}

/**
* release: Unregisters a reader and closes the file if the segment was retired
* and this was the last reader.
**/
func (s *segment) release() {
	if s.readers.Add(-1) == 0 && s.retired.Load() {
		s.Close()
	}
}

/**
* retire: Marks the segment as no longer part of the store. The file is closed
* now if nobody is reading it, otherwise by the last reader's release().
**/
func (s *segment) retire() {
	s.retired.Store(true)
	if s.readers.Load() == 0 {
		s.Close()
	}
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
* decodeFixedHeader: Decodes the fixed part of a record header.
* Layout: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1][Data]
* @param b []byte (at least 18 bytes)
* @return recordHeader
**/
func decodeFixedHeader(b []byte) recordHeader {
	return recordHeader{
		LSN:     binary.BigEndian.Uint64(b[0:8]),
		DataLen: getUint32(b[8:12]),
		CRC:     getUint32(b[12:16]),
		IDLen:   getUint16(b[16:18]),
	}
}

/**
* decodeRecord: Validates a whole record read into buf and returns its data.
* Checks that it is the live record ref points at (same id, length and Active
* status) and that the payload matches its CRC.
* @param buf []byte, ref *RecordRef, id string
* @return []byte, error
**/
func decodeRecord(buf []byte, ref *RecordRef, id string) ([]byte, error) {
	h := decodeFixedHeader(buf)
	idLen := int(h.IDLen)
	if idLen != len(id) || h.DataLen != ref.length || len(buf) != int(fixedHeaderSize)+idLen+int(h.DataLen) {
		return nil, errors.New(msg.MSG_CORRUPTED_RECORD)
	}
	if string(buf[18:18+idLen]) != id || buf[18+idLen] != Active {
		return nil, errors.New(msg.MSG_CORRUPTED_RECORD)
	}

	data := buf[int(fixedHeaderSize)+idLen:]
	if checksum(data) != h.CRC {
		return nil, errors.New(msg.MSG_CORRUPTED_RECORD)
	}

	return data, nil
}

/**
* readRecord: Reads the record of id in a single ReadAt (header, id and data are
* contiguous and their sizes are known from id and ref).
* @param ref *RecordRef, id string
* @return []byte, error
**/
func (s *segment) readRecord(ref *RecordRef, id string) ([]byte, error) {
	buf := make([]byte, int(fixedHeaderSize)+len(id)+int(ref.length))
	if _, err := s.ReadAt(buf, ref.offset); err != nil {
		return nil, err
	}

	return decodeRecord(buf, ref, id)
}

/**
* read: Reads the record at ref when its id is not known: one ReadAt for the
* fixed header (to learn the id length) and one for the whole record.
* @param ref *RecordRef
* @return []byte, error
**/
func (s *segment) read(ref *RecordRef) ([]byte, error) {
	head := make([]byte, 18)
	if _, err := s.ReadAt(head, ref.offset); err != nil {
		return nil, err
	}

	idLen := int(getUint16(head[16:18]))
	buf := make([]byte, int(fixedHeaderSize)+idLen+int(ref.length))
	if _, err := s.ReadAt(buf, ref.offset); err != nil {
		return nil, err
	}

	return decodeRecord(buf, ref, string(buf[18:18+idLen]))
}

/**
* ReadHeader: Reads the header (including id and status) of the record at ref
* @param ref *RecordRef
* @return recordHeader, error
**/
func (s *segment) ReadHeader(ref *RecordRef) (recordHeader, error) {
	head := make([]byte, 18)
	if _, err := s.ReadAt(head, ref.offset); err != nil {
		return recordHeader{}, err
	}

	h := decodeFixedHeader(head)
	rest := make([]byte, int(h.IDLen)+1)
	if _, err := s.ReadAt(rest, ref.offset+18); err != nil {
		return recordHeader{}, err
	}
	h.ID = string(rest[:h.IDLen])
	h.Status = rest[h.IDLen]

	return h, nil
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

		h := decodeFixedHeader(fixed)
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
