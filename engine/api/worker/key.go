package worker

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadUserKey retrieves a user key in database
// /!\ DEPRECATED
func LoadUserKey(db gorp.SqlExecutor, key string) (int64, sdk.Expiration, error) {
	query := `SELECT user_id, expiry FROM user_key WHERE user_key = $1`

	hasher := sha512.New()
	hashedKey := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(key)))

	var e sdk.Expiration
	var userID int64
	err := db.QueryRow(query, hashedKey).Scan(&userID, &e)
	if err != nil {
		return 0, e, err
	}
	return userID, e, nil
}

// DeleteUserKey remove a key from database
// /!\ DEPRECATED
func DeleteUserKey(db database.Executer, key string) error {
	query := `DELETE FROM user_key WHERE user_key = $1`

	hasher := sha512.New()
	hashedKey := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(key)))

	_, err := db.Exec(query, hashedKey)
	if err != nil {
		return err
	}

	return nil
}

// InsertUserKey inserts a new user key in database
// /!\ DEPRECATED
func InsertUserKey(db gorp.SqlExecutor, userID int64, key string, e sdk.Expiration) error {
	query := `INSERT INTO user_key (user_id, user_key, expiry) VALUES ($1, $2, $3)`

	hasher := sha512.New()
	hashedKey := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(key)))

	_, err := db.Exec(query, userID, hashedKey, int(e))
	if err != nil {
		return err
	}

	return nil
}

// GenerateKey Generate key for worker
// /!\ DEPRECATED
func GenerateKey() (string, error) {
	size := 64
	bs := make([]byte, size)
	_, err := rand.Read(bs)
	if err != nil {
		log.Critical("generateKey: rand.Read failed: %s\n", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	key := []byte(str)[0:size]

	log.Debug("generateKey: new generated id: %s\n", key)
	return string(key), nil
}
