package sdk

import "time"

// Auth Authentifaction Struct for user
type Auth struct {
	HashedPassword    string `json:"hashedPassword"`
	HashedTokenVerify string `json:"hashedTokenVerify"`
	EmailVerified     bool   `json:"emailVerified"`
	DateReset         int64  `json:"dateReset"`
}

// UserToken for user persistent session
type UserToken struct {
	Token              string    `json:"token" db:"token"`
	Timestamp          int64     `json:"timestamp" db:"-"`
	Comment            string    `json:"comment" db:"comment"`
	CreationDate       time.Time `json:"creation_date" db:"creation_date"`
	LastConnectionDate time.Time `json:"last_connection_date" db:"last_connection_date"`
	UserID             int64     `json:"-" db:"user_id"`
}

// NewAuth instanciate a new Authentification struct
func NewAuth(hashedToken string) *Auth {
	a := &Auth{
		HashedTokenVerify: hashedToken,
		EmailVerified:     false,
	}
	return a
}
