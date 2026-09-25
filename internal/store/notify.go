package store

import (
	"slices"

	"github.com/cgalvisleon/et/et"
)

type Op string

const (
	OpInsert Op = "insert"
	OpUpdate Op = "update"
	OpDelete Op = "delete"
)

/**
* Change: Cambio aplicado al store, entregado a las funciones ancladas con Sync.
* Data es nil en OpDelete.
**/
type Change struct {
	Op   Op     `json:"op"`
	ID   string `json:"id"`
	Data []byte `json:"data"`
	LSN  uint64 `json:"lsn"`
}

/**
* ToJson: Retorna el cambio como JSON.
* @return et.Json
**/
func (s *Change) ToJson() et.Json {
	return et.Json{
		"op":   s.Op,
		"id":   s.ID,
		"data": string(s.Data),
		"lsn":  s.LSN,
	}
}

/**
* Sync: Ancla fn para recibir cada Insert, Update y Delete aplicado, incluidos los
* que llegan por ApplyWalEntry. fn se llama bajo writeMu, justo después de escribir
* el log y el índice, así que los cambios llegan exactamente en orden de LSN.
* fn debe ser rápida, no debe escribir en este store (se bloquearía) y debe copiar
* Data si la guarda, porque el slice es del llamador.
* @param fn func(change Change)
**/
func (s *FileStore) Sync(fn func(change Change)) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.syncFns = append(s.syncFns, fn)
}

/**
* emitLocked: Entrega change a las funciones ancladas con Sync (requiere writeMu).
* @param change Change
**/
func (s *FileStore) emitLocked(change Change) {
	for _, fn := range s.syncFns {
		fn(change)
	}
}

/**
* Stats: Retorna el tamaño del store y cuántas operaciones están en ejecución.
* Las escrituras en espera de su turno también cuentan como en ejecución.
* @return et.Json
**/
func (s *FileStore) Stats() et.Json {
	return et.Json{
		"count":       s.countIndex(),
		"size":        s.Size.count(),
		"wal":         s.WAL.count(),
		"tomb_stones": s.TombStones.count(),
		"reading":     s.reading.count(),
		"inserting":   s.inserting.count(),
		"updating":    s.updating.count(),
		"deleting":    s.deleting.count(),
	}
}

/**
* OnStats: Ancla fn para recibir Stats cada vez que cambian las operaciones en
* ejecución. fn se llama desde una sola goroutine de fondo; si hay varios cambios
* seguidos se entregan agrupados en el último estado, sin frenar las operaciones.
* @param fn func(stats et.Json)
**/
func (s *FileStore) OnStats(fn func(stats et.Json)) {
	s.statsMu.Lock()
	s.statsFns = append(s.statsFns, fn)
	s.statsMu.Unlock()

	s.statsStart.Do(func() {
		go s.statsLoop()
	})
	s.notifyStats()
}

/**
* statsLoop: Entrega Stats a las funciones ancladas con OnStats hasta que se cierre el store.
**/
func (s *FileStore) statsLoop() {
	for {
		select {
		case <-s.statsDone:
			return
		case <-s.statsCh:
			stats := s.Stats()
			s.statsMu.Lock()
			fns := slices.Clone(s.statsFns)
			s.statsMu.Unlock()
			for _, fn := range fns {
				fn(stats)
			}
		}
	}
}

/**
* notifyStats: Avisa a statsLoop que Stats cambió, sin bloquear.
**/
func (s *FileStore) notifyStats() {
	select {
	case s.statsCh <- struct{}{}:
	default:
	}
}

/**
* stopStats: Detiene statsLoop.
**/
func (s *FileStore) stopStats() {
	s.statsStop.Do(func() {
		close(s.statsDone)
	})
}

/**
* track: Cuenta una operación en ejecución; la función retornada la descuenta.
* Uso: defer s.track(&s.inserting)()
* @param c *counter[int]
* @return func()
**/
func (s *FileStore) track(c *counter[int]) func() {
	c.inc()
	s.notifyStats()
	return func() {
		c.dec()
		s.notifyStats()
	}
}
