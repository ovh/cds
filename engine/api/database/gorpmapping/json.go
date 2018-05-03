package gorpmapping

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

type jsonTag struct {
	dbColumn string
	field    string
}

var mapJSONTagsPerType = struct {
	mutex sync.RWMutex
	data  map[string][]jsonTag
}{
	data: map[string][]jsonTag{},
}

func registerJSONColumns(m TableMapping) {
	f := interfaceToValue(m.Target)
	tags := []jsonTag{}
	for i := 0; i < f.NumField(); i++ {
		if !f.Field(i).CanInterface() {
			continue
		}
		tag := f.Type().Field(i).Tag.Get("dbjson")
		if tag != "" {
			tags = append(tags, jsonTag{
				dbColumn: tag,
				field:    f.Type().Field(i).Name,
			})
		}
	}
	if len(tags) > 0 {
		mapJSONTagsPerType.mutex.Lock()
		mapJSONTagsPerType.data[f.Type().PkgPath()+"/"+f.Type().Name()] = tags
		mapJSONTagsPerType.mutex.Unlock()
	}
}

//JSONToNullString returns a valid sql.NullString with json-marshalled i
func JSONToNullString(i interface{}) (sql.NullString, error) {
	if i == nil {
		return sql.NullString{Valid: false}, nil
	}
	b, err := json.Marshal(i)
	if err != nil {
		return sql.NullString{Valid: false}, err
	}
	return sql.NullString{Valid: true, String: string(b)}, nil
}

//JSONNullString sets the holder with unmarshalled sql.NullString
func JSONNullString(s sql.NullString, holder interface{}) error {
	if !s.Valid {
		return nil
	}
	return json.Unmarshal([]byte(s.String), holder)
}

// UpdateJSONFields parse the type to discover dbjson tags and persist then in the gorp entity
func UpdateJSONFields(db gorp.SqlExecutor, i interface{}) error {
	debug("UpdateJSONFields - BEGIN")
	defer debug("UpdateJSONFields - END")

	f := interfaceToValue(i)

	typeName := f.Type().PkgPath() + "/" + f.Type().Name()

	mapJSONTagsPerType.mutex.RLock()
	tags, has := mapJSONTagsPerType.data[typeName]
	mapJSONTagsPerType.mutex.RUnlock()

	if !has {
		return fmt.Errorf("UpdateJSONFields> %s json columns mapping not found", f.Type())
	}

	values := map[string]interface{}{}
	for _, tag := range tags {
		val := f.FieldByName(tag.field)
		if !val.CanInterface() || val.Interface() == nil {
			continue
		}
		val = interfaceToValue(val.Interface())
		values[tag.dbColumn] = val.Interface()
	}

	//find the tablemapping
	mapMappingPerType.mutex.RLock()
	mapping, has := mapMappingPerType.data[typeName]
	mapMappingPerType.mutex.RUnlock()
	if !has {
		return fmt.Errorf("UpdateJSONFields> %s mapping not found", f.Type())
	}

	//search the key values
	keys := mapping.Keys
	keyValues := []interface{}{}
	for column, field := range keys {
		debug("searching key col=%s field=%s", column, field)
		keyValues = append(keyValues, f.FieldByName(field).Interface())
	}

	//prapare the query
	query := bytes.Buffer{}
	args := []interface{}{}
	query.WriteString("update ")
	query.WriteString(mapping.Name)
	query.WriteString(" set ")
	var x int
	var firstPosition = true
	for c, v := range values {
		if !firstPosition {
			query.WriteString(", ")
		}
		query.WriteString(c)
		query.WriteString(" = ")
		query.WriteString(fmt.Sprintf("$%d", x+1))
		x++

		btes, err := json.Marshal(v)
		if err != nil {
			return err
		}
		args = append(args, btes)
		firstPosition = false
	}
	query.WriteString(" where ")
	firstPosition = true
	for col, field := range keys {
		if !firstPosition {
			query.WriteString(" and ")
		}
		query.WriteString(col)
		query.WriteString(" = ")
		query.WriteString(fmt.Sprintf("$%d ", x+1))
		x++

		args = append(args, f.FieldByName(field).Interface())
		firstPosition = false
	}

	_, err := db.Exec(query.String(), args...)
	return sdk.WrapError(err, "UpdateJSONFields>")
}
