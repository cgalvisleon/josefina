package jdb

import "github.com/cgalvisleon/et/et"

type jqls struct {
	Token string    `json:"token"`
	Jqls  []et.Json `json:"jqls"`
}

func JqlIsExisted(to *From, field, key string) (bool, error) {
	return false, nil
}
