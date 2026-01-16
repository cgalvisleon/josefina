package store

import (
	"fmt"

	"github.com/cgalvisleon/et/iterate"
	"github.com/cgalvisleon/et/strs"
)

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
