package db

type Tx struct {
	Database string `json:"database"`
	Id       int    `json:"id"`
	Query    []byte `json:"query"`
}

type TxError struct {
	Database string `json:"database"`
	Id       int    `json:"id"`
	Error    []byte `json:"error"`
}
