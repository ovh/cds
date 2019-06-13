package sdk

import (
	"context"
	"database/sql/driver"
	json "encoding/json"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"
	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk/log"
)

type AuthDriver interface {
	GetManifest() AuthDriverManifest
	CheckRequest(AuthDriverRequest) error
	CheckAuthentication(context.Context, gorp.SqlExecutor, *http.Request) (*AuthentifiedUser, error)
}

type AuthDriverManifest struct {
	Type   AuthConsumerType          `json:"type"`
	URL    string                    `json:"url"`
	Method string                    `json:"method"`
	Fields []AuthDriverManifestField `json:"fields"`
}

type AuthDriverManifestFieldType string

const (
	FieldString   AuthDriverManifestFieldType = "string"
	FieldEmail    AuthDriverManifestFieldType = "email"
	FieldPassword AuthDriverManifestFieldType = "password"
)

type AuthDriverManifestField struct {
	Name string                      `json:"name"`
	Type AuthDriverManifestFieldType `json:"type"`
}

type AuthDriverRequest map[string]string

// IsValid checks that current driver request is valid according given driver manifest.
func (r AuthDriverRequest) IsValid(m AuthDriverManifest) error {
	for _, f := range m.Fields {
		v, okValue := r[f.Name]
		if !okValue || v == "" {
			return NewErrorFrom(ErrWrongRequest, "missing driver field '%s' of type '%s'", f.Name, f.Type)
		}

		// check value of type email
		if f.Type == FieldEmail && !IsValidEmail(v) {
			return NewErrorFrom(ErrWrongRequest, "given value for field '%s' is not an email", f.Name)
		}
	}

	return nil
}

// AuthConsumerRequest struct used by clients to create a new builtin auth consumer.
type AuthConsumerRequest struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	GroupsIDs             []int64  `json:"groups"`
	Scopes                []string `json:"scopes"`
	ExpirationDelaySecond float64  `json:"expiration_delay_second"`
}

// AuthConsumerType constant to identify what is the driver used to create a consumer.
type AuthConsumerType string

// Consumer types.
const (
	ConsumerBuiltin      AuthConsumerType = "builtin"
	ConsumerLocal        AuthConsumerType = "local"
	ConsumerLDAP         AuthConsumerType = "ldap"
	ConsumerCorporateSSO AuthConsumerType = "corporate-sso"
	ConsumerGithub       AuthConsumerType = "github"
)

// AuthConsumerData contains specific information from the auth driver.
type AuthConsumerData map[string]string

// Scan consumer data.
func (d *AuthConsumerData) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, d), "cannot unmarshal AuthConsumerData")
}

// Value returns driver.Value from consumer data.
func (d AuthConsumerData) Value() (driver.Value, error) {
	j, err := json.Marshal(d)
	return j, WrapError(err, "cannot marshal AuthConsumerData")
}

// AuthConsumer issues session linked to an authentified user.
type AuthConsumer struct {
	ID                 string           `json:"id" cli:"id,key" db:"id"`
	Name               string           `json:"name" cli:"name" db:"name"`
	Description        string           `json:"description" cli:"description" db:"description"`
	ParentID           *string          `json:"parent_id" db:"parent_id"`
	AuthentifiedUserID string           `json:"user_id,omitempty" db:"user_id"`
	Type               AuthConsumerType `json:"type" cli:"type" db:"type"`
	Data               AuthConsumerData `json:"data" db:"data"`
	Created            time.Time        `json:"created" cli:"created" db:"created"`
	GroupIDs           Int64Slice       `json:"group_ids" cli:"group_ids" db:"group_ids"`
	Scopes             StringSlice      `json:"scopes" cli:"scopes" db:"scopes"`
	// aggregates
	AuthentifiedUser *AuthentifiedUser `json:"user" db:"-"`
}

func (c AuthConsumer) GetGroupIDs() []int64 {
	return c.GroupIDs
}

func (c AuthConsumer) Admin() bool {
	admin := c.AuthentifiedUser.Admin()
	log.Debug("AuthConsumer.Admin> consumer on behalf of user %s is admin: %t", c.AuthentifiedUser.GetFullname(), admin)
	return admin
}

func (c AuthConsumer) Maintainer() bool {
	return c.AuthentifiedUser.Maintainer()
}

func (c AuthConsumer) GetConsumerName() string {
	return c.Name
}

func (c AuthConsumer) GetUsername() string {
	return c.AuthentifiedUser.Username
}

func (c AuthConsumer) GetFullname() string {
	return c.AuthentifiedUser.Fullname
}

func (c AuthConsumer) GetEmail() string {
	return c.AuthentifiedUser.GetEmail()
}

func (c AuthConsumer) GetDEPRECATEDUserStruct() *User {
	return c.AuthentifiedUser.OldUserStruct
}

// AuthSession struct.
type AuthSession struct {
	ID         string      `json:"id" cli:"id,key" db:"id"`
	ConsumerID string      `json:"consumer_id" cli:"consumer_id" db:"consumer_id"`
	ExpireAt   time.Time   `json:"expired_at,omitempty" cli:"expired_at" db:"expired_at"`
	Created    time.Time   `json:"created" cli:"created" db:"created"`
	GroupIDs   Int64Slice  `json:"group_ids" cli:"group_ids" db:"group_ids"`
	Scopes     StringSlice `json:"scopes" cli:"scopes" db:"scopes"`
	// aggregates
	Consumer *AuthConsumer `json:"consumer" db:"-"`
	Groups   []Group       `json:"groups" db:"-"`
}

// AuthSessionJWTClaims is the specific claims format for JWT session.
type AuthSessionJWTClaims struct {
	ID       string
	GroupIDs []int64
	Scopes   []string
	jwt.StandardClaims
}

// AuthSessionsToIDs returns ids of given auth sessions.
func AuthSessionsToIDs(ass []*AuthSession) []string {
	ids := make([]string, len(ass))
	for i := range ass {
		ids[i] = ass[i].ID
	}
	return ids
}

// AuthConsumersToAuthentifiedUserIDs returns ids of given auth consumers.
func AuthConsumersToAuthentifiedUserIDs(cs []*AuthConsumer) []string {
	ids := make([]string, len(cs))
	for i := range cs {
		ids[i] = cs[i].AuthentifiedUserID
	}
	return ids
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

// Available access tokens scopes
const (
	AccessTokenScopeALL          = "all"
	AccessTokenScopeUser         = "User"
	AccessTokenScopeAccessToken  = "AccessToken"
	AccessTokenScopeAction       = "Action"
	AccessTokenScopeAdmin        = "Admin"
	AccessTokenScopeGroup        = "Group"
	AccessTokenScopeTemplate     = "Template"
	AccessTokenScopeProject      = "Project"
	AccessTokenScopeRun          = "Run"
	AccessTokenScopeRunExecution = "RunExecution"
	AccessTokenScopeHooks        = "Hooks"
	AccessTokenScopeWorker       = "Worker"
	AccessTokenScopeWorkerModel  = "WorkerModel"
	AccessTokenScopeHatchery     = "Hatchery"
)

type Permission struct {
	IDName
	Role    int   `db:"role"`
	GroupID int64 `db:"groupID"`
}

type Permissions []Permission

type GroupPermissions struct {
	Projects     Permissions
	Workflows    Permissions
	Acions       Permissions
	WorkerModels Permissions
}

func (perms GroupPermissions) ProjectPermission(key string) (int, bool) {
	for _, projPerm := range perms.Projects {
		if projPerm.Name == key {
			return projPerm.Role, true
		}
	}
	return -1, false
}

// AuthConsumerLocalSignupResponse response for a auth local signup.
type AuthConsumerLocalSignupResponse struct {
	VerifyToken string `json:"verify_token"`
}

// AuthConsumerLocalSigninRequest request struct to signin on local auth.
type AuthConsumerLocalSigninRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// IsValid returns validity for signin request.
func (r AuthConsumerLocalSigninRequest) IsValid() error {
	if r.Username == "" {
		return NewErrorFrom(ErrInvalidUsername, "empty username is invalid")
	}
	if r.Password == "" {
		return NewErrorFrom(ErrInvalidUsername, "empty password is invalid")
	}

	return nil
}

// AuthConsumerLocalSigninResponse response for a auth local signin.
type AuthConsumerLocalSigninResponse struct {
	Token string            `json:"token"`
	User  *AuthentifiedUser `json:"user"`
}
