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
* ToJson: Returns the WAL entry as a JSON object.
* @return et.Json
**/
func (e *WalEntry) ToJson() et.Json {
	return et.Json{
		"lsn":    e.LSN,
		"id":     e.ID,
		"data":   string(e.Data),
		"status": e.Status,
	}
}

/**
* ApplyWalEntry: Writes a WAL entry received from the leader into this store.
* Bypasses the ReadOnly check — only the replication path may call this.
* Preserves the original LSN so leader and follower share the same sequence.
* @param entry WalEntry
* @return error
**/
func (s *FileStore) ApplyWalEntry(entry WalEntry) error {
	// Same as Put/Delete: log append and index update under one writeMu hold.
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	ref, err := s.appendRecordLocked(entry.LSN, entry.ID, entry.Data, entry.Status)
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

	return nil
}

/**
* WalSince: Returns all WAL entries with LSN strictly greater than since,
* in write order. Followers call this with their last acknowledged LSN
* to receive only the entries they are missing.
* @param since uint64
* @return []WalEntry, error
**/
func (s *FileStore) WalSince(since uint64) ([]WalEntry, error) {
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
