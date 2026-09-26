package store

import (
	"github.com/cgalvisleon/et/et"
)

type WalEntry struct {
	LSN    uint64 `json:"lsn"`
	ID     string `json:"id"`
	Data   []byte `json:"data"`
	Status byte   `json:"status"`
}

/**
* toJson: Retorna la entrada del WAL como JSON.
* @return et.Json
**/
func (e *WalEntry) toJson() et.Json {
	return et.Json{
		"lsn":    e.LSN,
		"id":     e.ID,
		"data":   string(e.Data),
		"status": e.Status,
	}
}

/**
* applyWalEntry: Aplica una entrada recibida del líder conservando su LSN (solo replicación).
* @param entry WalEntry
* @return error
**/
func (s *FileStore) applyWalEntry(entry WalEntry) error {
	// Igual que Insert/Delete: log e índice en un solo paso bajo writeMu.
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	ref, lsn, err := s.appendRecordLocked(entry.LSN, entry.ID, entry.Data, entry.Status)
	if err != nil {
		return err
	}

	var existed bool
	if entry.Status == Active {
		existed = s.setIndex(entry.ID, ref)
	} else {
		existed = s.deleteIndex(entry.ID)
	}
	if existed {
		s.TombStones.inc()
	}

	// Avisar a Sync con la operación que el cambio representa en este nodo.
	switch {
	case entry.Status == Active && existed:
		s.emitLocked(Change{Op: OpUpdate, ID: entry.ID, Data: entry.Data, LSN: lsn})
	case entry.Status == Active:
		s.emitLocked(Change{Op: OpInsert, ID: entry.ID, Data: entry.Data, LSN: lsn})
	case existed:
		s.emitLocked(Change{Op: OpDelete, ID: entry.ID, LSN: lsn})
	}

	return nil
}

/**
* walSince: Retorna, en orden de escritura, las entradas con LSN mayor que since.
* @param since uint64
* @return []WalEntry, error
**/
func (s *FileStore) walSince(since uint64) ([]WalEntry, error) {
	s.indexMu.RLock()
	segs := s.segments
	s.indexMu.RUnlock()

	var entries []WalEntry
	for _, seg := range segs {
		err := seg.scan(0, func(offset int64, h recordHeader, data []byte) error {
			if h.LSN > since {
				entries = append(entries, WalEntry{
					LSN:    h.LSN,
					ID:     h.ID,
					Data:   data,
					Status: h.Status,
				})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return entries, nil
}
