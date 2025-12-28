package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/logs"
)

/**
* createSnapshot
* @return error
**/
func (s *FileStore) createSnapshot() error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	path := filepath.Join(s.PathSnapshot, "state.snap")
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
		if s.IsDebug {
			logs.Log(packegeName, "snapshot:", s.Database, ":", s.Name, ":ID:", id, "seg:", ref.segment, ":offset:", ref.offset, ":len:", ref.length)
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
* FlushSnapshot
* @return
**/
func (s *FileStore) FlushSnapshot() {
	if s.SnapshotEvery > 0 {
		s.WritesSinceSnapshot++
		if s.WritesSinceSnapshot >= s.SnapshotEvery {
			s.WritesSinceSnapshot = 0
			go func() {
				s.createSnapshot()
			}()
		}
	}

	n := len(s.index)
	threshold := int(float64(n) * 0.1) // 10% del tamaño del índice
	if s.TombStones > threshold {
		go func() {
			s.Compact()
		}()
	}
}

/**
* tryLoadSnapshot
* @return bool, error
**/
func (s *FileStore) tryLoadSnapshot() (bool, error) {
	path := filepath.Join(s.PathSnapshot, "state.snap")

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
	s.index = make(map[string]*recordRef, count)

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

	return true, nil
}
