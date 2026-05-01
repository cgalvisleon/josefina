package jdb

import (
	"errors"
	"io"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

// Cursor holds iteration state for a single sequential scan of a Model.
// Create one via Model.NewCursor — do not share between goroutines.
type Cursor struct {
	model *Model
	keys  []string
	pos   int
}

/**
* NewCursor: Returns a cursor for sequential iteration over the model.
* asc controls sort direction; offset and limit mirror Limit() semantics
* (limit 0 = no limit). The key snapshot is taken once at creation —
* records written after this point are not visible to this cursor.
* @param asc bool, offset int, limit int
* @return *Cursor, error
**/
func (s *Model) NewCursor(asc bool, offset, limit int) (*Cursor, error) {
	st, exists := s.Source()
	if !exists {
		return nil, errors.New(msg.MSG_STORE_NOT_FOUND)
	}
	keys := st.Keys(asc, offset, limit)
	return &Cursor{model: s, keys: keys}, nil
}

/**
* Next: Advances the cursor and writes the current record into dest.
* Returns io.EOF when there are no more records, nil on success, or
* another error on store failure. Records deleted after cursor creation
* are skipped silently.
* @param dest *et.Json
* @return error
**/
func (c *Cursor) Next(dest *et.Json) error {
	for c.pos < len(c.keys) {
		idx := c.keys[c.pos]
		c.pos++
		item, exists, err := c.model.Current(idx)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		*dest = item
		return nil
	}
	return io.EOF
}

/**
* Reset: Rewinds the cursor to the beginning of the key snapshot.
**/
func (c *Cursor) Reset() {
	c.pos = 0
}

/**
* Len: Returns the total number of keys in the cursor snapshot.
* @return int
**/
func (c *Cursor) Len() int {
	return len(c.keys)
}

/**
* Pos: Returns the current position within the key snapshot.
* @return int
**/
func (c *Cursor) Pos() int {
	return c.pos
}

/**
* Close: Releases the key snapshot.
**/
func (c *Cursor) Close() {
	c.keys = nil
}
