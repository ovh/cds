package sdk

import (
	"encoding/json"
	"fmt"
)

// Auth Authentifaction Struct for user
type Auth struct {
	HashedPassword    string      `json:"hashedPassword"`
	HashedTokenVerify string      `json:"hashedTokenVerify"`
	EmailVerified     bool        `json:"emailVerified"`
	DateReset         int64       `json:"dateReset"`
	Tokens            []UserToken `json:"tokens,omitempty"`
}

// UserToken for user persistent session
type UserToken struct {
	Token     string `json:"token"`
	Timestamp int64  `json:"timestamp"`
	Comment   string `json:"comment"`
}

// NewAuth instanciate a new Authentification struct
func NewAuth(hashedToken string) *Auth {
	a := &Auth{
		HashedTokenVerify: hashedToken,
		EmailVerified:     false,
	}
	return a
}

// JSON return the marshalled string of Auth object
func (a *Auth) JSON() string {

	data, err := json.Marshal(a)
	if err != nil {
		fmt.Printf("Auth.JSON: cannot marshal: %s\n", err)
		return ""
	}

	return string(data)
}

// FromJSON unmarshal given json data into Auth object
func (a *Auth) FromJSON(data []byte) (*Auth, error) {
	return a, json.Unmarshal(data, &a)
}
