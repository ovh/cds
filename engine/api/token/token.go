package token

import (
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// GenerateToken generate a random 64bytes hexadecimal string
func GenerateToken() (string, error) {
	size := 64
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		log.Error("GenerateToken: rand.Read failed: %s", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	key := []byte(str)[0:size]

	log.Debug("GenerateToken: new generated id: %s", key)
	return string(key), nil
}

// InsertToken inserts a new token in database
func InsertToken(db gorp.SqlExecutor, groupID int64, token string, e sdk.Expiration, description, creator string) error {
	query := `INSERT INTO token (group_id, token, expiration, created, description, creator) VALUES ($1, $2, $3, $4, $5, $6)`

	hasher := sha512.New()
	hashedToken := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(token)))

	if _, err := db.Exec(query, groupID, hashedToken, int(e), time.Now(), description, creator); err != nil {
		return err
	}
	return nil
}

// CountToken returns nb token attached to a group
func CountToken(db gorp.SqlExecutor, groupID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM token WHERE group_id = $1`
	if err := db.QueryRow(query, groupID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// LoadToken fetch token infos from database
func LoadToken(db gorp.SqlExecutor, token string) (*sdk.Token, error) {
	query := `SELECT id, group_id, expiration, created, description, creator FROM token WHERE token = $1`

	hasher := sha512.New()
	hashed := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(token)))

	var t sdk.Token
	var exp int
	var description, creator sql.NullString
	if err := db.QueryRow(query, hashed).Scan(&t.ID, &t.GroupID, &exp, &t.Created, &description, &creator); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrInvalidToken
		}
		return nil, err
	}
	if description.Valid {
		t.Description = description.String
	}
	if creator.Valid {
		t.Creator = creator.String
	}
	t.Token = token
	t.Expiration = sdk.Expiration(exp)

	return &t, nil
}

// LoadTokenWithGroup fetch token infos with group infos from database
func LoadTokenWithGroup(db gorp.SqlExecutor, token string) (*sdk.Token, error) {
	query := `
	SELECT token.id, token.group_id, token.expiration, token.created, token.description, token.creator, "group".name
	FROM token
		JOIN "group" ON "group".id = token.group_id
	WHERE token = $1`

	hasher := sha512.New()
	hashed := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(token)))

	var t sdk.Token
	var description, creator sql.NullString
	if err := db.QueryRow(query, hashed).Scan(&t.ID, &t.GroupID, &t.Expiration, &t.Created, &description, &creator, &t.GroupName); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrInvalidToken
		}
		return nil, err
	}
	if description.Valid {
		t.Description = description.String
	}
	if creator.Valid {
		t.Creator = creator.String
	}
	t.Token = token

	return &t, nil
}

// Delete delete a token in db given his value
func Delete(db gorp.SqlExecutor, tokenID int64) error {
	_, err := db.Exec("DELETE FROM token WHERE id = $1", tokenID)
	return sdk.WrapError(err, "DeleteToken> Cannot delete token %d", tokenID)
}

// LoadTokens load all tokens linked that a user can see
func LoadTokens(db gorp.SqlExecutor, userID int64) ([]sdk.Token, error) {
	tokens := []sdk.Token{}

	query := `
		SELECT DISTINCT(token.id), token.creator, token.description, token.expiration, token.created, "group".name
		FROM "group"
		JOIN token ON "group".id = token.group_id
		JOIN group_user ON group_user.user_id = $1
		WHERE group_user.group_admin = true
		ORDER BY "group".name
	`
	rows, err := db.Query(query, userID)
	if err == sql.ErrNoRows {
		return tokens, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var creator, description sql.NullString
		tok := sdk.Token{}
		if err := rows.Scan(&tok.ID, &creator, &description, &tok.Expiration, &tok.Created, &tok.GroupName); err != nil {
			return nil, sdk.WrapError(err, "LoadTokens> Cannot scan the token line")
		}

		if creator.Valid {
			tok.Creator = creator.String
		}
		if description.Valid {
			tok.Description = description.String
		}

		tokens = append(tokens, tok)
	}
	return tokens, nil
}
