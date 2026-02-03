package stmt

type Stmt interface {
	stmt()
}

type CreateUserStmt struct {
	Username string
	Password string
}

func (CreateUserStmt) stmt() {}

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
