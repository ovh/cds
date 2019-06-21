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

	return btes, nil
}

// CheckSignature return true if a given signature is valid for given object.
func CheckSignature(i interface{}, sig []byte) (bool, error) {
	k, err := keyloader.LoadKey(KeySignIdentifier)
	if err != nil {
		return false, sdk.WrapError(err, "unable to the load the key")
	}

	var clearContent []byte
	if cannonical, ok := i.(Canonicaller); ok {
		clearContent, err = cannonical.Canonical()
		if err != nil {
			return false, sdk.WrapError(err, "unable to marshal content")
		}
	} else {
		clearContent, err = json.Marshal(i)
		if err != nil {
			return false, sdk.WrapError(err, "unable to marshal content")
		}
	}

	decryptedSig, err := k.Decrypt(sig)
	if err != nil {
		return false, sdk.WrapError(err, "unable to decrypt content")
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
