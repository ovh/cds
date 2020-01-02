package sdk

import (
	"fmt"
	"io"
	"crypto/rand"

	"github.com/pborman/uuid"
)

// UUID returns a UUID v4
func UUID() string {
	uuID := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuID)
	if n != len(uuID) || err != nil {
		panic(err)
	}
	// variant bits; see section 4.1.1
	uuID[8] = uuID[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuID[6] = uuID[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuID[0:4], uuID[4:6], uuID[6:8], uuID[8:10], uuID[10:])
}

func IsValidUUID(uu string) bool {
	if result := uuid.Parse(uu); result == nil {
		return false
	}
	return true
}
