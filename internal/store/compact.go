package store

import (
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/cgalvisleon/et/logs"
)

/**
* compactWriter: Writes records into the fresh segment files of a compaction,
* rotating to a new file when the current one is full.
**/
type compactWriter struct {
	dir        string
	maxSegment int64
	segments   []*segment
	current    *segment
	maxLSN     uint64
}

/**
* newSegment: Seals the current file and opens the next one
* @return error
**/
func (s *compactWriter) newSegment() error {
	if s.current != nil {
		if err := s.current.Seal(); err != nil {
			return err
		}
	}

	name := fmt.Sprintf("segment-%06d.dat", len(s.segments)+1)
	fd, err := os.OpenFile(filepath.Join(s.dir, name), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	s.current = newSegment(fd, 0, name)
	s.segments = append(s.segments, s.current)
	return nil
}

/**
* write: Appends a record keeping its original LSN
* @param lsn uint64, id string, data []byte, status byte
* @return *RecordRef, error
**/
func (s *compactWriter) write(lsn uint64, id string, data []byte, status byte) (*RecordRef, error) {
	recordSize := int64(fixedHeaderSize) + int64(len(id)) + int64(len(data))
	if s.current.size > 0 && s.current.size+recordSize > s.maxSegment {
		if err := s.newSegment(); err != nil {
			return nil, err
		}
	}

	ref, err := s.current.WriteRecord(lsn, id, data, status)
	if err != nil {
		return nil, err
	}
	ref.segment = len(s.segments) - 1
	s.maxLSN = max(s.maxLSN, lsn)
	return ref, nil
}

/**
* size: Returns the total bytes written across all new segments
* @return int64
**/
func (s *compactWriter) size() int64 {
	var result int64
	for _, seg := range s.segments {
		result += seg.size
	}
	return result
}

/**
* discard: Closes and deletes the new segment files (compaction aborted)
**/
func (s *compactWriter) discard() {
	for _, seg := range s.segments {
		seg.Close()
	}
	os.RemoveAll(s.dir)
}

/**
* Compact: Rewrites all live records into fresh segment files, removing tombstones.
* Runs in three phases so writers are only blocked at the start and at the swap:
*  1. Under writeMu, take a copy of the index and the log position (cut point).
*  2. Without locks, copy every live record into new segments in a temp dir.
*  3. Under writeMu + indexMu, replay whatever was written after the cut point
*     (Puts are copied, Deletes are written as tombstones), then swap directories.
* The snapshot file is removed during the swap (its refs point at the old layout)
* and rebuilt afterwards.
* @return error
**/
func (s *FileStore) Compact() error {
	s.compactMu.Lock()
	defer s.compactMu.Unlock()

	// ---- Fase 1: punto de corte ----
	s.writeMu.Lock()
	s.indexMu.RLock()
	indexCopy := maps.Clone(s.index)
	oldSegs := s.segments
	cutSeg := len(s.segments) - 1
	cutOffset := s.active.size
	s.indexMu.RUnlock()
	s.writeMu.Unlock()

	keys := slices.Sorted(maps.Keys(indexCopy))

	// ---- Fase 2: copiar registros vivos sin bloquear escrituras ----
	tmpDir := filepath.Join(s.PathCompact, fmt.Sprintf("segments-%s.tmp", s.Name))
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}

	w := &compactWriter{dir: tmpDir, maxSegment: s.MaxSegment}
	swapped := false
	defer func() {
		if !swapped {
			w.discard()
		}
	}()

	if err := w.newSegment(); err != nil {
		return err
	}

	newIndex := make(map[string]*RecordRef, len(indexCopy))
	for _, id := range keys {
		ref := indexCopy[id]
		oldSeg := oldSegs[ref.segment]

		// Leer header real: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1]
		fixed := make([]byte, fixedHeaderSize)
		if _, err := oldSeg.ReadAt(fixed, ref.offset); err != nil {
			return err
		}

		lsn := binary.BigEndian.Uint64(fixed[0:8])
		idLen := getUint16(fixed[16:18])
		payloadOffset := ref.offset + int64(fixedHeaderSize+idLen)

		data := make([]byte, ref.length)
		if ref.length > 0 {
			if _, err := oldSeg.ReadAt(data, payloadOffset); err != nil {
				return err
			}
		}

		newRef, err := w.write(lsn, id, data, Active)
		if err != nil {
			return err
		}
		newIndex[id] = newRef
		if s.isDebug {
			logs.Debug("compacted:", s.Path, ":ID:", id, ":segment:", newRef.segment, ":offset:", newRef.offset, ":size:", newRef.length)
		}
	}

	// ---- Fase 3: ponerse al día y hacer el swap ----
	// Readers never hold indexMu while reading from disk: they pin the segment
	// (acquire/release) and the old segments are retired below, not closed.
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.indexMu.Lock()
	indexLocked := true
	defer func() {
		if indexLocked {
			s.indexMu.Unlock()
		}
	}()

	// Replay everything appended after the cut point, in log order.
	tombStones := 0
	for i := cutSeg; i < len(s.segments); i++ {
		from := int64(0)
		if i == cutSeg {
			from = cutOffset
		}

		err := s.segments[i].scan(from, func(offset int64, h recordHeader, data []byte) error {
			_, existed := newIndex[h.ID]
			if h.Status == Deleted && !existed {
				return nil // nada que borrar en el layout nuevo
			}

			ref, err := w.write(h.LSN, h.ID, data, h.Status)
			if err != nil {
				return err
			}

			if h.Status == Active {
				newIndex[h.ID] = ref
			} else {
				delete(newIndex, h.ID)
			}
			if existed {
				tombStones++
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	// Tombstones are dropped, so if the newest record in the log is one, the new
	// layout would lose the highest LSN and a rebuild would regress the WAL.
	// Keep that last tombstone (it is a real delete of a key that is not live).
	if w.maxLSN < s.WAL.count() {
		h, found, err := s.lastRecordLocked()
		if err != nil {
			return err
		}
		if found && h.Status == Deleted {
			if _, err := w.write(h.LSN, h.ID, nil, Deleted); err != nil {
				return err
			}
		}
	}

	if err := w.current.Sync(); err != nil {
		return err
	}

	// The snapshot describes the old layout; drop it before the swap so a crash
	// in between falls back to a full rebuild instead of loading stale refs.
	if err := s.removeSnapshot(); err != nil {
		return err
	}

	// oldDir must be a sibling of s.Path, not a child of it — renaming a directory
	// into its own subtree fails with EINVAL ("invalid argument").
	oldDir := filepath.Join(filepath.Dir(s.Path), s.Name+".segments.old")
	os.RemoveAll(oldDir)

	if err := os.Rename(s.Path, oldDir); err != nil {
		return err
	}
	if err := os.Rename(tmpDir, s.Path); err != nil {
		// Put the old directory back; the store keeps running on it.
		if rbErr := os.Rename(oldDir, s.Path); rbErr != nil {
			return fmt.Errorf("compact: %w (rollback: %v)", err, rbErr)
		}
		return err
	}
	swapped = true

	// Retire instead of Close: a ForEach/Get that pinned an old segment keeps
	// reading it; its file closes when the last of those readers releases it.
	for _, seg := range s.segments {
		seg.retire()
	}
	os.RemoveAll(oldDir)

	// Activar nuevos segmentos
	s.index = newIndex
	s.segments = w.segments
	s.active = w.current
	s.TombStones.set(tombStones)
	s.Size.set(w.size())

	s.indexMu.Unlock()
	indexLocked = false

	// writeMu is still held, so no write can land between the swap and the new snapshot.
	return s.CreateSnapshot()
}

/**
* lastRecordLocked: Returns the header of the newest record in the log.
* Caller must hold writeMu (no append can move the end of the log meanwhile).
* @return recordHeader, bool, error
**/
func (s *FileStore) lastRecordLocked() (recordHeader, bool, error) {
	for i := len(s.segments) - 1; i >= 0; i-- {
		if s.segments[i].size == 0 {
			continue
		}

		var last recordHeader
		found := false
		err := s.segments[i].scan(0, func(offset int64, h recordHeader, data []byte) error {
			last, found = h, true
			return nil
		})
		if err != nil || found {
			return last, found, err
		}
	}

	return recordHeader{}, false, nil
}
