package sdk

import (
	"crypto/rand"
	"encoding/hex"
)

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
