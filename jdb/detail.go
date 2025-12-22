package jdb

type Detail struct {
	To              *Model            `json:"to"`
	As              string            `json:"as"`
	Keys            map[string]string `json:"keys"`
	Select          []string          `json:"select"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
	Page            int               `json:"page"`
	Rows            int               `json:"rows"`
}

/**
* newDetail
* @param to *Model, as string, keys map[string]string, selecs []string, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func newDetail(to *Model, as string, keys map[string]string, selecs []string, onDeleteCascade, onUpdateCascade bool) *Detail {
	return &Detail{
		To:              to,
		As:              as,
		Keys:            keys,
		Select:          selecs,
		OnDeleteCascade: onDeleteCascade,
		OnUpdateCascade: onUpdateCascade,
	}
}
