package tests

import (
	"math/rand"
	"testing"
	"time"
)

// RandomString have to be used only for tests
func RandomString(t *testing.T, strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
