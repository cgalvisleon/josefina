package stmt

type Stmt interface {
	stmt()
}

type CreateUserStmt struct {
	Username string
	Password string
}

func (CreateUserStmt) stmt() {}

type GetUserStmt struct {
	Username string
	Password string
}

func (GetUserStmt) stmt() {}

type DropUserStmt struct {
	Username string
	Password string
}

func (DropUserStmt) stmt() {}

type ChangePasswordStmt struct {
	Username    string
	OldPassword string
	NewPassword string
}

func (ChangePasswordStmt) stmt() {}

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
