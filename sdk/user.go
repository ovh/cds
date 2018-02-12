package sdk

import (
	"encoding/json"
	"fmt"
	"strings"
)

// User represent a CDS user.
type User struct {
	ID          int64           `json:"id" yaml:"-" cli:"-"`
	Username    string          `json:"username" yaml:"username" cli:"username,key"`
	Fullname    string          `json:"fullname" yaml:"fullname,omitempty" cli:"fullname"`
	Email       string          `json:"email" yaml:"email,omitempty" cli:"-"`
	Admin       bool            `json:"admin" yaml:"admin,omitempty" cli:"-"`
	Auth        Auth            `json:"-" yaml:"-" cli:"-"`
	Groups      []Group         `json:"groups,omitempty" yaml:"-" cli:"-"`
	Origin      string          `json:"origin" yaml:"origin,omitempty"`
	Permissions UserPermissions `json:"permissions,omitempty" yaml:"-" cli:"-"`
}

// UserPermissions is the set of permissions for a user
//easyjson:json
type UserPermissions struct {
	Groups           []string           `json:"Groups,omitempty"` // json key are capitalized to ensure exising data in cache are still valid
	GroupsAdmin      []string           `json:"GroupsAdmin,omitempty"`
	ProjectsPerm     map[string]int     `json:"ProjectsPerm,omitempty"`
	ApplicationsPerm UserPermissionsMap `json:"ApplicationsPerm,omitempty"`
	WorkflowsPerm    UserPermissionsMap `json:"WorkflowsPerm,omitempty"`
	PipelinesPerm    UserPermissionsMap `json:"PipelinesPerm,omitempty"`
	EnvironmentsPerm UserPermissionsMap `json:"EnvironmentsPerm,omitempty"`
}

// UserPermissionsMap is a type of map. The in key the key and name of the object and value is the level of permissions
type UserPermissionsMap map[UserPermissionKey]int

// UserPermissionKey is used as a key in UserPermissionsMap
type UserPermissionKey struct {
	Key  string
	Name string
}

//MarshalJSON is the json.Marshaller implementation usefull to serialize UserPermissionsMap
func (m UserPermissionsMap) MarshalJSON() ([]byte, error) {
	var data = make(map[string]int, len(m))
	for k, v := range m {
		data[k.Key+"/"+k.Name] = v
	}
	return json.Marshal(data)
}

//UnmarshalJSON is the json.Unmarshaller implementation usefull to deserialize UserPermissionsMap
func (m *UserPermissionsMap) UnmarshalJSON(b []byte) error {
	data := map[string]int{}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	*m = make(map[UserPermissionKey]int)

	for k, v := range data {
		t := strings.SplitN(k, "/", 2)
		if len(t) != 2 {
			return fmt.Errorf("json: unable to unmarshal permissions")
		}
		(*m)[UserPermissionKey{Key: t[0], Name: t[1]}] = v
	}
	return nil
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

	data, _, err = Request("POST", "/login", data)
	if err != nil {
		return false, nil, err
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
	data, _, err := Request("DELETE", url, nil)
	if err != nil {
		return err
	}

	return DecodeError(data)
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

	data, _, err = Request("POST", "/user/signup", data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

func updateUser(username string, user *User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/user/%s", username)
	data, _, err = Request("PUT", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
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
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return confirmResponse, err
	}

	if err := json.Unmarshal(data, &confirmResponse); err != nil {
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
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// GetUser return the given user
func GetUser(username string) (*User, error) {
	if username == "" {
		return nil, ErrInvalidUsername
	}

	user := &User{}
	data, _, errR := Request("GET", fmt.Sprintf("/user/%s", username), nil)
	if errR != nil {
		return nil, errR
	}

	if err := json.Unmarshal(data, user); err != nil {
		return nil, err
	}
	return user, nil
}

// ListUsers returns all available user to caller
func ListUsers() ([]User, error) {
	data, _, err := Request("GET", "/user", nil)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
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
