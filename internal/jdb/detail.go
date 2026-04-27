package jdb

/**
* Detail: Represents a detail in the database
**/
type Detail struct {
	to              *Model            `json:"-"`                 // Target model
	Keys            map[string]string `json:"key"`               // Keys
	Selects         []string          `json:"select"`            // Selects
	OnDeleteCascade bool              `json:"on_delete_cascade"` // On delete cascade
	OnUpdateCascade bool              `json:"on_update_cascade"` // On update cascade
}

/**
* newDetail
* @param to *Model, keys map[string]string, select []string, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func newDetail(to *Model, keys map[string]string, selects []string, onDeleteCascade, onUpdateCascade bool) *Detail {
	return &Detail{
		to:              to,
		Keys:            keys,
		Selects:         selects,
		OnDeleteCascade: onDeleteCascade,
		OnUpdateCascade: onUpdateCascade,
	}
}
