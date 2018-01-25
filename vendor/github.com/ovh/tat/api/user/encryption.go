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

	log "github.com/Sirupsen/logrus"
)

const (
	pwSaltBytes     = 32
	pwPasswordBytes = 64
	sep             = "$"
)

// GenerateSalt generates salt, 32 length, with rand.Reader
func GenerateSalt() (string, error) {
	salt := make([]byte, pwSaltBytes)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		log.Errorf("Error whith GenerateSalt:%s", err.Error())
		return "", err
	}
	return hex.EncodeToString(salt), nil
}

func generatePassword() (string, error) {
	password := make([]byte, pwPasswordBytes)
	_, err := io.ReadFull(rand.Reader, password)
	if err != nil {
		log.Errorf("Error whith generatePassword:%s", err.Error())
		return "", err
	}
	return hex.EncodeToString(password), nil
}

// HashPassword hashes password, with given salt
// It uses sha512.Sum
func HashPassword(password, salt string) string {
	h := password + salt
	s := sha512.Sum512([]byte(h)) // 11 micro s
	return hex.EncodeToString(s[:])
}

// generateUserPassword return clear password (to user only), salt and hashed password
// salt and hashed password are stored in db
func generateUserPassword() (string, string, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return "", "", err
	}
	password, err := generatePassword()
	if err != nil {
		return "", "", err
	}
	passwordHashed := HashPassword(password, salt)
	hash := fmt.Sprintf("%s%s%s%s%s", "sha512", sep, salt, sep, passwordHashed)
	return password, hash, nil
}

// isCheckValid return false if hashedVersion of clar is not equals to hash
// hash contains $sha512$salt$passwordHashed$
func isCheckValid(clear, hashField string) bool {
	_, salt, hashedInDB, err := splitHash(hashField)
	if err != nil {
		log.Errorf("Invalid Hash field in db : %s", err.Error())
		return false
	}
	hashed := HashPassword(clear, salt)
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
