package josefina

type Ql struct {
	Froms  []*From   `json:"froms"`
	Wheres []*Wheres `json:"wheres"`
}
