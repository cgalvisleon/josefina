package old

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* createSerie: Creates a series
* @param to, tag, format string, value int
* @return error
**/
func (s *Syn) createSerie(to, tag, format string, value int) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag":    tag,
		"format": format,
		"value":  value,
	}
	var reply string
	err := jrpc.CallRpc(to, "Syn.CreateSerie", data, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* CreateSerie: Creates a series
* @param require et.Json, response *string
* @return error
**/
func (s *Syn) CreateSerie(require et.Json, response *string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	tag := require.Str("tag")
	format := require.Str("format")
	value := require.Int("value")
	err := createSerie(tag, format, value)
	if err != nil {
		return err
	}

	return nil
}

/**
* dropSerie: Drops a series
* @param tag string
* @return error
**/
func (s *Syn) dropSerie(to, tag string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag": tag,
	}
	var reply string
	err := jrpc.CallRpc(to, "Syn.DropSerie", data, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* DropSerie: Drops a series
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) DropSerie(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	tag := require.Str("tag")
	err := dropSerie(tag)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* setSerie: Sets a series
* @param to, tag string, value int
* @return error
**/
func (s *Syn) setSerie(to, tag string, value int) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag":   tag,
		"value": value,
	}
	var reply string
	err := jrpc.CallRpc(to, "Syn.SetSerie", data, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetSerie: Sets a series
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) SetSerie(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	tag := require.Str("tag")
	value := require.Int("value")
	err := setSerie(tag, value)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* getSerie: Gets a series
* @param to, tag string
* @return error
**/
func (s *Syn) getSerie(to, tag string) (et.Json, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag": tag,
	}
	var reply et.Json
	err := jrpc.CallRpc(to, "Syn.GetSerie", data, &reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

/**
* GetSerie: Gets a series
* @param require et.Json, response *et.Json
* @return error
**/
func (s *Syn) GetSerie(require et.Json, response *et.Json) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	tag := require.Str("tag")
	result, err := getSerie(tag)
	if err != nil {
		return err
	}

	*response = result
	return nil
}
