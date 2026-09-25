package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/msg"
)

/**
* snapshotPath: Returns the path of this store's snapshot file
* @return string
**/
func (s *FileStore) snapshotPath() string {
	return filepath.Join(s.PathSnapshot, fmt.Sprintf("state-%s.snap", s.Name))
}

/**
* removeSnapshot: Deletes the snapshot file so the next Open does a full rebuild.
* Used when the segment layout changes and the snapshot's refs no longer apply.
* @return error
**/
func (s *FileStore) removeSnapshot() error {
	err := os.Remove(s.snapshotPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

/**
* CreateSnapshot: Persists the current in-memory index to a snapshot file.
* Only records from segments other than the active one are included.
* The file is written atomically via a tmp-then-rename pattern.
* @return error
**/
func (s *FileStore) CreateSnapshot() error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	path := s.snapshotPath()
	tmp := path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// ---- Entries ----
	// Only records outside the active segment are stored; buildIndex() replays the
	// active one on load. The header count must match the entries actually written,
	// otherwise tryLoadSnapshot() reads past the end and rejects the file.
	entries := bytes.NewBuffer(nil)
	count := uint64(0)
	currentSegment := len(s.segments) - 1
	for id, ref := range s.index {
		if ref.segment == currentSegment {
			continue
		}
		idBytes := []byte(id)
		binary.Write(entries, binary.BigEndian, uint16(len(idBytes)))
		entries.Write(idBytes)
		binary.Write(entries, binary.BigEndian, uint32(ref.segment))
		binary.Write(entries, binary.BigEndian, ref.offset)
		binary.Write(entries, binary.BigEndian, ref.length)
		count++
		if s.isDebug {
			logs.Debug("snapshot:", s.Path, ":ID:", id, "seg:", ref.segment, ":offset:", ref.offset, ":len:", ref.length)
		}
	}

	// ---- Header ----
	// Version 2 adds the WAL field right after count; tryLoadSnapshot() branches
	// on version so snapshots written by older builds (version 1, no WAL) still load.
	buf := bytes.NewBuffer(nil)
	buf.WriteString("SNAP")
	binary.Write(buf, binary.BigEndian, uint16(2))
	binary.Write(buf, binary.BigEndian, count)
	binary.Write(buf, binary.BigEndian, s.WAL.count())
	buf.Write(entries.Bytes())

	// ---- CRC ----
	crc := checksum(buf.Bytes())
	if err := binary.Write(buf, binary.BigEndian, crc); err != nil {
		return err
	}

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
* tryLoadSnapshot: Loads the snapshot index if it exists and passes CRC validation.
* Returns (true, nil) when loaded, (false, nil) when absent, (false, err) when corrupt.
* @return bool, error
**/
func (s *FileStore) tryLoadSnapshot() (bool, error) {
	data, err := os.ReadFile(s.snapshotPath())
	if err != nil {
		return false, nil // snapshot opcional
	}

	if len(data) < 10 {
		return false, errors.New(msg.MSG_INVALID_SNAPSHOT)
	}

	// CRC check
	payload := data[:len(data)-4]
	storedCRC := getUint32(data[len(data)-4:])
	if checksum(payload) != storedCRC {
		return false, errors.New(msg.MSG_SNAPSHOT_CORRUPTED)
	}

	buf := bytes.NewReader(payload)

	// ---- Header ----
	magic := make([]byte, 4)
	buf.Read(magic)
	if string(magic) != "SNAP" {
		return false, errors.New(msg.MSG_INVALID_SNAPSHOT_MAGIC)
	}

	var version uint16
	if err := binary.Read(buf, binary.BigEndian, &version); err != nil {
		return false, err
	}
	if version != 1 && version != 2 {
		return false, errors.New(msg.MSG_INVALID_SNAPSHOT)
	}

	var count uint64
	if err := binary.Read(buf, binary.BigEndian, &count); err != nil {
		return false, err
	}

	// version 1 snapshots predate the WAL field: fall back to 0 and let the
	// caller's rebuild of the active segment recover whatever LSN it can see.
	var wal uint64
	if version >= 2 {
		if err := binary.Read(buf, binary.BigEndian, &wal); err != nil {
			return false, err
		}
	}

	// ---- Entries ----
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	s.index = make(map[string]*RecordRef, count)
	for i := uint64(0); i < count; i++ {
		var idLen uint16
		binary.Read(buf, binary.BigEndian, &idLen)

		idBytes := make([]byte, idLen)
		buf.Read(idBytes)

		var segIndex uint32
		var offset int64
		var dataLen uint32
		if err := binary.Read(buf, binary.BigEndian, &segIndex); err != nil {
			return false, err
		}
		if err := binary.Read(buf, binary.BigEndian, &offset); err != nil {
			return false, err
		}
		if err := binary.Read(buf, binary.BigEndian, &dataLen); err != nil {
			return false, err
		}

		id := string(idBytes)
		s.setIndexLocked(id, &RecordRef{segment: int(segIndex), offset: offset, length: dataLen})
	}

	// buildIndex() replays only the active segment right after this returns, so
	// restore WAL here or a fresh/near-empty active segment would make the LSN
	// counter regress below records already folded into this snapshot.
	s.WAL.setMax(wal)

	return true, nil
}
