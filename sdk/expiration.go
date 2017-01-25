package sdk

import "fmt"

// Expiration defines how worker key should expire
type Expiration int

// Worker key expiry options
const (
	_ Expiration = iota
	Session
	Daily
	Persistent
)

func (e Expiration) String() string {
	switch e {
	case Session:
		return "session"
	case Daily:
		return "daily"
	case Persistent:
		return "persistent"
	default:
		return "sessions"
	}
}

// ExpirationFromString returns a typed Expiration from a string
func ExpirationFromString(s string) (Expiration, error) {
	switch s {
	case "session":
		return Session, nil
	case "daily":
		return Daily, nil
	case "persistent":
		return Persistent, nil
	}

	return Expiration(0), fmt.Errorf("invalid expiration format (%s)",
		[]Expiration{Session, Daily, Persistent})
}
