package store

import (
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/logs"
)

var storeCallsMap = map[string]*uint64{
	"put":     new(uint64),
	"get":     new(uint64),
	"delete":  new(uint64),
	"iterate": new(uint64),
}

func logMetrics() {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for range t.C {
		for tag, ptr := range storeCallsMap {
			calls := atomic.SwapUint64(ptr, 0)
			logs.Logf("metrics", "tag:%s calls/sec: %d", tag, calls)
		}
	}
}
