package jdb

type Persist struct{}

var persist *Persist

func init() {
	persist = &Persist{}
}
