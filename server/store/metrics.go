package store

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/iterate"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
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
			if s.IsDebug {
				path := strs.ReplaceAll(s.Path, []string{"/"}, ":")
				logs.Logf("metrics", "path:%s:model:%s:tag:%s:calls:%d:per/sec", path, s.Name, tag, calls)
			}
		}
	}
}

/**
* metricStart
* @param tag string
**/
func (s *FileStore) metricStart(tag string) {
	path := strs.ReplaceAll(s.Path, []string{"/"}, ":")
	tag = fmt.Sprintf("%s:%s:%s", path, s.Name, tag)
	iterate.Start(tag)
}

/**
* metricSegment
* @param tag string, msg string
* @return map[string]int64
**/
func (s *FileStore) metricSegment(tag, msg string) {
	path := strs.ReplaceAll(s.Path, []string{"/"}, ":")
	tag = fmt.Sprintf("%s:%s:%s", path, s.Name, tag)
	duration := iterate.Segment(tag, msg, s.IsDebug)
	value := int64(duration.Milliseconds())
	s.Metrics[tag] = value
}

func (s *FileStore) metricEnd(tag, msg string) {
	path := strs.ReplaceAll(s.Path, []string{"/"}, ":")
	tag = fmt.Sprintf("%s:%s:%s", path, s.Name, tag)
	duration := iterate.End(tag, msg, s.IsDebug)
	value := int64(duration.Milliseconds())
	s.Metrics[tag] = value
}
