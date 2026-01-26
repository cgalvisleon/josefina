package jdb

import "github.com/cgalvisleon/et/et"

type Jql struct {
	Token string    `json:"token"`
	Jqls  []et.Json `json:"jqls"`
}

func JqlIsExisted(to *From, field, key string) (bool, error) {
	return false, nil
}
