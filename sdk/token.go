package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk/log"
)

// AuthDriver interface.
type AuthDriver interface {
	GetManifest() AuthDriverManifest
	GetSessionDuration() time.Duration
	GetSigninURI(state string) string
	CheckSigninRequest(AuthConsumerSigninRequest) error
	GetUserInfo(AuthConsumerSigninRequest) (AuthDriverUserInfo, error)
}

// AuthDriverManifest struct discribe a auth driver.
type AuthDriverManifest struct {
	Type           AuthConsumerType `json:"type"`
	SignupDisabled bool             `json:"signup_disabled,omitempty"`
}

// AuthConsumerScope alias type for string.
type AuthConsumerScope string

// IsValid returns validity for scope.
func (s AuthConsumerScope) IsValid() bool {
	for i := range AuthConsumerScopes {
		if AuthConsumerScopes[i] == s {
			return true
		}
	}
	return false
}

// Available auth consumer scopes.
const (
	AuthConsumerScopeUser         AuthConsumerScope = "User"
	AuthConsumerScopeAccessToken  AuthConsumerScope = "AccessToken"
	AuthConsumerScopeAction       AuthConsumerScope = "Action"
	AuthConsumerScopeAdmin        AuthConsumerScope = "Admin"
	AuthConsumerScopeGroup        AuthConsumerScope = "Group"
	AuthConsumerScopeTemplate     AuthConsumerScope = "Template"
	AuthConsumerScopeProject      AuthConsumerScope = "Project"
	AuthConsumerScopeRun          AuthConsumerScope = "Run"
	AuthConsumerScopeRunExecution AuthConsumerScope = "RunExecution"
	AuthConsumerScopeHooks        AuthConsumerScope = "Hooks"
	AuthConsumerScopeWorker       AuthConsumerScope = "Worker"
	AuthConsumerScopeWorkerModel  AuthConsumerScope = "WorkerModel"
	AuthConsumerScopeHatchery     AuthConsumerScope = "Hatchery"
	AuthConsumerScopeService      AuthConsumerScope = "Service"
)

// AuthConsumerScopes list.
var AuthConsumerScopes = []AuthConsumerScope{
	AuthConsumerScopeUser,
	AuthConsumerScopeAccessToken,
	AuthConsumerScopeAction,
	AuthConsumerScopeAdmin,
	AuthConsumerScopeGroup,
	AuthConsumerScopeTemplate,
	AuthConsumerScopeProject,
	AuthConsumerScopeRun,
	AuthConsumerScopeRunExecution,
	AuthConsumerScopeHooks,
	AuthConsumerScopeWorker,
	AuthConsumerScopeWorkerModel,
	AuthConsumerScopeHatchery,
	AuthConsumerScopeService,
}

// AuthConsumerScopeSlice type used for database json storage.
type AuthConsumerScopeSlice []AuthConsumerScope

// Scan scope slice.
func (s *AuthConsumerScopeSlice) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, s), "cannot unmarshal AuthConsumerScopeSlice")
}

// Value returns driver.Value from scope slice.
func (s AuthConsumerScopeSlice) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal AuthConsumerScopeSlice")
}

// AuthConsumerSigninRequest struct for auth consumer signin request.
type AuthConsumerSigninRequest map[string]string

// AuthConsumerSigninResponse response for a auth consumer signin.
type AuthConsumerSigninResponse struct {
	Token string            `json:"token"`
	User  *AuthentifiedUser `json:"user"`
}

// AuthConsumerCreateResponse response for a auth consumer creation.
type AuthConsumerCreateResponse struct {
	Token    string        `json:"token"`
	Consumer *AuthConsumer `json:"consumer"`
}

// AuthDriverUserInfo struct discribed a user returns by a auth driver.
type AuthDriverUserInfo struct {
	ExternalID string
	Username   string
	Fullname   string
	Email      string
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
	ConsumerGitlab       AuthConsumerType = "gitlab"
)

// IsValid returns validity of given auth consumer type.
func (t AuthConsumerType) IsValid() bool {
	switch t {
	case ConsumerBuiltin, ConsumerLocal, ConsumerLDAP, ConsumerCorporateSSO, ConsumerGithub, ConsumerGitlab:
		return true
	}
	return false
}

// IsValidExternal returns validity of given auth consumer type.
func (t AuthConsumerType) IsValidExternal() bool {
	switch t {
	case ConsumerLDAP, ConsumerCorporateSSO, ConsumerGithub, ConsumerGitlab:
		return true
	}
	return false
}

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
	ID                 string                 `json:"id" cli:"id,key" db:"id"`
	Name               string                 `json:"name" cli:"name" db:"name"`
	Description        string                 `json:"description" cli:"description" db:"description"`
	ParentID           *string                `json:"parent_id,omitempty" db:"parent_id"`
	AuthentifiedUserID string                 `json:"user_id,omitempty" db:"user_id"`
	Type               AuthConsumerType       `json:"type" cli:"type" db:"type"`
	Data               AuthConsumerData       `json:"-" db:"data"` // NEVER returns auth consumer data in json, TODO this fields should be visible only in auth package
	Created            time.Time              `json:"created" cli:"created" db:"created"`
	GroupIDs           Int64Slice             `json:"group_ids,omitempty" cli:"group_ids" db:"group_ids"`
	Scopes             AuthConsumerScopeSlice `json:"scopes,omitempty" cli:"scopes" db:"scopes"`
	// aggregates
	AuthentifiedUser *AuthentifiedUser `json:"user,omitempty" db:"-"`
	Groups           Groups            `json:"groups,omitempty" db:"-"`
}

// IsValid returns validity for auth consumer.
func (c AuthConsumer) IsValid() error {
	for _, s := range c.Scopes {
		if !s.IsValid() {
			return NewErrorFrom(ErrWrongRequest, "invalid given scope value %s", s)
		}
	}
	return nil
}

// GetGroupIDs returns group ids for auth consumer, if empty
// in consumer returns group ids from authentified user.
func (c AuthConsumer) GetGroupIDs() []int64 {
	var groupIDs []int64

	if len(c.GroupIDs) > 0 {
		groupIDs = c.GroupIDs
	} else if c.AuthentifiedUser != nil && c.Type != ConsumerBuiltin {
		groupIDs = c.AuthentifiedUser.GetGroupIDs()
	}

	return groupIDs
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
	ID         string                 `json:"id" cli:"id,key" db:"id"`
	ConsumerID string                 `json:"consumer_id" cli:"consumer_id" db:"consumer_id"`
	ExpireAt   time.Time              `json:"expire_at,omitempty" cli:"expire_at" db:"expire_at"`
	Created    time.Time              `json:"created" cli:"created" db:"created"`
	GroupIDs   Int64Slice             `json:"group_ids" cli:"group_ids" db:"group_ids"`
	Scopes     AuthConsumerScopeSlice `json:"scopes" cli:"scopes" db:"scopes"`
	// aggregates
	Consumer *AuthConsumer `json:"consumer,omitempty" db:"-"`
	Groups   []Group       `json:"groups,omitempty" db:"-"`
	Current  bool          `json:"current,omitempty" db:"-"`
}

// AuthSessionJWTClaims is the specific claims format for JWT session.
type AuthSessionJWTClaims struct {
	ID       string
	GroupIDs []int64
	Scopes   []AuthConsumerScope
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

// AuthConsumersToIDs returns ids of given auth consumers.
func AuthConsumersToIDs(cs []AuthConsumer) []string {
	ids := make([]string, len(cs))
	for i := range cs {
		ids[i] = cs[i].ID
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
