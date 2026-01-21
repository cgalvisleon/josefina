package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* CreateSnapshot
* @return error
**/
func (s *FileStore) CreateSnapshot() error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	name := fmt.Sprintf("state-%s.snap", s.Name)
	path := filepath.Join(s.PathSnapshot, name)
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
	currentSegment := len(s.segments) - 1
	for id, ref := range s.index {
		if ref.segment == currentSegment {
			continue
		}
		idBytes := []byte(id)
		binary.Write(buf, binary.BigEndian, uint16(len(idBytes)))
		buf.Write(idBytes)
		binary.Write(buf, binary.BigEndian, uint32(ref.segment))
		binary.Write(buf, binary.BigEndian, ref.offset)
		binary.Write(buf, binary.BigEndian, ref.length)
		isDebug := envar.GetBool("DEBUG", false)
		if isDebug {
			logs.Debug("snapshot:", s.Path, ":", s.Name, ":ID:", id, "seg:", ref.segment, ":offset:", ref.offset, ":len:", ref.length)
		}
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
* tryLoadSnapshot
* @return error
**/
func (s *FileStore) tryLoadSnapshot() error {
	name := fmt.Sprintf("state-%s.snap", s.Name)
	path := filepath.Join(s.PathSnapshot, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // snapshot opcional
	}

	if len(data) < 10 {
		return errors.New(msg.MSG_INVALID_SNAPSHOT)
	}

	// CRC check
	payload := data[:len(data)-4]
	storedCRC := getUint32(data[len(data)-4:])
	if checksum(payload) != storedCRC {
		return errors.New(msg.MSG_SNAPSHOT_CORRUPTED)
	}

	buf := bytes.NewReader(payload)

	// ---- Header ----
	magic := make([]byte, 4)
	buf.Read(magic)
	if string(magic) != "SNAP" {
		return errors.New(msg.MSG_INVALID_SNAPSHOT_MAGIC)
	}

	var version uint16
	binary.Read(buf, binary.BigEndian, &version)

	var count uint64
	binary.Read(buf, binary.BigEndian, &count)

	// ---- Entries ----
	s.index = make(map[string]*RecordRef, count)
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
		s.setIndex(id, int(segIndex), offset, dataLen)
	}

	return nil
}
