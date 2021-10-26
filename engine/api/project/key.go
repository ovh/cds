package project

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/fsamin/go-shredder"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbEncryptedData{}, "encrypted_data", false, "project_id", "content_name"))
	gorpmapping.Register(gorpmapping.New(sdk.Secret{}, "encrypted_data", false, "project_id", "content_name"))

}

type dbEncryptedData struct {
	ProjectID       int64  `db:"project_id"`
	Name            string `db:"content_name"`
	Token           string `db:"token"`
	EncyptedContent []byte `db:"encrypted_content"`
}

func ListEncryptedData(ctx context.Context, db gorp.SqlExecutor, projectID int64) ([]sdk.Secret, error) {
	var res []sdk.Secret
	query := gorpmapping.NewQuery("select content_name, token from encrypted_data where project_id = $1").Args(projectID)
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func DeleteEncryptedVariable(db gorp.SqlExecutor, projectID int64, name string) error {
	_, err := db.Exec("delete from encrypted_data where project_id = $1 and content_name = $2", projectID, name)
	return sdk.WithStack(err)
}

// EncryptWithBuiltinKey encrypt a content with the builtin gpg key encode, compress it and encode with base64
func EncryptWithBuiltinKey(ctx context.Context, db gorp.SqlExecutor, projectID int64, name, content string) (string, error) {
	existingToken, err := db.SelectStr("select token from encrypted_data where project_id = $1 and content_name = $2", projectID, name)
	if err != nil && err != sql.ErrNoRows {
		return "", sdk.WrapError(err, "Unable to request encrypted_data")
	}

	k, err := loadBuiltinKey(ctx, db, projectID)
	if err != nil {
		return "", sdk.WrapError(err, "Unable to load builtin key")
	}

	encryptedReader, err := shredder.GPGEncrypt([]byte(k.Public), strings.NewReader(content))
	if err != nil {
		return "", sdk.WrapError(err, "Unable to encrypt content")
	}

	encryptedContent, err := io.ReadAll(encryptedReader)
	if err != nil {
		return "", sdk.WrapError(err, "Unable to ungzip content")
	}

	compressedContent := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(compressedContent)
	if _, err := gzipWriter.Write(encryptedContent); err != nil {
		return "", sdk.WrapError(err, "Unable to write gzip content")
	}
	if err := gzipWriter.Close(); err != nil {
		return "", sdk.WrapError(err, "Unable to gzip content")
	}

	s := base64.StdEncoding.EncodeToString(compressedContent.Bytes())

	token := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, token)
	if n != len(token) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	token[8] = token[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	token[6] = token[6]&^0xf0 | 0x40

	bded := dbEncryptedData{
		ProjectID:       projectID,
		Name:            name,
		Token:           fmt.Sprintf("%x%x%x%x%x", token[0:4], token[4:6], token[6:8], token[8:10], token[10:]),
		EncyptedContent: []byte(s),
	}

	if existingToken != "" {
		bded.Token = existingToken
		if _, err := db.Update(&bded); err != nil {
			return "", sdk.WrapError(err, "Unable to update encrypted_data")
		}
	} else {
		if err := db.Insert(&bded); err != nil {
			return "", sdk.WrapError(err, "Unable to save encrypted_data")
		}
	}

	return bded.Token, nil
}

// DecryptWithBuiltinKey decrypt a base64-ed, gzipped, content
func DecryptWithBuiltinKey(ctx context.Context, db gorp.SqlExecutor, projectID int64, token string) (string, error) {
	dbed := dbEncryptedData{}
	if err := db.SelectOne(&dbed, "select * from encrypted_data where token = $1", token); err != nil {
		return "", sdk.WithStack(sdk.ErrProjectSecretDataUnknown)
	}

	k, err := loadBuiltinKey(ctx, db, projectID)
	if err != nil {
		return "", sdk.WrapError(sdk.ErrProjectSecretDataUnknown, "Unable to load builtin key")
	}

	b, err := base64.StdEncoding.DecodeString(string(dbed.EncyptedContent))
	if err != nil {
		return "", sdk.WrapError(err, "Unable to decode content")
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return "", sdk.WrapError(err, "Unable to ungzip content buffer")
	}

	uncompressedContent := new(bytes.Buffer)
	if _, err := io.Copy(uncompressedContent, gzipReader); err != nil {
		return "", sdk.WrapError(err, "Unable to write ungzip content")
	}

	if err := gzipReader.Close(); err != nil {
		return "", sdk.WrapError(err, "Unable to ungzip content")
	}

	decryptedReader, err := shredder.GPGDecrypt([]byte(k.Private), []byte{}, uncompressedContent)
	if err != nil {
		return "", sdk.WrapError(err, "Unable to decrypt content")
	}

	decryptedContent, err := io.ReadAll(decryptedReader)
	if err != nil {
		return "", err
	}

	return string(decryptedContent), nil
}
