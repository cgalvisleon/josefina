package jdb

type Froms struct {
	Model *Model `json:"model"`
	As    string `json:"as"`
}

/**
* newFrom
* @param model *Model, as string
* @return *Froms
**/
func newFrom(model *Model, as string) *Froms {
	if as == "" {
		as = model.Name
	}

	return &Froms{
		Model: model,
		As:    as,
	}
}

/**
* FindField
* @param name string
* @return *Field
**/
func (s *Froms) FindField(name string) *Field {
	result := s.Model.FindField(name)
	if result != nil {
		result.From = s
	}

	return result
}
