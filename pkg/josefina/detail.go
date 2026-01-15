package josefina

type Detail struct {
	From            *From             `json:"from"`
	To              *From             `json:"to"`
	Keys            map[string]string `json:"keys"`
	Select          []string          `json:"select"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
}

/**
* newDetail
* @param from *Model, to *Model, keys map[string]string, selecs []string, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func newDetail(from, to *Model, keys map[string]string, selecs []string, onDeleteCascade, onUpdateCascade bool) *Detail {
	return &Detail{
		From:            from.From,
		To:              to.From,
		Keys:            keys,
		Select:          selecs,
		OnDeleteCascade: onDeleteCascade,
		OnUpdateCascade: onUpdateCascade,
	}
}
