package sdk

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
)

func Hash512(in string) string {
	hasher := sha512.New()
	hasher.Write([]byte(in))
	return hex.EncodeToString(hasher.Sum(nil))
}

func GenerateHash() (string, error) {
	size := 128
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		return "", WrapError(err, "rand.Read failed")
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]
	return string(token), nil
}
