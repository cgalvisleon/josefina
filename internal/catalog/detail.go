package catalog

type Detail struct {
	to              *Model            `json:"_"`
	Keys            map[string]string `json:"key"`
	Selects         []string          `json:"select"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
}

/**
* newDetail
* @param to *From, keys map[string]string, select []string, onDeleteCascade, onUpdateCascade bool
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
