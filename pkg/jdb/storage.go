package jdb

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/logs"
)

/**
* save: Save the current state of the store
* @params src []byte, path string, fileName string
* @return error
 */
func save(src []byte, path, fileName string) error {
	path = filepath.Join(path, fileName)
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

	logs.Log(packageName, "saved:", path)

	return nil
}

/**
* load
* Load the store state from disk
* @return bool, error
 */
func load(path, fileName string, v any) (bool, error) {
	path = filepath.Join(path, fileName)
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

	err = json.Unmarshal(data, v)
	if err != nil {
		return false, err
	}

	return true, nil
}
