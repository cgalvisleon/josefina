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
* compactWriter: Escribe los segmentos nuevos de una compactación, rotando cuando se llenan.
**/
type compactWriter struct {
	dir        string
	maxSegment int64
	segments   []*segment
	current    *segment
	maxLSN     uint64
}

/**
* newSegment: Sella el segmento actual y abre el siguiente.
* @return error
**/
func (s *compactWriter) newSegment() error {
	if s.current != nil {
		if err := s.current.seal(); err != nil {
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
* write: Agrega un registro conservando su LSN original.
* @param lsn uint64, id string, data []byte, status byte
* @return *recordRef, error
**/
func (s *compactWriter) write(lsn uint64, id string, data []byte, status byte) (*recordRef, error) {
	recordSize := int64(fixedHeaderSize) + int64(len(id)) + int64(len(data))
	if s.current.size > 0 && s.current.size+recordSize > s.maxSegment {
		if err := s.newSegment(); err != nil {
			return nil, err
		}
	}

	ref, err := s.current.writeRecord(lsn, id, data, status)
	if err != nil {
		return nil, err
	}
	ref.segment = len(s.segments) - 1
	s.maxLSN = max(s.maxLSN, lsn)
	return ref, nil
}

/**
* size: Retorna los bytes escritos en todos los segmentos nuevos.
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
* discard: Cierra y borra los segmentos nuevos (compactación abortada).
**/
func (s *compactWriter) discard() {
	for _, seg := range s.segments {
		seg.close()
	}
	os.RemoveAll(s.dir)
}

/**
* compactTmpPath: Retorna el directorio temporal donde la compactación escribe los segmentos nuevos.
* @return string
**/
func (s *FileStore) compactTmpPath() string {
	return filepath.Join(s.PathCompact, fmt.Sprintf("segments-%s.tmp", s.Name))
}

/**
* oldSegmentsPath: Retorna el directorio al que la compactación mueve los segmentos viejos.
* Es hermano de s.Path: renombrar un directorio dentro de sí mismo falla.
* @return string
**/
func (s *FileStore) oldSegmentsPath() string {
	return filepath.Join(filepath.Dir(s.Path), s.Name+".segments.old")
}

/**
* compact: Reescribe solo los registros vivos en segmentos nuevos, en tres fases:
*  1. Bajo writeMu: copia del índice y punto de corte del log.
*  2. Sin locks: copia de los registros vivos.
*  3. Bajo writeMu e indexMu: aplica lo escrito después del corte e intercambia directorios.
* @return error
**/
func (s *FileStore) compact() error {
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
	tmpDir := s.compactTmpPath()
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

	newIndex := make(map[string]*recordRef, len(indexCopy))
	for _, id := range keys {
		ref := indexCopy[id]
		oldSeg := oldSegs[ref.segment]

		// Leer header real: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1]
		fixed := make([]byte, fixedHeaderSize)
		if _, err := oldSeg.readAt(fixed, ref.offset); err != nil {
			return err
		}

		lsn := binary.BigEndian.Uint64(fixed[0:8])
		idLen := getUint16(fixed[16:18])
		payloadOffset := ref.offset + int64(fixedHeaderSize+idLen)

		data := make([]byte, ref.length)
		if ref.length > 0 {
			if _, err := oldSeg.readAt(data, payloadOffset); err != nil {
				return err
			}
		}

		newRef, err := w.write(lsn, id, data, Active)
		if err != nil {
			return err
		}
		newIndex[id] = newRef
		if s.debug {
			logs.Debug("compacted:", s.Path, ":ID:", id, ":segment:", newRef.segment, ":offset:", newRef.offset, ":size:", newRef.length)
		}
	}

	// ---- Fase 3: ponerse al día y hacer el swap ----
	// Los lectores no usan indexMu mientras leen: fijan el segmento, y los
	// segmentos viejos se retiran en lugar de cerrarse.
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.indexMu.Lock()
	indexLocked := true
	defer func() {
		if indexLocked {
			s.indexMu.Unlock()
		}
	}()

	// Aplicar, en orden, lo escrito después del punto de corte.
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

	// Si el registro más reciente es un tombstone, se conserva para que el WAL no retroceda.
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

	if err := w.current.flush(); err != nil {
		return err
	}

	// El snapshot describe el layout viejo: se borra antes del intercambio.
	if err := s.removeSnapshot(); err != nil {
		return err
	}

	oldDir := s.oldSegmentsPath()
	os.RemoveAll(oldDir)

	if err := os.Rename(s.Path, oldDir); err != nil {
		return err
	}
	if err := os.Rename(tmpDir, s.Path); err != nil {
		// Restaurar el directorio original; el store sigue funcionando con él.
		if rbErr := os.Rename(oldDir, s.Path); rbErr != nil {
			return fmt.Errorf("compact: %w (rollback: %v)", err, rbErr)
		}
		return err
	}
	swapped = true

	// Retirar en vez de cerrar: los lectores en curso terminan antes de cerrar el archivo.
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

	// writeMu sigue tomado: ninguna escritura entra entre el intercambio y el nuevo snapshot.
	return s.createSnapshot()
}

/**
* lastRecordLocked: Retorna el encabezado del registro más reciente del log (requiere writeMu).
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
