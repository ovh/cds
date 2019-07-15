package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

// User represent a CDS user.
type User struct {
	ID        int64      `json:"id" yaml:"-" cli:"-"`
	Username  string     `json:"username" yaml:"username" cli:"username,key"`
	Fullname  string     `json:"fullname" yaml:"fullname,omitempty" cli:"fullname"`
	Email     string     `json:"email" yaml:"email,omitempty" cli:"email"`
	Groups    Groups     `json:"groups,omitempty" yaml:"-" cli:"-"`
	Origin    string     `json:"origin" yaml:"origin,omitempty"`
	Favorites []Favorite `json:"favorites" yaml:"favorites"`
	// aggregated
	Admin      bool `json:"admin,omitempty" yaml:"admin,omitempty" cli:"admin"`
	GroupAdmin bool `json:"group_admin,omitempty" yaml:"group_admin,omitempty" cli:"group_admin"`
}

// Value returns driver.Value from user.
func (u User) Value() (driver.Value, error) {
	j, err := json.Marshal(u)
	return j, WrapError(err, "cannot marshal User")
}

// Scan user.
func (u *User) Scan(src interface{}) error {
	source, ok := src.(string)
	if !ok {
		return WithStack(errors.New("type assertion .(string) failed"))
	}
	return WrapError(json.Unmarshal([]byte(source), u), "cannot unmarshal User")
}

// User rings.
const (
	UserRingAdmin      = "ADMIN"
	UserRingMaintainer = "MAINTAINER"
	UserRingUser       = "USER"
)

// AuthentifiedUsersToIDs returns ids for given authentified user list.
func AuthentifiedUsersToIDs(users []*AuthentifiedUser) []string {
	ids := make([]string, len(users))
	for i := range users {
		ids[i] = (users)[i].ID
	}
	return ids
}

type Identifiable interface {
	GetConsumerName() string
	GetUsername() string
	GetFullname() string
	GetEmail() string
}

type AuthentifiedUser struct {
	ID       string    `json:"id" yaml:"id" cli:"id,key" db:"id"`
	Created  time.Time `json:"created" yaml:"created" cli:"created" db:"created"`
	Username string    `json:"username" yaml:"username" cli:"username" db:"username"`
	Fullname string    `json:"fullname" yaml:"fullname,omitempty" cli:"fullname" db:"fullname"`
	Ring     string    `json:"ring" yaml:"ring,omitempty" cli:"ring" db:"ring"`
	// aggregates
	Contacts      UserContacts `json:"contacts" yaml:"contacts" db:"-"`
	OldUserStruct *User        `json:"old_user_struct" yaml:"old_user_struct" db:"-"`
}

// Value returns driver.Value from workflow template request.
func (w AuthentifiedUser) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal AuthentifiedUser")
}

// Scan workflow template request.
func (w *AuthentifiedUser) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, w), "cannot unmarshal AuthentifiedUser")
}

func (u AuthentifiedUser) GetGroupIDs() []int64 {
	if u.OldUserStruct == nil {
		return nil
	}
	return u.OldUserStruct.Groups.ToIDs()
}

func (u AuthentifiedUser) GetUsername() string {
	return u.Username
}

func (u AuthentifiedUser) GetFullname() string {
	return u.Fullname
}

func (u AuthentifiedUser) GetConsumerName() string {
	return u.Fullname
}

func (u AuthentifiedUser) Admin() bool {
	return u.Ring == UserRingAdmin
}

func (u AuthentifiedUser) Maintainer() bool {
	return u.Ring == UserRingMaintainer
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

// IsValidEmail  Check if user email address is ok
func IsValidEmail(email string) bool {
	pattern := "(\\w[-._\\w]*\\w@\\w[-._\\w]*\\w\\.\\w{2,3})"
	regexp := regexp.MustCompile(pattern)
	return regexp.MatchString(email)
}
