package server

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

func (s *Router) initEvents() {
	err := event.Subscribe(event.EVENT_WORK, s.eventWork)
	if err != nil {
		logs.Error(err)
	}
}

func (s *Router) eventWork(m event.Message) {
	work := m.Data

	logs.Log(s.Name, "eventWork", work.ToString())
}
