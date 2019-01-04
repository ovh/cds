package sdk

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

const (
	AccessTokenStatusEnabled  = "enabled"
	AccessTokenStatusDisabled = "disabled"
)

// AccessToken is either a Personnal Access Token or a Group Access Token
type AccessToken struct {
	ID          string     `json:"id" cli:"-" db:"id"`
	Description string     `json:"description" cli:"description,key" db:"description"`
	UserID      int64      `json:"user_id,omitempty" db:"user_id"`
	User        User       `json:"user" db:"-"`
	ExpireAt    *time.Time `json:"expired_at,omitempty" cli:"expired_at" db:"expired_at"`
	Created     time.Time  `json:"created" cli:"created" db:"created"`
	Status      string     `json:"status" cli:"status" db:"status"`
	Origin      string     `json:"-" cli:"-" db:"origin"`
	Groups      []Group    `json:"groups" cli:"-" db:"-"`
}

// Token describes tokens used by worker to access the API
// on behalf of a group.
type Token struct {
	ID          int64      `json:"id" cli:"id,key"`
	GroupID     int64      `json:"group_id"`
	GroupName   string     `json:"group_name" cli:"group_name"`
	Token       string     `json:"token" cli:"token"`
	Description string     `json:"description" cli:"description"`
	Creator     string     `json:"creator" cli:"creator"`
	Expiration  Expiration `json:"expiration" cli:"expiration"`
	Created     time.Time  `json:"created" cli:"created"`
}

// AccessTokenJWTClaims is the specific claims format for JWT Tokens
type AccessTokenJWTClaims struct {
	ID     string
	Groups []int64
	jwt.StandardClaims
}
