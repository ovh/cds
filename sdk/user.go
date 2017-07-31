package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// User represent a CDS user.
type User struct {
	ID       int64   `json:"id" yaml:"-" cli:"-"`
	Username string  `json:"username" yaml:"username" cli:"username,key"`
	Fullname string  `json:"fullname" yaml:"fullname,omitempty" cli:"fullname"`
	Email    string  `json:"email" yaml:"email,omitempty" cli:"-"`
	Admin    bool    `json:"admin" yaml:"admin,omitempty" cli:"-"`
	Auth     Auth    `json:"-" yaml:"-" cli:"-"`
	Groups   []Group `json:"groups,omitempty" yaml:"-" cli:"-"`
	Origin   string  `json:"origin" yaml:"origin,omitempty"`
}

// UserAPIRequest  request for rest API
type UserAPIRequest struct {
	User     User   `json:"user"`
	Callback string `json:"callback"`
}

// UserLoginRequest login request
type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserAPIResponse  response from rest API
type UserAPIResponse struct {
	User     User   `json:"user"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// UserEmailPattern  pattern for user email address
const UserEmailPattern = "(\\w[-._\\w]*\\w@\\w[-._\\w]*\\w\\.\\w{2,3})"

// NewUser instanciate a new User
func NewUser(username string) *User {
	u := &User{
		Username: username,
	}
	return u
}

//LoginUser call the /login handler
func LoginUser(username, password string) (bool, *UserAPIResponse, error) {
	request := UserLoginRequest{
		Username: username,
		Password: password,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return false, nil, err
	}

	data, code, err := Request("POST", "/login", data)
	if err != nil {
		return false, nil, err
	}

	if code != http.StatusOK {
		return false, nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	loginResponse := &UserAPIResponse{}
	if err := json.Unmarshal(data, loginResponse); err != nil {
		return false, nil, fmt.Errorf("Error unmarshalling response: %s", err)
	}

	return true, loginResponse, nil
}

// DeleteUser Call API to delete the given user
func DeleteUser(name string) error {

	url := fmt.Sprintf("/user/%s", name)
	data, code, err := Request("DELETE", url, nil)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
}

// AddUser creates a new user available only to creator by default
func AddUser(name, fname, email, callback string) error {
	u := NewUser(name)
	u.Fullname = fname
	u.Email = email

	request := UserAPIRequest{
		User:     *u,
		Callback: callback,
	}

	data, err := json.MarshalIndent(request, " ", " ")
	if err != nil {
		return err
	}

	data, code, err := Request("POST", "/user/signup", data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}

	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil

}

func updateUser(username string, user *User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/user/%s", username)
	data, code, err := Request("PUT", url, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
}

// UpdateUserEmail Change user email address
func UpdateUserEmail(name, email string) error {
	u, err := GetUser(name)
	if err != nil {
		return err
	}

	u.Email = email

	return updateUser(u.Username, u)
}

// RenameUser Rename given user
func RenameUser(name, fname string) error {
	u, err := GetUser(name)
	if err != nil {
		return err
	}

	u.Fullname = fname

	return updateUser(u.Username, u)
}

// UpdateUsername Change username
func UpdateUsername(oldUsername, newUsername string) error {
	u, err := GetUser(oldUsername)
	if err != nil {
		return err
	}

	u.Username = newUsername

	return updateUser(oldUsername, u)
}

// VerifyUser verify the token received by mail
func VerifyUser(name, token string) (UserAPIResponse, error) {
	confirmResponse := UserAPIResponse{}

	path := fmt.Sprintf("/user/%s/confirm/%s", name, token)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return confirmResponse, err
	}

	if code != http.StatusOK {
		return confirmResponse, fmt.Errorf("Error [%d]: %s", code, data)
	}

	err = json.Unmarshal(data, &confirmResponse)
	if err != nil {
		return confirmResponse, fmt.Errorf("Error unmarshalling response: %s", err)
	}

	return confirmResponse, nil
}

// ResetUser reset user password
func ResetUser(name, email, callback string) error {
	u := NewUser(name)
	u.Email = email

	request := UserAPIRequest{
		User:     *u,
		Callback: callback,
	}

	data, err := json.MarshalIndent(request, " ", " ")
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/user/%s/reset", name)
	data, code, err := Request("POST", path, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
}

// GetUser return the given user
func GetUser(username string) (*User, error) {
	if username == "" {
		return nil, ErrInvalidUsername
	}

	user := &User{}

	data, code, errR := Request("GET", fmt.Sprintf("/user/%s", username), nil)
	if errR != nil {
		return nil, errR
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	if err := json.Unmarshal(data, user); err != nil {
		return nil, err
	}
	return user, nil
}

// ListUsers returns all available user to caller
func ListUsers() ([]User, error) {

	data, code, err := Request("GET", "/user", nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	var users []User
	err = json.Unmarshal(data, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// Me returns user instance for connected user
func Me() (*User, error) {
	return GetUser(user)
}

// IsAdmin checks if user is admin
func IsAdmin() (bool, error) {
	me, err := Me()
	if err != nil {
		return false, err
	}
	return me.Admin, nil
}
