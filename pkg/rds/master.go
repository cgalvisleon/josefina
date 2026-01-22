package rds

type Master struct{}

/**
* Ping: Pings the master
* @param response *string
* @return error
**/
func (s *Master) Ping(require string, response *string) error {
	*response = "pong"
	return nil
}
