package jql

import "github.com/cgalvisleon/et/et"

type Cmd struct {
	host string
}

func toCmd(cmd et.Json) (*Cmd, error) {
	return &Cmd{}, nil
}

func (s *Cmd) run() (et.Items, error) {
	return et.Items{}, nil
}
