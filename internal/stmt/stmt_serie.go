package stmt

type CreateSerieStmt struct {
	Tag    string
	Format string
	Value  int
}

func (CreateSerieStmt) stmt() {}

type SetSerieStmt struct {
	Tag   string
	Value int
}

func (SetSerieStmt) stmt() {}

type GetSerieStmt struct {
	Tag string
}

func (GetSerieStmt) stmt() {}

type DropSerieStmt struct {
	Tag string
}

func (DropSerieStmt) stmt() {}
