package user

import (
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ovh/cds/sdk/log"
)

const (
	pwSaltBytes     = 32
	pwPasswordBytes = 64
	sep             = "$"
)

// GeneratePassword Generate a password/token for the user
func GeneratePassword() (string, string, error) {
	salt, err := generateSalt()
	if err != nil {
		return "", "", err
	}
	password, err := generatePassword()
	if err != nil {
		return "", "", err
	}
	passwordHashed := hashPassword(password, salt)
	hash := fmt.Sprintf("%s%s%s%s%s", "sha512", sep, salt, sep, passwordHashed)
	return password, hash, nil
}

// IsCheckValid return false if hashedVersion of clar is not equals to hash. Hash contains $sha512$salt$passwordHashed$
func IsCheckValid(clear, hashField string) bool {
	_, salt, hashedInDB, err := splitHash(hashField)
	if err != nil {
		log.Warning("Invalid Hash field in db : %s\n", err)
		return false
	}
	hashed := hashPassword(clear, salt)
	if subtle.ConstantTimeCompare([]byte(hashedInDB), []byte(hashed)) != 1 {
		return false
	}
	return true
}

// splitHash split hash "sha512$salt$hash" to return method, salt, hashedPassword
func splitHash(hashField string) (string, string, string, error) {
	s := strings.Split(hashField, sep)
	if len(s) != 3 {
		return "", "", "", errors.New("Invalid hash")
	}
	return s[0], s[1], s[2], nil
}

func generateSalt() (string, error) {
	salt := make([]byte, pwSaltBytes)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		log.Warning("GenerateSalt: Error generating salt: %s\n", err)
		return "", err
	}
	return hex.EncodeToString(salt), nil
}

func generatePassword() (string, error) {
	password := make([]byte, pwPasswordBytes)
	_, err := io.ReadFull(rand.Reader, password)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(password), nil
}

func hashPassword(password, salt string) string {
	h := password + salt
	s := sha512.Sum512([]byte(h))
	return hex.EncodeToString(s[:])
}
