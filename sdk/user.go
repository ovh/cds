package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"
	"time"
)

// User represent a CDS user.
type User struct {
	ID         int64      `json:"id" yaml:"-" cli:"-"`
	Username   string     `json:"username" yaml:"username" cli:"username,key"`
	Fullname   string     `json:"fullname" yaml:"fullname,omitempty" cli:"fullname"`
	Email      string     `json:"email" yaml:"email,omitempty" cli:"email"`
	Admin      bool       `json:"admin" yaml:"admin,omitempty" cli:"-"`
	Auth       Auth       `json:"-" yaml:"-" cli:"-"`
	Groups     []Group    `json:"groups,omitempty" yaml:"-" cli:"-"`
	Origin     string     `json:"origin" yaml:"origin,omitempty"`
	Favorites  []Favorite `json:"favorites" yaml:"favorites"`
	GroupAdmin bool       `json:"-" yaml:"-" cli:"group_admin"`
}

const (
	UserRingAdmin      = "ADMIN"
	UserRingMaintainer = "MAINTAINER"
	UserRingUser       = "USER"
)

func AuthentifiedUsersToIDs(users []*AuthentifiedUser) []string {
	ids := make([]string, len(users))
	for i := range users {
		ids[i] = (users)[i].ID
	}
	return ids
}

type Identifiable interface {
	GetUsername() string
	GetFullname() string
	Email() string
}

type GroupMember interface {
	GetGroups() []Group
}

type IdentifiableGroupMember interface {
	Identifiable
	GroupMember
}

type AuthentifiedUser struct {
	ID            string       `json:"id" yaml:"id" cli:"id,key" db:"id"`
	Username      string       `json:"username" yaml:"username" cli:"username,key" db:"username"`
	Fullname      string       `json:"fullname" yaml:"fullname,omitempty" cli:"fullname" db:"fullname"`
	Ring          string       `json:"ring" yaml:"ring,omitempty" db:"ring"`
	DateCreation  time.Time    `json:"date_creation" yaml:"date_creation" db:"date_creation"`
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

func (u AuthentifiedUser) GetGroups() []Group {
	if u.OldUserStruct == nil {
		return nil
	}
	return u.OldUserStruct.Groups
}

func (u AuthentifiedUser) GetUsername() string {
	return u.Username
}

func (u AuthentifiedUser) GetFullname() string {
	return u.Username
}

func (u AuthentifiedUser) Admin() bool {
	return u.Ring == UserRingAdmin
}

func (u AuthentifiedUser) Maintainer() bool {
	return u.Ring == UserRingMaintainer
}

func (u AuthentifiedUser) Email() string {
	if u.Contacts == nil {
		return ""
	}
	byEmails := u.Contacts.Filter(UserContactTypeEmail)
	if len(byEmails) == 0 {
		return ""
	}
	primaryEmailAdress := byEmails.Primary()
	if primaryEmailAdress != nil {
		return primaryEmailAdress.Value
	}
	return byEmails[0].Value
}

type UserLocalAuthentication struct {
	UserID        string `json:"user_id" db:"user_id"`
	ClearPassword string `json:"clear_password" db:"-"`
	Verified      bool   `json:"verified" db:"verified"`
}

type UserContact struct {
	ID             int    `json:"id" db:"id"`
	UserID         string `json:"user_id" db:"user_id"`
	Type           string `json:"type" db:"type"`
	Value          string `json:"value" db:"value"`
	PrimaryContact bool   `json:"primary_contact" db:"primary_contact"`
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
func (u UserContacts) Find(t, v string) *UserContact {
	for _, c := range u {
		if c.Type == t && c.Value == v {
			return &c
		}
	}
	return nil
}

func (u UserContacts) Primary() *UserContact {
	for _, c := range u {
		if c.PrimaryContact {
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

// UserRequest request new user creation
type UserRequest struct {
	Fullname             string `json:"fullname"`
	Email                string `json:"email"`
	Username             string `json:"username"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
	Callback             string `json:"callback"`
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

// UserLoginResponse  response from rest API
type UserLoginResponse struct {
	User  AuthentifiedUser `json:"user"`
	Token string           `json:"token"`
}

type UserLoginCallbackRequest struct {
	RequestToken string `json:"request_token"`
	PublicKey    []byte `json:"public_key"`
}

// UserEmailPattern  pattern for user email address
const UserEmailPattern = "(\\w[-._\\w]*\\w@\\w[-._\\w]*\\w\\.\\w{2,3})"
