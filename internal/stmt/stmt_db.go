package stmt

type CreateDbStmt struct {
	Name string
}

func (CreateDbStmt) stmt() {}

type GetDbStmt struct {
	Name string
}

func (GetDbStmt) stmt() {}

type DropDbStmt struct {
	Name string
}

func (DropDbStmt) stmt() {}

type UseDbStmt struct {
	Name string
}

func (UseDbStmt) stmt() {}
