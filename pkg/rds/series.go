package rds

var series *Model

/**
* initSeries: Initializes the series model
* @param db *DB
* @return error
**/
func initSeries(db *DB) error {
	var err error
	series, err = db.newModel("", "series", true, 1)
	if err != nil {
		return err
	}
	series.DefineAtrib("name", TpText, "")
	series.DefineAtrib("tag", TpText, "")
	series.DefineAtrib("value", TpInt, 0)
	series.DefineAtrib("format", TpText, "")
	series.DefinePrimaryKeys("name", "tag")
	if err := series.init(); err != nil {
		return err
	}

	return nil
}
