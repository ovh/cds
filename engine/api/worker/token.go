package worker

import (
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Token describes tokens used by worker to access the API
// on behalf of a group.
type Token struct {
	GroupID    int64          `json:"group_id"`
	Token      string         `json:"token"`
	Expiration sdk.Expiration `json:"expiration"`
	Created    time.Time      `json:"created"`
}

// GenerateToken generate a random 64bytes hexadecimal string
func GenerateToken() (string, error) {
	size := 64
	bs := make([]byte, size)
	_, err := rand.Read(bs)
	if err != nil {
		log.Critical("GenerateToken: rand.Read failed: %s\n", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	key := []byte(str)[0:size]

	log.Debug("GenerateToken: new generated id: %s\n", key)
	return string(key), nil

}

// InsertToken inserts a new token in database
func InsertToken(db gorp.SqlExecutor, groupID int64, token string, e sdk.Expiration) error {
	query := `INSERT INTO token (group_id, token, expiration, created) VALUES ($1, $2, $3, current_timestamp)`

	hasher := sha512.New()
	hashedToken := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(token)))

	_, err := db.Exec(query, groupID, hashedToken, int(e))
	if err != nil {
		return err
	}

	return nil
}

// LoadToken fetch token infos from database
func LoadToken(db gorp.SqlExecutor, token string) (*Token, error) {
	query := `SELECT group_id, expiration, created FROM token
		WHERE token = $1`

	hasher := sha512.New()
	hashed := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(token)))

	var t Token
	var exp int
	err := db.QueryRow(query, hashed).Scan(&t.GroupID, &exp, &t.Created)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrInvalidToken
		}
		return nil, err
	}
	t.Token = token
	t.Expiration = sdk.Expiration(exp)

	return &t, nil
}
