package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"
	"regexp"
	"time"
)

// User rings.
const (
	UserRingAdmin      = "ADMIN"
	UserRingMaintainer = "MAINTAINER"
	UserRingUser       = "USER"
)

type Identifiable interface {
	GetUsername() string
	GetEmail() string
	GetFullname() string
}

var _ Identifiable = new(AuthConsumer)
var _ Identifiable = new(AuthentifiedUser)

type UserRegistration struct {
	ID       string    `json:"id" db:"id"`
	Created  time.Time `json:"created" db:"created"`
	Username string    `json:"username"  db:"username"`
	Fullname string    `json:"fullname"  db:"fullname"`
	Email    string    `json:"email"  db:"email"`
	Hash     string    `json:"-"  db:"hash"` // do no return hash in json
}

var UsernameRegex = regexp.MustCompile("[a-z0-9._-]{3,32}")

// AuthentifiedUser struct contains all information about a cds user.
type AuthentifiedUser struct {
	ID       string    `json:"id" yaml:"id" cli:"id" db:"id"`
	Created  time.Time `json:"created" yaml:"created" cli:"created" db:"created"`
	Username string    `json:"username" yaml:"username" cli:"username,key" db:"username"`
	Fullname string    `json:"fullname" yaml:"fullname,omitempty" cli:"fullname" db:"fullname"`
	Ring     string    `json:"ring" yaml:"ring,omitempty" cli:"ring" db:"ring"`
	// aggregates
	Contacts  UserContacts `json:"-" yaml:"-" db:"-"`
	Favorites []Favorite   `json:"favorites" yaml:"favorites" db:"-"`
	Groups    Groups       `json:"groups" yaml:"groups" db:"-"`
}

// IsValid returns an error if given user's infos are not valid.
func (u AuthentifiedUser) IsValid() error {
	if u.Username == "" || u.Username == "me" {
		return NewErrorFrom(ErrWrongRequest, "invalid given username")
	}
	if u.Fullname == "" {
		return NewErrorFrom(ErrWrongRequest, "invalid given fullname")
	}

	switch u.Ring {
	case UserRingAdmin, UserRingMaintainer, UserRingUser:
		return nil
	}
	return NewErrorFrom(ErrWrongRequest, "invalid given ring value")
}

// Value returns driver.Value from workflow template request.
func (u AuthentifiedUser) Value() (driver.Value, error) {
	j, err := json.Marshal(u)
	return j, WrapError(err, "cannot marshal AuthentifiedUser")
}

// Scan workflow template request.
func (u *AuthentifiedUser) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, u), "cannot unmarshal AuthentifiedUser")
}

// GetGroupIDs returns groups ids for user
func (u AuthentifiedUser) GetGroupIDs() []int64 {
	if u.Groups == nil {
		return nil
	}
	return u.Groups.ToIDs()
}

func (u AuthentifiedUser) GetUsername() string {
	return u.Username
}

// GetEmail return the primary email for the authentified user (should exists).
func (u AuthentifiedUser) GetEmail() string {
	if u.Contacts == nil {
		return ""
	}
	byEmails := u.Contacts.Filter(UserContactTypeEmail)
	primaryEmailAdress := byEmails.Primary()
	return primaryEmailAdress.Value
}

func (u AuthentifiedUser) GetFullname() string {
	return u.Fullname
}

// AuthentifiedUsers provides func for authentified user list.
type AuthentifiedUsers []AuthentifiedUser

func (a AuthentifiedUsers) IDs() []string {
	ids := make([]string, len(a))
	for i := range a {
		ids[i] = (a)[i].ID
	}
	return ids
}

// ToMapByID returns a map of authentified users indexed by ids.
func (a AuthentifiedUsers) ToMapByID() map[string]AuthentifiedUser {
	m := make(map[string]AuthentifiedUser, len(a))
	for i := range a {
		m[a[i].ID] = a[i]
	}
	return m
}

// AuthentifiedUsersToIDs returns ids for given authentified user list.
func AuthentifiedUsersToIDs(users []*AuthentifiedUser) []string {
	ids := make([]string, len(users))
	for i := range users {
		ids[i] = (users)[i].ID
	}
	return ids
}

// UserContact struct
type UserContact struct {
	ID       int64     `json:"id" cli:"id,key" db:"id"`
	Created  time.Time `json:"created" cli:"created" db:"created"`
	UserID   string    `json:"user_id" db:"user_id"`
	Type     string    `json:"type" cli:"type" db:"type"`
	Value    string    `json:"value" cli:"value" db:"value"`
	Primary  bool      `json:"primary" cli:"primary" db:"primary_contact"`
	Verified bool      `json:"verified" cli:"verified" db:"verified"`
}

const UserContactTypeEmail = "email"

type UserContacts []UserContact

func (u UserContacts) Filter(t string) UserContacts {
	var filtered = UserContacts{}
	for _, c := range u {
		if c.Type == t {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (u UserContacts) Find(contactType, contactValue string) *UserContact {
	for _, c := range u {
		if c.Type == contactType && c.Value == contactValue {
			return &c
		}
	}
	return nil
}

func (u UserContacts) Primary() *UserContact {
	for _, c := range u {
		if c.Primary {
			return &c
		}
	}
	return nil
}

// Favorite represent the favorites workflow or project of the user
type Favorite struct {
	ProjectIDs  []int64 `json:"project_ids" yaml:"project_ids"`
	WorkflowIDs []int64 `json:"workflow_ids" yaml:"workflow_ids"`
}

type UserResponse struct {
	AuthentifiedUser
	VerifyToken string `json:"verify_token"`
}

type UserResetRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	VerifyToken string `json:"verify_token"`
	Callback    string `json:"callback"`
}

// UserLoginRequest login request
type UserLoginRequest struct {
	RequestToken string `json:"request_token"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

var emailPattern = regexp.MustCompile(`\w[+-._\w]*\w@\w[-._\w]*\w\.\w*`)

// IsValidEmail  Check if user email address is ok
func IsValidEmail(email string) bool {
	return emailPattern.MatchString(email)
}
