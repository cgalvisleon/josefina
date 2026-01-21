package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
)

/**
* Compact
* @return error
**/
func (s *FileStore) Compact() error {
	s.indexMu.RLock()
	keys := make([]string, 0)
	indexCopy := make(map[string]*RecordRef, len(s.index))
	for k, v := range s.index {
		keys = append(keys, k)
		indexCopy[k] = v
	}
	s.indexMu.RUnlock()

	// Orden determinista
	sort.Strings(keys)

	// Directorio temporal
	name := fmt.Sprintf("segments-%s.tmp", s.Name)
	tmpDir := filepath.Join(s.PathCompact, name)
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}

	var (
		newSegments []*segment
		current     *segment
	)

	createSegment := func() error {
		name := fmt.Sprintf("segment-%06d.dat", len(newSegments)+1)
		path := filepath.Join(tmpDir, name)

		fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return err
		}

		current = newSegment(fd, 0, name)
		newSegments = append(newSegments, current)
		return nil
	}

	if err := createSegment(); err != nil {
		return err
	}

	newIndex := make(map[string]*RecordRef, len(indexCopy))

	n := 0
	for _, id := range keys {
		ref := indexCopy[id]
		oldSeg := s.segments[ref.segment]

		// Leer header real
		fixed := make([]byte, fixedHeaderSize)
		if _, err := oldSeg.ReadAt(fixed, ref.offset); err != nil {
			return err
		}

		idLen := getUint16(fixed[8:10])
		payloadOffset := ref.offset + int64(fixedHeaderSize+idLen)

		data := make([]byte, ref.length)
		if ref.length > 0 {
			if _, err := oldSeg.ReadAt(data, payloadOffset); err != nil {
				return err
			}
		}

		// Rotar segmento si es necesario
		recordSize := int64(fixedHeaderSize) + int64(idLen) + int64(len(data))
		if current.size+recordSize > s.MaxSegment {
			if err := createSegment(); err != nil {
				return err
			}
		}

		newRef, err := current.WriteRecord(id, data, Active)
		if err != nil {
			return err
		}
		newRef.segment = len(newSegments) - 1
		newIndex[id] = newRef
		isDebug := envar.GetBool("DEBUG", false)
		if isDebug {
			logs.Log(packageName, "compacted:", s.Path, ":", s.Name, ":ID:", id, ":segment:", newRef.segment, ":offset:", newRef.offset, ":size:", newRef.length)
		}

		n++
	}

	// Swap atómico
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	for _, seg := range s.segments {
		seg.file.Close()
	}

	oldDir := filepath.Join(s.Path, "segments.old")
	os.RemoveAll(oldDir)

	if err := os.Rename(s.PathSegments, oldDir); err != nil {
		return err
	}
	if err := os.Rename(tmpDir, s.PathSegments); err != nil {
		return err
	}

	// Activar nuevos segmentos
	s.indexMu.Lock()
	s.index = newIndex
	s.keys = keys
	s.segments = newSegments
	s.active = newSegments[len(newSegments)-1]
	s.TombStones = 0
	s.indexMu.Unlock()

	return nil
}
