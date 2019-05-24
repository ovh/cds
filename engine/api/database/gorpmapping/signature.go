package gorpmapping

import (
	"encoding/json"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/symmecrypt/keyloader"
)

const (
	KeySignIdentifier = "db-sign"
)

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
	table, key, id, err := dbMappingPKey(i)
	if err != nil {
		return false, sdk.WithStack(fmt.Errorf("table %s primary key field %s not found: %v", table, key, err))
	}
	query := fmt.Sprintf("SELECT sig FROM %s WHERE %s = $1", table, key)
	var sig []byte
	if err := db.SelectOne(&sig, query, id); err != nil {
		log.Error("unable to check signature in table '%s' for key '%s' = '%v'", table, key, id)
		return false, sdk.WithStack(err)
	}
	return checkSign(i, sig)
}

func sign(i interface{}) ([]byte, error) {
	k, err := keyloader.LoadKey(KeySignIdentifier)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	var clearContent []byte
	if cannonical, ok := i.(Canonicaller); ok {
		clearContent, err = cannonical.Canonical()
		if err != nil {
			return nil, sdk.WithStack(fmt.Errorf("unable to marshal content: %v", err))
		}
	} else {
		clearContent, err = json.Marshal(i)
		if err != nil {
			return nil, sdk.WithStack(fmt.Errorf("unable to marshal content: %v", err))
		}
	}

	btes, err := k.Encrypt(clearContent)
	if err != nil {
		return nil, sdk.WithStack(fmt.Errorf("unable to encrypt content: %v", err))
	}

	return []byte(btes), nil
}

func checkSign(i interface{}, sig []byte) (bool, error) {
	k, err := keyloader.LoadKey(KeySignIdentifier)
	if err != nil {
		return false, sdk.WithStack(fmt.Errorf("unable to the load the key: %v", err))
	}

	var clearContent []byte
	if cannonical, ok := i.(Canonicaller); ok {
		clearContent, err = cannonical.Canonical()
		if err != nil {
			return false, sdk.WithStack(fmt.Errorf("unable to marshal content: %v", err))
		}
	} else {
		clearContent, err = json.Marshal(i)
		if err != nil {
			return false, sdk.WithStack(fmt.Errorf("unable to marshal content: %v", err))
		}
	}

	decryptedSig, err := k.Decrypt([]byte(sig))
	if err != nil {
		return false, sdk.WithStack(fmt.Errorf("unable to decrypt content: %v", err))
	}
	return string(clearContent) == string(decryptedSig), nil
}

func dbSign(db gorp.SqlExecutor, i interface{}) error {
	sig, err := sign(i)
	if err != nil {
		return err
	}

	table, key, id, err := dbMappingPKey(i)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("primary key field %s not found in: %v", table, err))
	}

	query := fmt.Sprintf("UPDATE %s SET sig = $2 WHERE %s = $1", table, key)
	res, err := db.Exec(query, id, sig)
	if err != nil {
		log.Error("error executing query %s with parameters %s, %s: %v", query, table, key, err)
		return sdk.WithStack(err)
	}

	n, _ := res.RowsAffected()
	if n != 1 {
		return sdk.WithStack(fmt.Errorf("%d number of rows affected (table=%s, key=%s, id=%v)", n, table, key, id))
	}
	return nil
}

type Canonicaller interface {
	Canonical() ([]byte, error)
}
