package local

import (
	zxcvbn "github.com/nbutton23/zxcvbn-go"
	"golang.org/x/crypto/bcrypt"

	"github.com/ovh/cds/sdk"
)

func isPasswordValid(password string) error {
	if len(password) > 256 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "given password is not strong enough, level should be >= 3")
	}
	passwordStrength := zxcvbn.PasswordStrength(password, nil).Score
	if passwordStrength < 3 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "given password is not strong enough, level should be >= 3")
	}
	return nil
}

// HashPassword return a hash from given password.
func HashPassword(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot generate hash for given password")
	}
	return hash, nil
}

// CompareHashAndPassword returns an error if given password don't match given hash.
func CompareHashAndPassword(hash []byte, password string) error {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
