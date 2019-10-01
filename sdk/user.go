package sdk

// User represent a CDS user.
type User struct {
	ID          int64           `json:"id" yaml:"-" cli:"-"`
	Username    string          `json:"username" yaml:"username" cli:"username,key"`
	Fullname    string          `json:"fullname" yaml:"fullname,omitempty" cli:"fullname"`
	Email       string          `json:"email" yaml:"email,omitempty" cli:"email"`
	Admin       bool            `json:"admin" yaml:"admin,omitempty" cli:"-"`
	Auth        Auth            `json:"-" yaml:"-" cli:"-"`
	Groups      []Group         `json:"groups,omitempty" yaml:"-" cli:"-"`
	Origin      string          `json:"origin" yaml:"origin,omitempty"`
	Favorites   []Favorite      `json:"favorites" yaml:"favorites"`
	Permissions UserPermissions `json:"permissions,omitempty" yaml:"-" cli:"-"`
	GroupAdmin  bool            `json:"-" yaml:"-" cli:"group_admin"`
}

// Favorite represent the favorites workflow or project of the user
type Favorite struct {
	ProjectIDs  []int64 `json:"project_ids" yaml:"project_ids"`
	WorkflowIDs []int64 `json:"workflow_ids" yaml:"workflow_ids"`
}

// UserPermissions is the set of permissions for a user
//easyjson:json
type UserPermissions struct {
	Groups        []string           `json:"Groups,omitempty"` // json key are capitalized to ensure exising data in cache are still valid
	GroupsAdmin   []string           `json:"GroupsAdmin,omitempty"`
	ProjectsPerm  UserPermissionsMap `json:"ProjectsPerm,omitempty"`
	WorkflowsPerm UserPermissionsMap `json:"WorkflowsPerm,omitempty"`
}

func (u UserPermissions) Clone() UserPermissions {
	up := UserPermissions{}
	up.Groups = make([]string, len(u.Groups))
	up.GroupsAdmin = make([]string, len(u.GroupsAdmin))
	up.ProjectsPerm = make(UserPermissionsMap, len(u.ProjectsPerm))
	up.WorkflowsPerm = make(UserPermissionsMap, len(u.WorkflowsPerm))
	copy(up.Groups, u.Groups)
	copy(up.GroupsAdmin, u.GroupsAdmin)

	for i, v := range u.ProjectsPerm {
		up.ProjectsPerm[i] = v
	}
	for i, v := range u.WorkflowsPerm {
		up.WorkflowsPerm[i] = v
	}
	return up
}

// UserPermissionsMap is a type of map. The in key the key and name of the object and value is the level of permissions
//easyjson:json
type UserPermissionsMap map[string]int

// UserPermissionKey returns a string representing a key for a user permission
func UserPermissionKey(k, n string) string {
	return k + "/" + n
}

// UserAPIRequest  request for rest API
type UserAPIRequest struct {
	User     User   `json:"user"`
	Callback string `json:"callback"`
}

// UserLoginRequest login request
type UserLoginRequest struct {
	RequestToken string `json:"request_token"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

type UserLoginCallbackRequest struct {
	RequestToken string `json:"request_token"`
	PublicKey    []byte `json:"public_key"`
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
