package sessionstore

import (
	"crypto/rand"
	"fmt"
	"io"
)

// NewSessionKey generates a random UUID according to RFC 4122
func NewSessionKey() (SessionKey, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return SessionKey(fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])), nil
}

//SessionKey is the session ID
type SessionKey string

//Store is the session store interface
type Store interface {
	New(SessionKey) (SessionKey, error)
	Exists(SessionKey) (bool, error)
	Get(SessionKey, string, interface{}) error
	Set(SessionKey, string, interface{}) error
	Delete(SessionKey) error
}

//Options is a struct to switch from in memory to redis session store
type Options struct {
	RedisHost, RedisPassword string
	TTL                      int
}
