package store

import (
	"encoding/binary"
	"errors"
	"io"

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
	ref, err := s.appendRecordAt(entry.LSN, entry.ID, entry.Data, entry.Status)
	if err != nil {
		return err
	}

	s.indexMu.Lock()
	if entry.Status == Active {
		if _, exists := s.index[entry.ID]; exists {
			s.TombStones++
		}
		s.index[entry.ID] = ref
	} else {
		if _, exists := s.index[entry.ID]; exists {
			s.TombStones++
			s.deleteIndex(entry.ID)
		}
	}
	s.indexMu.Unlock()

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
		offset := int64(0)
		for {
			// Layout: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1]
			fixed := make([]byte, fixedHeaderSize)
			n, err := seg.ReadAt(fixed, offset)
			if err != nil {
				if errors.Is(err, io.EOF) || n < len(fixed) {
					break
				}
				return nil, err
			}

			lsn := binary.BigEndian.Uint64(fixed[0:8])
			dataLen := getUint32(fixed[8:12])
			crcStored := getUint32(fixed[12:16])
			idLen := getUint16(fixed[16:18])

			if idLen == 0 || idLen > maxIdLen {
				break // corrupción → parar seguro
			}

			idBytes := make([]byte, idLen)
			if _, err := seg.ReadAt(idBytes, offset+18); err != nil {
				break
			}

			statusByte := make([]byte, 1)
			if _, err := seg.ReadAt(statusByte, offset+18+int64(idLen)); err != nil {
				break
			}

			var data []byte
			if dataLen > 0 {
				data = make([]byte, dataLen)
				if _, err := seg.ReadAt(data, offset+18+int64(idLen)+1); err != nil {
					break
				}
				if checksum(data) != crcStored {
					break // registro corrupto → detener el escaneo
				}
			}

			if lsn > since {
				entries = append(entries, WalEntry{
					LSN:    lsn,
					ID:     string(idBytes),
					Data:   data,
					Status: statusByte[0],
				})
			}

			offset += int64(fixedHeaderSize) + int64(idLen) + int64(dataLen)
		}
	}

	return entries, nil
}
