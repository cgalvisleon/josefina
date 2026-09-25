# Features

dejemos Put de scope local y creemos un Insert que solo inserte si el id no existe y usa put y un Update que solo actualice si existe y el valos binario es diferente o cambio y usa put un Delete que solo elimine si existe. El Insert y el Update reciven un id y el valor binario el Delete solo el ID; Tambien crea un Get que recibe un id y retorna el binario, si exite, y un error

/\*\*

- EvaluateValue
- @param conditions []\*et.Condition
- @return et.Items, error \**/
  func (s *FileStore) EvaluateValue(conditions []\*et.Condition, result et.Items) (et.Items, error) {
  s.ForEach(func(key string, data []byte) (bool, error) {
  item := et.Json{}
  if err := json.Unmarshal(data, &item); err != nil {
  return false, err
  }

      	ok := et.EvaluateValue(key, conditions)
      	if ok {
      		result.Add(item)
      	}

      	return ok, nil
      }, true, 0, 0)

      return result, nil

  }

/\*\*

- EvaluateObject
- @param conditions []\*et.Condition
- @return et.Items, error \**/
  func (s *FileStore) EvaluateObject(conditions []\*et.Condition, result et.Items) (et.Items, error) {
  s.ForEach(func(key string, data []byte) (bool, error) {
  item := et.Json{}
  if err := json.Unmarshal(data, &item); err != nil {
  return false, err
  }

      	ok := et.EvaluateObject(item, conditions)
      	if ok {
      		result.Add(item)
      	}

      	return true, nil
      }, true, 0, 0)

      return result, nil

  }
