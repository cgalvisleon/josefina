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
* snapshotPath: Retorna la ruta del archivo de snapshot.
* @return string
**/
func (s *FileStore) snapshotPath() string {
	return filepath.Join(s.PathSnapshot, fmt.Sprintf("state-%s.snap", s.Name))
}

/**
* removeSnapshot: Borra el snapshot para que el próximo Open reconstruya todo el índice.
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
* createSnapshot: Guarda en el snapshot el índice de los segmentos cerrados y el WAL.
* @return error
**/
func (s *FileStore) createSnapshot() error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	path := s.snapshotPath()
	tmp := path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// ---- Entradas ----
	// Solo los registros fuera del segmento activo; el contador del encabezado
	// debe coincidir con las entradas escritas.
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

	// ---- Encabezado ----
	// La versión 2 agrega el WAL; la versión 1 (sin WAL) se sigue leyendo.
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

	// reemplazo atómico
	return os.Rename(tmp, path)
}

/**
* tryLoadSnapshot: Carga el snapshot si existe y es válido; retorna false si no existe o está corrupto.
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

	// validar CRC
	payload := data[:len(data)-4]
	storedCRC := getUint32(data[len(data)-4:])
	if checksum(payload) != storedCRC {
		return false, errors.New(msg.MSG_SNAPSHOT_CORRUPTED)
	}

	buf := bytes.NewReader(payload)

	// ---- Encabezado ----
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

	// La versión 1 no trae WAL: queda en 0 y se recupera al reconstruir el segmento activo.
	var wal uint64
	if version >= 2 {
		if err := binary.Read(buf, binary.BigEndian, &wal); err != nil {
			return false, err
		}
	}

	// ---- Entradas ----
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	s.index = make(map[string]*recordRef, count)
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
		s.setIndexLocked(id, &recordRef{segment: int(segIndex), offset: offset, length: dataLen})
	}

	// Restaurar el WAL del snapshot para que no retroceda si el segmento activo está casi vacío.
	s.WAL.setMax(wal)

	return true, nil
}
