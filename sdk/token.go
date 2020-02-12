package sdk

import (
	"context"
	"database/sql/driver"
	json "encoding/json"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// AuthDriver interface.
type AuthDriver interface {
	GetManifest() AuthDriverManifest
	GetSessionDuration() time.Duration
	CheckSigninRequest(AuthConsumerSigninRequest) error
	GetUserInfo(context.Context, AuthConsumerSigninRequest) (AuthDriverUserInfo, error)
}

type AuthDriverWithRedirect interface {
	AuthDriver
	GetSigninURI(AuthSigninConsumerToken) (AuthDriverSigningRedirect, error)
}

type AuthDriverWithSigninStateToken interface {
	AuthDriver
	CheckSigninStateToken(AuthConsumerSigninRequest) error
}

type AuthDriverSigningRedirect struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Body        map[string]string `json:"body"`
	ContentType string            `json:"content_type"`
}

// AuthDriverResponse struct contains drivers manifest and some info about auth config.
type AuthDriverResponse struct {
	IsFirstConnection bool                `json:"is_first_connection"`
	Drivers           AuthDriverManifests `json:"manifests"`
}

// AuthDriverManifests gives functions on driver manifest slice.
type AuthDriverManifests []AuthDriverManifest

// FindByConsumerType returns a manifest for given consumer type if exists.
func (a AuthDriverManifests) FindByConsumerType(consumerType AuthConsumerType) (AuthDriverManifest, bool) {
	for _, m := range a {
		if m.Type == consumerType {
			return m, true
		}
	}
	return AuthDriverManifest{}, false
}

// ExistsConsumerType returns if a driver exists for given consumer type.
func (a AuthDriverManifests) ExistsConsumerType(consumerType AuthConsumerType) bool {
	_, found := a.FindByConsumerType(consumerType)
	return found
}

// AuthDriverManifest struct describe a auth driver.
type AuthDriverManifest struct {
	Type           AuthConsumerType `json:"type"`
	SignupDisabled bool             `json:"signup_disabled"`
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

// AuthConsumerRegenRequest struct.
type AuthConsumerRegenRequest struct {
	RevokeSessions bool `json:"revoke_sessions"`
}

// AuthConsumerSigninRequest struct for auth consumer signin request.
type AuthConsumerSigninRequest map[string]string

// AuthConsumerSigninResponse response for a auth consumer signin.
type AuthConsumerSigninResponse struct {
	APIURL string            `json:"api_url,omitempty"`
	Token  string            `json:"token"` // session token
	User   *AuthentifiedUser `json:"user"`
}

// AuthConsumerCreateResponse response for a auth consumer creation.
type AuthConsumerCreateResponse struct {
	Token    string        `json:"token"` // sign in token
	Consumer *AuthConsumer `json:"consumer"`
}

// AuthDriverUserInfo struct discribed a user returns by a auth driver.
type AuthDriverUserInfo struct {
	ExternalID string
	Username   string
	Fullname   string
	Email      string
	MFA        bool
}

// AuthCurrentConsumerResponse describe the current consumer and the current session
type AuthCurrentConsumerResponse struct {
	Consumer AuthConsumer `json:"consumer"`
	Session  AuthSession  `json:"session"`
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
	ConsumerTest         AuthConsumerType = "futurama"
	ConsumerTest2        AuthConsumerType = "planet-express"
)

// IsValid returns validity of given auth consumer type.
func (t AuthConsumerType) IsValid() bool {
	switch t {
	case ConsumerBuiltin, ConsumerLocal:
		return true
	}
	return t.IsValidExternal()
}

// IsValidExternal returns validity of given auth consumer type.
func (t AuthConsumerType) IsValidExternal() bool {
	switch t {
	case ConsumerLDAP, ConsumerCorporateSSO, ConsumerGithub, ConsumerGitlab, ConsumerTest, ConsumerTest2:
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

// AuthConsumerWarningType constant for consumer warnings.
type AuthConsumerWarningType string

// Consumer warning types.
const (
	WarningGroupInvalid     AuthConsumerWarningType = "group-invalid"
	WarningGroupRemoved     AuthConsumerWarningType = "group-removed"
	WarningLastGroupRemoved AuthConsumerWarningType = "last-group-removed"
)

// AuthConsumerWarnings contains specific information from the auth driver.
type AuthConsumerWarnings []AuthConsumerWarning

// NewConsumerWarningGroupInvalid returns a new warning for given group info.
func NewConsumerWarningGroupInvalid(groupID int64, groupName string) AuthConsumerWarning {
	return AuthConsumerWarning{
		Type:      WarningGroupInvalid,
		GroupID:   groupID,
		GroupName: groupName,
	}
}

// NewConsumerWarningGroupRemoved returns a new warning for given group info.
func NewConsumerWarningGroupRemoved(groupID int64, groupName string) AuthConsumerWarning {
	return AuthConsumerWarning{
		Type:      WarningGroupRemoved,
		GroupID:   groupID,
		GroupName: groupName,
	}
}

// NewConsumerWarningLastGroupRemoved returns a new warning.
func NewConsumerWarningLastGroupRemoved() AuthConsumerWarning {
	return AuthConsumerWarning{Type: WarningLastGroupRemoved}
}

// AuthConsumerWarning contains info about a warning.
type AuthConsumerWarning struct {
	Type      AuthConsumerWarningType `json:"type"`
	GroupID   int64                   `json:"group_id,omitempty"`
	GroupName string                  `json:"group_name,omitempty"`
}

// Scan consumer data.
func (w *AuthConsumerWarnings) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, w), "cannot unmarshal AuthConsumerWarnings")
}

// Value returns driver.Value from consumer warnings.
func (w AuthConsumerWarnings) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal AuthConsumerWarnings")
}

// AuthConsumers gives functions for auth consumer slice.
type AuthConsumers []AuthConsumer

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
	InvalidGroupIDs    Int64Slice             `json:"invalid_group_ids,omitempty" db:"invalid_group_ids"`
	Scopes             AuthConsumerScopeSlice `json:"scopes,omitempty" cli:"scopes" db:"scopes"`
	IssuedAt           time.Time              `json:"issued_at" cli:"issued_at" db:"issued_at"`
	Disabled           bool                   `json:"disabled" cli:"disabled" db:"disabled"`
	Warnings           AuthConsumerWarnings   `json:"warnings,omitempty" db:"warnings"`
	// aggregates
	AuthentifiedUser *AuthentifiedUser `json:"user,omitempty" db:"-"`
	Groups           Groups            `json:"groups,omitempty" db:"-"`
	Service          *Service          `json:"-" db:"-"`
	Worker           *Worker           `json:"-" db:"-"`
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
	} else if c.AuthentifiedUser != nil {
		groupIDs = c.AuthentifiedUser.GetGroupIDs()
	}

	return groupIDs
}

func (c AuthConsumer) Admin() bool {
	return c.AuthentifiedUser.Ring == UserRingAdmin
}

func (c AuthConsumer) Maintainer() bool {
	return c.AuthentifiedUser.Ring == UserRingMaintainer
}

func (c AuthConsumer) GetUsername() string {
	if c.Service != nil || c.Worker != nil {
		return c.Name
	}
	return c.AuthentifiedUser.GetUsername()
}

func (c AuthConsumer) GetEmail() string {
	if c.Service != nil || c.Worker != nil {
		return ""
	}
	return c.AuthentifiedUser.GetEmail()
}

func (c AuthConsumer) GetFullname() string {
	if c.Service != nil || c.Worker != nil {
		return c.Name
	}
	return c.AuthentifiedUser.GetFullname()
}

// AuthSessions gives functions for auth session slice.
type AuthSessions []AuthSession

// AuthSession struct.
type AuthSession struct {
	ID         string    `json:"id" cli:"id,key" db:"id"`
	ConsumerID string    `json:"consumer_id" cli:"consumer_id" db:"consumer_id"`
	ExpireAt   time.Time `json:"expire_at,omitempty" cli:"expire_at" db:"expire_at"`
	Created    time.Time `json:"created" cli:"created" db:"created"`
	MFA        bool      `json:"mfa" cli:"mfa" db:"mfa"`
	// aggregates
	Consumer *AuthConsumer `json:"consumer,omitempty" db:"-"`
	Groups   []Group       `json:"groups,omitempty" db:"-"`
	Current  bool          `json:"current,omitempty" cli:"current" db:"-"`
}

// AuthSessionJWTClaims is the specific claims format for JWT session.
type AuthSessionJWTClaims struct {
	ID string
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

const AuthSigninConsumerTokenDuration time.Duration = time.Minute * 5

// AuthSigninConsumerToken discribes the payload for a signin state token.
type AuthSigninConsumerToken struct {
	IssuedAt          int64  `json:"issued_at"`
	Origin            string `json:"origin,omitempty"`
	RedirectURI       string `json:"redirect_uri,omitempty"`
	RequireMFA        bool   `json:"require_mfa,omitempty"`
	IsFirstConnection bool   `json:"is_first_connection,omitempty"`
}
