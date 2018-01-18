package application

import (
	"encoding/base64"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// EncryptVCSStrategyPassword Encrypt vcs password
func EncryptVCSStrategyPassword(app *sdk.Application) error {
	encryptedPwd, err := secret.Encrypt([]byte(app.RepositoryStrategy.Password))
	if err != nil {
		return sdk.WrapError(err, "EncryptVCSStrategyPassword> Unable to encrypt password")
	}

	app.RepositoryStrategy.Password = base64.StdEncoding.EncodeToString(encryptedPwd)
	return nil
}

// DecryptVCSStrategyPassword Decrypt vs password
func DecryptVCSStrategyPassword(app *sdk.Application) error {
	if app.RepositoryStrategy.Password == "" {
		return nil
	}
	encryptedPassword, err64 := base64.StdEncoding.DecodeString(app.RepositoryStrategy.Password)
	if err64 != nil {
		return sdk.WrapError(err64, "EncryptVCSStrategyPassword> Unable to decoding password")
	}

	clearPWD, err := secret.Decrypt([]byte(encryptedPassword))
	if err != nil {
		return sdk.WrapError(err, "EncryptVCSStrategyPassword> Unable to decrypt password")
	}

	app.RepositoryStrategy.Password = string(clearPWD)
	return nil
}
