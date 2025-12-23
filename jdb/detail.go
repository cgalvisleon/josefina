package jdb

type Detail struct {
	To              *Model            `json:"to"`
	Keys            map[string]string `json:"keys"`
	Select          []string          `json:"select"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
}

/**
* newDetail
* @param to *Model, keys map[string]string, selecs []string, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func newDetail(to *Model, keys map[string]string, selecs []string, onDeleteCascade, onUpdateCascade bool) *Detail {
	return &Detail{
		To:              to,
		Keys:            keys,
		Select:          selecs,
		OnDeleteCascade: onDeleteCascade,
		OnUpdateCascade: onUpdateCascade,
	}
}

type TypeJoin string

const (
	TpJoin  TypeJoin = "join"
	TpLeft  TypeJoin = "left"
	TpRight TypeJoin = "right"
	TpFull  TypeJoin = "full"
)

type Joins struct {
	Type TypeJoin
	To   *Froms
	Keys map[string]string
}

/**
* newJoins
* @param tp TypeJoin, from *Froms, keys map[string]string
* @return *Joins
**/
func newJoins(tp TypeJoin, from *Froms, keys map[string]string) *Joins {
	return &Joins{
		Type: tp,
		To:   from,
		Keys: keys,
	}
}
