package sdk

import (
	"math/rand"
	"time"
)

// IsInArray Check if the element is in the array
func IsInArray(elt int64, array []int64) bool {
	for _, item := range array {
		if item == elt {
			return true
		}
	}
	return false
}

// RandomString generate random string containing a-zA-Z0-9
func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
