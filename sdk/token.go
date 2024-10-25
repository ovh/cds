package sdk

import (
	"context"
	"database/sql/driver"
	json "encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

type Driver interface {
	GetUserInfoFromDriver(ctx context.Context, req AuthConsumerSigninRequest) (AuthDriverUserInfo, error)
}

type DriverWithSignInRequest interface {
	Driver
	CheckSigninRequest(req AuthConsumerSigninRequest) error
}

type DriverWithRedirect interface {
	DriverWithSignInRequest
	GetSigninURI(AuthSigninConsumerToken) (AuthDriverSigningRedirect, error)
}

type DriverWithSigninStateToken interface {
	DriverWithSignInRequest
	CheckSigninStateToken(AuthConsumerSigninRequest) error
}

// AuthDriver interface.
type AuthDriver interface {
	GetManifest() AuthDriverManifest
	GetSessionDuration() time.Duration
	GetUserInfo(context.Context, AuthConsumerSigninRequest) (AuthDriverUserInfo, error)
	GetDriver() Driver
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
	SupportMFA     bool             `json:"support_mfa"`
}

// AuthConsumerScope alias type for string.
type AuthConsumerScope string

type AuthConsumerScopeEndpoints []AuthConsumerScopeEndpoint

func (e AuthConsumerScopeEndpoints) FindEndpoint(route string) (bool, AuthConsumerScopeEndpoint) {
	for i := range e {
		if e[i].Route == route {
			return true, e[i]
		}
	}
	return false, AuthConsumerScopeEndpoint{}
}

type AuthConsumerScopeEndpoint struct {
	Route   string      `json:"route"`
	Methods StringSlice `json:"methods,omitempty"`
}

// AuthConsumerScopeDetail contains all endpoints for a scope.
type AuthConsumerScopeDetail struct {
	Scope     AuthConsumerScope          `json:"scope"`
	Endpoints AuthConsumerScopeEndpoints `json:"endpoints,omitempty"`
}

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
	AuthConsumerScopeWorkerModel,
	AuthConsumerScopeHatchery,
	AuthConsumerScopeService,
}

func NewAuthConsumerScopeDetails(scopes ...AuthConsumerScope) AuthConsumerScopeDetails {
	ds := make(AuthConsumerScopeDetails, len(scopes))
	for i := range scopes {
		ds[i] = AuthConsumerScopeDetail{
			Scope: scopes[i],
		}
	}
	return ds
}

// AuthConsumerScopeDetails type used for database json storage.
type AuthConsumerScopeDetails []AuthConsumerScopeDetail

// IsValid returns an error if current details are invalids.
func (s AuthConsumerScopeDetails) IsValid() error {
	// Check that each scope is unique
	mScope := map[AuthConsumerScope]struct{}{}
	for _, detail := range s {
		if _, ok := mScope[detail.Scope]; ok {
			return NewErrorFrom(ErrWrongRequest, "duplicated scope %s in given details", detail.Scope)
		}
		mScope[detail.Scope] = struct{}{}

		// Check that each route is unique for scope
		mRoute := map[string]struct{}{}
		for _, endpoint := range detail.Endpoints {
			if _, ok := mRoute[endpoint.Route]; ok {
				return NewErrorFrom(ErrWrongRequest, "duplicated route %s for scope %s in given details", endpoint.Route, detail.Scope)
			}
			mRoute[endpoint.Route] = struct{}{}

			// Check that each method is unique for scope and match GET, POST, PUT or DELETE
			mMethod := map[string]struct{}{}
			for _, method := range endpoint.Methods {
				if _, ok := mMethod[method]; ok {
					return NewErrorFrom(ErrWrongRequest, "duplicated method %s for route %s and scope %s in given details", method, endpoint.Route, detail.Scope)
				}
				mMethod[method] = struct{}{}
				if !(method == http.MethodGet || method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete) {
					return NewErrorFrom(ErrWrongRequest, "invalid method %s for route %s and scope %s in given details", method, endpoint.Route, detail.Scope)
				}
			}
		}
	}

	return nil
}

func (s AuthConsumerScopeDetails) ToEndpointsMap() map[AuthConsumerScope]AuthConsumerScopeEndpoints {
	m := make(map[AuthConsumerScope]AuthConsumerScopeEndpoints, len(s))
	for i := range s {
		m[s[i].Scope] = s[i].Endpoints
	}
	return m
}

// Scan scope detail slice.
func (s *AuthConsumerScopeDetails) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(JSONUnmarshal(source, s), "cannot unmarshal AuthConsumerScopeDetails")
}

// Value returns driver.Value from scope detail slice.
func (s AuthConsumerScopeDetails) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal AuthConsumerScopeDetails")
}

// AuthConsumerScopeSlice type used for database json storage.
type AuthConsumerScopeSlice []AuthConsumerScope

// Scan scope slice.
func (s *AuthConsumerScopeSlice) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(JSONUnmarshal(source, s), "cannot unmarshal AuthConsumerScopeSlice")
}

// Value returns driver.Value from scope slice.
func (s AuthConsumerScopeSlice) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal AuthConsumerScopeSlice")
}

// AuthConsumerRegenRequest struct.
type AuthConsumerRegenRequest struct {
	RevokeSessions  bool   `json:"revoke_sessions"`
	OverlapDuration string `json:"overlap_duration"`
	NewDuration     int64  `json:"new_duration"`
}

// AuthConsumerSigninRequest struct for auth consumer signin request.
type AuthConsumerSigninRequest map[string]interface{}

func (req AuthConsumerSigninRequest) String(s string) string {
	return cast.ToString(req[s])
}

func (req AuthConsumerSigninRequest) StringE(s string) (string, error) {
	val, err := cast.ToStringE(req[s])
	if err != nil {
		return "", NewError(ErrWrongRequest, err)
	}
	return val, nil
}

// AuthConsumerSigninResponse response for a auth consumer signin.
type AuthConsumerSigninResponse struct {
	APIURL  string            `json:"api_url,omitempty"`
	Token   string            `json:"token"` // session token
	User    *AuthentifiedUser `json:"user"`
	Service *Service          `json:"service,omitempty"`
}

// AuthConsumerCreateResponse response for a auth consumer creation.
type AuthConsumerCreateResponse struct {
	Token    string            `json:"token"` // sign in token
	Consumer *AuthUserConsumer `json:"consumer"`
}

// AuthDriverUserInfo struct discribed a user returns by a auth driver.
type AuthDriverUserInfo struct {
	ExternalID      string
	Username        string
	Fullname        string
	Email           string
	MFA             bool
	ExternalTokenID string
	Organization    string
}

// AuthCurrentConsumerResponse describe the current consumer and the current session
type AuthCurrentConsumerResponse struct {
	User           AuthentifiedUser   `json:"user"`
	Consumer       AuthUserConsumer   `json:"consumer"`
	Session        AuthSession        `json:"session"`
	DriverManifest AuthDriverManifest `json:"driver_manifest"`
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
	ConsumerOIDC         AuthConsumerType = "openid-connect"
	ConsumerHatchery     AuthConsumerType = "hatchery"
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
	case ConsumerLDAP, ConsumerCorporateSSO, ConsumerGithub, ConsumerGitlab, ConsumerOIDC, ConsumerTest, ConsumerTest2:
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
	return WrapError(JSONUnmarshal(source, d), "cannot unmarshal AuthConsumerData")
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
	return WrapError(JSONUnmarshal(source, w), "cannot unmarshal AuthConsumerWarnings")
}

// Value returns driver.Value from consumer warnings.
func (w AuthConsumerWarnings) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal AuthConsumerWarnings")
}

// AuthUserConsumers gives functions for auth consumer slice.
type AuthUserConsumers []AuthUserConsumer

type AuthConsumer struct {
	ID                 string                      `json:"id" cli:"id,key" db:"id"`
	Name               string                      `json:"name" cli:"name" db:"name"`
	Type               AuthConsumerType            `json:"type" cli:"type" db:"type"`
	Description        string                      `json:"description" cli:"description" db:"description"`
	ParentID           *string                     `json:"parent_id,omitempty" db:"parent_id"`
	Created            time.Time                   `json:"created" cli:"created" db:"created"`
	DeprecatedIssuedAt time.Time                   `json:"issued_at" cli:"issued_at" db:"issued_at"`
	Disabled           bool                        `json:"disabled" cli:"disabled" db:"disabled"`
	Warnings           AuthConsumerWarnings        `json:"warnings,omitempty" db:"warnings"`
	LastAuthentication *time.Time                  `json:"last_authentication,omitempty" db:"last_authentication"`
	ValidityPeriods    AuthConsumerValidityPeriods `json:"validity_periods,omitempty" db:"validity_periods"`
}

// AuthUserConsumer issues session linked to an authentified user.
type AuthUserConsumer struct {
	AuthConsumer
	AuthConsumerUser AuthUserConsumerData `json:"auth_consumer_user,omitempty" db:"-"`
}

// AuthHatcheryConsumer issues session linked to an authentified user.
type AuthHatcheryConsumer struct {
	AuthConsumer
	AuthConsumerHatchery AuthConsumerHatcheryData `json:"auth_consumer_hatchery,omitempty" db:"-"`
}

// AuthUserConsumerData issues session linked to an authentified user.
type AuthUserConsumerData struct {
	ID                           string                   `json:"id" cli:"id,key" db:"id"`
	AuthConsumerID               string                   `json:"auth_consumer_id" cli:"auth_consumer_id,key" db:"auth_consumer_id"`
	AuthentifiedUserID           string                   `json:"user_id,omitempty" db:"user_id"`
	Data                         AuthConsumerData         `json:"-" db:"data"` // NEVER returns auth consumer data in json, TODO this fields should be visible only in auth package
	GroupIDs                     Int64Slice               `json:"group_ids,omitempty" cli:"group_ids" db:"group_ids"`
	InvalidGroupIDs              Int64Slice               `json:"invalid_group_ids,omitempty" db:"invalid_group_ids"`
	ScopeDetails                 AuthConsumerScopeDetails `json:"scope_details,omitempty" cli:"scope_details" db:"scope_details"`
	ServiceName                  *string                  `json:"service_name,omitempty" db:"service_name"`
	ServiceType                  *string                  `json:"service_type,omitempty" db:"service_type"`
	ServiceRegion                *string                  `json:"service_region,omitempty" db:"service_region"`
	ServiceIgnoreJobWithNoRegion *bool                    `json:"service_ignore_job_with_no_region,omitempty" db:"service_ignore_job_with_no_region"`
	// aggregates
	AuthentifiedUser *AuthentifiedUser `json:"user,omitempty" db:"-"`
	Groups           Groups            `json:"groups,omitempty" db:"-"`
	// aggregates by router auth middleware
	Service *Service `json:"-" db:"-"`
	Worker  *Worker  `json:"-" db:"-"`
}

type AuthConsumerHatcheryData struct {
	ID             string `json:"id" cli:"id,key" db:"id"`
	AuthConsumerID string `json:"auth_consumer_id" cli:"auth_consumer_id,key" db:"auth_consumer_id"`
	HatcheryID     string `json:"hatchery_id" cli:"hatchery_id" db:"hatchery_id"`
}

func NewAuthConsumerValidityPeriod(iat time.Time, duration time.Duration) AuthConsumerValidityPeriods {
	return AuthConsumerValidityPeriods{
		{
			IssuedAt: iat,
			Duration: duration,
		},
	}
}

type AuthConsumerValidityPeriods []AuthConsumerValidityPeriod

func (p AuthConsumerValidityPeriods) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, WrapError(err, "cannot marshal AuthConsumerValidityPeriods")
}

func (p *AuthConsumerValidityPeriods) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, p), "cannot unmarshal AuthConsumerValidityPeriods")
}

func (p AuthConsumerValidityPeriods) Latest() *AuthConsumerValidityPeriod {
	if len(p) == 0 {
		return nil
	}
	p.Sort()
	return &p[0]
}

func (p *AuthConsumerValidityPeriods) Sort() {
	sort.Slice(*p, func(i, j int) bool {
		return (*p)[j].IssuedAt.Before((*p)[i].IssuedAt)
	})
}

type AuthConsumerValidityPeriod struct {
	IssuedAt time.Time     `json:"issued_at" cli:"issued_at" `
	Duration time.Duration `json:"duration" cli:"duration"`
}

// IsValid returns validity for auth consumer.
func (c AuthUserConsumer) IsValid(scopeDetails AuthConsumerScopeDetails) error {
	if c.Name == "" {
		return NewErrorFrom(ErrWrongRequest, "invalid given name")
	}

	if err := c.AuthConsumerUser.ScopeDetails.IsValid(); err != nil {
		return err
	}

	mEndpoints := scopeDetails.ToEndpointsMap()

	for _, s := range c.AuthConsumerUser.ScopeDetails {
		if !s.Scope.IsValid() {
			return NewErrorFrom(ErrWrongRequest, "invalid given scope value %s", s)
		}

		// Checks that given route/method restriction match existing routes
		existingEndpoints := mEndpoints[s.Scope]
		for _, e := range s.Endpoints {
			exists, existingEndpoint := existingEndpoints.FindEndpoint(e.Route)
			if !exists {
				return NewErrorFrom(ErrWrongRequest, "invalid given route %s for scope %s", e.Route, s)
			}

			for _, m := range e.Methods {
				if !existingEndpoint.Methods.Contains(m) {
					return NewErrorFrom(ErrWrongRequest, "invalid given method %s for route %s with scope %s", m, e.Route, s)
				}
			}
		}
	}

	return nil
}

// GetGroupIDs returns group ids for auth consumer, if empty
// in consumer returns group ids from authentified user.
func (c AuthUserConsumer) GetGroupIDs() []int64 {
	var groupIDs []int64

	if len(c.AuthConsumerUser.GroupIDs) > 0 {
		groupIDs = c.AuthConsumerUser.GroupIDs
	} else if c.AuthConsumerUser.AuthentifiedUser != nil && c.AuthConsumerUser.Worker == nil {
		groupIDs = c.AuthConsumerUser.AuthentifiedUser.GetGroupIDs()
	}

	return groupIDs
}

func (c AuthUserConsumer) Admin() bool {
	// Worker and Service can't be considered as admin
	return c.AuthConsumerUser.AuthentifiedUser.Ring == UserRingAdmin && c.AuthConsumerUser.Worker == nil && c.AuthConsumerUser.Service == nil
}

func (c AuthUserConsumer) Maintainer() bool {
	return (c.AuthConsumerUser.AuthentifiedUser.Ring == UserRingMaintainer || c.AuthConsumerUser.AuthentifiedUser.Ring == UserRingAdmin) && c.AuthConsumerUser.Worker == nil && c.AuthConsumerUser.Service == nil
}

func (c AuthUserConsumer) GetUsername() string {
	if c.AuthConsumerUser.Service != nil || c.AuthConsumerUser.Worker != nil {
		return c.Name
	}
	return c.AuthConsumerUser.AuthentifiedUser.GetUsername()
}

func (c AuthUserConsumer) GetEmail() string {
	if c.AuthConsumerUser.Service != nil || c.AuthConsumerUser.Worker != nil {
		return ""
	}
	return c.AuthConsumerUser.AuthentifiedUser.GetEmail()
}

func (c AuthUserConsumer) GetFullname() string {
	if c.AuthConsumerUser.Service != nil || c.AuthConsumerUser.Worker != nil {
		return c.Name
	}
	return c.AuthConsumerUser.AuthentifiedUser.GetFullname()
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
	Consumer     *AuthUserConsumer `json:"consumer,omitempty" db:"-"`
	Groups       []Group           `json:"groups,omitempty" db:"-"`
	Current      bool              `json:"current,omitempty" cli:"current" db:"-"`
	LastActivity *time.Time        `json:"last_activity,omitempty" cli:"last_activity,omitempty" db:"-"`
}

// AuthSessionJWTClaims is the specific claims format for JWT session.
type AuthSessionJWTClaims struct {
	ID      string
	TokenID string
	jwt.StandardClaims
}

// AuthSessionsToIDs returns ids of given auth sessions.

// AuthConsumersToIDs returns ids of given auth consumers.
func AuthConsumersToIDs(cs []AuthUserConsumer) []string {
	ids := make([]string, len(cs))
	for i := range cs {
		ids[i] = cs[i].ID
	}
	return ids
}

// AuthConsumersToAuthentifiedUserIDs returns ids of given auth consumers.
func AuthConsumersToAuthentifiedUserIDs(cs []*AuthUserConsumer) []string {
	ids := make([]string, len(cs))
	for i := range cs {
		ids[i] = cs[i].AuthConsumerUser.AuthentifiedUserID
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

// Expiration defines how worker key should expire
type Expiration int

const AuthSigninConsumerTokenDuration time.Duration = time.Minute * 5

// AuthSigninConsumerToken discribes the payload for a signin state token.
type AuthSigninConsumerToken struct {
	IssuedAt          int64  `json:"issued_at"`
	Origin            string `json:"origin,omitempty"`
	RedirectURI       string `json:"redirect_uri,omitempty"`
	RequireMFA        bool   `json:"require_mfa,omitempty"`
	IsFirstConnection bool   `json:"is_first_connection,omitempty"`
	LinkUser          bool   `json:"link_user,omitempty"`
}
