package store

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/iterate"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
)

var storeCallsMap = map[string]*uint64{
	"put":    new(uint64),
	"get":    new(uint64),
	"delete": new(uint64),
}

func (s *FileStore) logMetrics() {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for range t.C {
		for tag, ptr := range storeCallsMap {
			calls := atomic.SwapUint64(ptr, 0)
			s.Metrics[tag] = int64(calls)
			event.Emiter("store_metrics", et.Json{
				"timestamp": timezone.NowTime(),
				"database":  s.Database,
				"model":     s.Name,
				"tag":       tag,
				"calls":     calls,
			})
			if s.IsDebug {
				logs.Logf("metrics", "database:%s:model:%s:tag:%s:calls:%d:per/sec", s.Database, s.Name, tag, calls)
			}
		}
	}
}

/**
* metricStart
* @param tag string
**/
func (s *FileStore) metricStart(tag string) {
	tag = fmt.Sprintf("%s:%s:%s", s.Database, s.Name, tag)
	event.Emiter("store_metrics", et.Json{
		"timestamp": timezone.NowTime(),
		"database":  s.Database,
		"model":     s.Name,
		"tag":       tag,
	})
	iterate.Start(tag)
}

/**
* metricSegment
* @param tag string, msg string
* @return map[string]int64
**/
func (s *FileStore) metricSegment(tag, msg string) {
	tag = fmt.Sprintf("%s:%s:%s", s.Database, s.Name, tag)
	duration := iterate.Segment(tag, msg, s.IsDebug)
	value := int64(duration.Milliseconds())
	s.Metrics[tag] = value
	event.Emiter("store_metrics", et.Json{
		"timestamp":    timezone.NowTime(),
		"database":     s.Database,
		"model":        s.Name,
		"tag":          tag,
		"milliseconds": value,
	})
}

func (s *FileStore) metricEnd(tag, msg string) {
	tag = fmt.Sprintf("%s:%s:%s", s.Database, s.Name, tag)
	duration := iterate.End(tag, msg, s.IsDebug)
	value := int64(duration.Milliseconds())
	s.Metrics[tag] = value
	event.Emiter("store_metrics", et.Json{
		"timestamp": timezone.NowTime(),
		"database":  s.Database,
		"model":     s.Name,
		"tag":       tag,
		"metrics":   s.Metrics,
	})
}
