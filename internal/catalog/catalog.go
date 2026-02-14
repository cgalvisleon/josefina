package catalog

var (
	address string
)

/**
* Load: Loads the cache
* @param addr string
* @return error
**/
func Load(addr string) error {
	address = addr

	return nil
}
