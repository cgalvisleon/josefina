package stmt

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
