package local

import (
	zxcvbn "github.com/nbutton23/zxcvbn-go"
	"golang.org/x/crypto/bcrypt"

	"github.com/ovh/cds/sdk"
)

func isPasswordValid(password string) error {
	passwordStrength := zxcvbn.PasswordStrength(password, nil).Score
	if passwordStrength < 3 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "given password is not strong enough")
	}
	return nil
}

func HashPassword(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot generate hash for given password")
	}
	return hash, nil
}

func CompareHashAndPassword(hash []byte, password string) error {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
