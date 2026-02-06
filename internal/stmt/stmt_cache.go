package stmt

type SetCacheStmt struct {
	Key      string
	Value    string
	Duration float64
}

func (SetCacheStmt) stmt() {}

type GetCacheStmt struct {
	Key string
}

func (GetCacheStmt) stmt() {}

type DelCacheStmt struct {
	Key string
}

func (DelCacheStmt) stmt() {}

type ExistCacheStmt struct {
	Key string
}

func (ExistCacheStmt) stmt() {}
