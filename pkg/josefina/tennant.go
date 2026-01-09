package josefina

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

const (
	packageName = "josefina"
	fileName    = "tennant.dat"
)

type Tennant struct {
	Name string         `json:"name"`
	Path string         `json:"path"`
	Dbs  map[string]*DB `json:"dbs"`
}

/**
* newTennant
* @param name string
* @return *Tennant, error
**/
func newTennant(path, name string) (*Tennant, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	return &Tennant{
		Name: name,
		Path: fmt.Sprintf("%s/%s", path, name),
		Dbs:  make(map[string]*DB),
	}, nil
}

/**
* Serialize
* @return []byte, error
**/
func (s *Tennant) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* Save
* Save the current state of the store
* @return error
 */
func (s *Tennant) save() error {
	src, err := s.serialize()
	if err != nil {
		return err
	}

	path := filepath.Join(s.Path, fileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(len(src)))

	_, err = f.Write(buf)
	if err != nil {
		return err
	}

	_, err = f.Write(src)
	if err != nil {
		return err
	}

	err = f.Sync()
	if err != nil {
		return err
	}

	logs.Log(packageName, "saved:tennant:", path)

	return nil
}

/**
* load
* Load the store state from disk
* @return bool, error
 */
func (s *Tennant) load() (bool, error) {
	path := filepath.Join(s.Path, fileName)
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	// Read the length prefix
	buf := make([]byte, 4)
	_, err = f.Read(buf)
	if err != nil {
		return false, err
	}

	length := binary.BigEndian.Uint32(buf)
	data := make([]byte, length)

	_, err = f.Read(data)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(data, s)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* init
* Initialize the store
* @return error
 */
func (s *Tennant) init() error {
	exist, err := s.load()
	if err != nil {
		return err
	}

	if !exist {
		return s.save()
	}

	return nil
}
