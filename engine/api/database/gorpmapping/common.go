package gorpmapping

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ovh/symmecrypt"
	"github.com/ovh/symmecrypt/keyloader"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/sdk"
)

const (
	// ViolateUniqueKeyPGCode is the pg code when duplicating unique key
	ViolateUniqueKeyPGCode = "23505"

	// StringDataRightTruncation is raisedalue is too long for varchar.
	StringDataRightTruncation = "22001"
)

func Init(signatureKeys []keyloader.KeyConfig) error {

	// Push the keys in the keyloader
	keyloader.LoadKey

	return nil
}

// IDsToQueryString returns a comma separated list of given ids.
func IDsToQueryString(ids []int64) string {
	res := make([]string, len(ids))
	for i := range ids {
		res[i] = fmt.Sprintf("%d", ids[i])
	}
	return strings.Join(res, ",")
}

// Insert value in given db.
func Insert(db gorp.SqlExecutor, i interface{}) error {
	err := db.Insert(i)
	if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ViolateUniqueKeyPGCode:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		case StringDataRightTruncation:
			err = sdk.NewError(sdk.ErrConflict, e)
		}
	}
	return sdk.WithStack(err)
}

// Update value in given db.
func Update(db gorp.SqlExecutor, i interface{}) error {
	_, err := db.Update(i)
	if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ViolateUniqueKeyPGCode:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		case StringDataRightTruncation:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		}
	}
	return sdk.WithStack(err)
}

// Delete value in given db.
func Delete(db gorp.SqlExecutor, i interface{}) error {
	_, err := db.Delete(i)
	return sdk.WithStack(err)
}

const keySignIdentifier = "db-sign"

func InsertAndSign(db gorp.SqlExecutor, i interface{}) error {
	if err := Insert(db, i); err != nil {
		return err
	}
	return sdk.WithStack(dbSign(db, i))
}

func UpdatetAndSign(db gorp.SqlExecutor, i interface{}) error {
	if err := Update(db, i); err != nil {
		return err
	}
	return sdk.WithStack(dbSign(db, i))
}

func CheckSignature(db gorp.SqlExecutor, i interface{}) (bool, error) {
	k, err := keyloader.LoadKey(keySignIdentifier)
	if err != nil {
		return false, err
	}

	table, key, id := dbMappingPKey(i)
	if id == nil {
		return false, sdk.WithStack(fmt.Errorf("primary key field %s not found", table, key))
	}

	clearContent, err := json.Marshal(i)
	if err != nil {
		return false, sdk.WithStack(err)
	}

	query := fmt.Sprintf("SELECT sig FROM %s WHERE %s = $1", table, key)
	sig, err := db.SelectNullStr(query, id)
	if err != nil {
		return false, sdk.WithStack(err)
	}

	if !sig.Valid {
		return false, sdk.WithStack(errors.New("database signature not found"))
	}

	decryptedSig, err := k.Decrypt([]byte(sig.String))
	if err != nil {
		return false, sdk.WithStack(err)
	}

	return string(clearContent) == string(decryptedSig), nil
}

func dbMappingPKey(i interface{}) (string, string, interface{}) {
	mapping, has := getTabbleMapping(i)
	if !has {
		return "", "", sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}

	if len(mapping.Keys) > 0 {
		return "", "", sdk.WithStack(errors.New("multiple primary key not supported"))
	}

	val := reflect.ValueOf(i).Elem()
	var id interface{}
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		tag := typeField.Tag
		column := tag.Get("db")
		if column == mapping.Keys[0] {
			id = valueField.Interface()
			break
		}
	}
	return mapping.Name, mapping.Keys[0], id
}

func dbSign(db gorp.SqlExecutor, i interface{}) error {
	k, err := keyloader.LoadKey(keySignIdentifier)
	if err != nil {
		return sdk.WithStack(err)
	}

	table, key, id := dbMappingPKey(i)
	if id == nil {
		return sdk.WithStack(fmt.Errorf("primary key field %s not found", table, key))
	}

	b := new(bytes.Buffer)
	w := symmecrypt.NewWriter(b, k)
	jsonEncoder := json.NewEncoder(w)
	if err := jsonEncoder.Encode(i); err != nil {
		return sdk.WithStack(err)
	}
	if err := w.Close(); err != nil {
		return sdk.WithStack(err)
	}
	sig := b.Bytes()

	query := fmt.Sprintf("UPDATE %s SET sig = $2 WHERE %s = $1", table, key)
	res, err := db.Exec(query, id, sig)
	if err != nil {
		return sdk.WithStack(err)
	}

	n, _ := res.RowsAffected()
	if n != 1 {
		return sdk.WithStack(fmt.Errorf("%d number of rows affected", n))
	}
	return nil
}
