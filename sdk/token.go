package sdk

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk/log"
)

const (
	AccessTokenStatusEnabled  = "enabled"
	AccessTokenStatusDisabled = "disabled"
)

// AccessTokenRequest a the type used by clients to ask a new access_token
type AccessTokenRequest struct {
	GroupsIDs             []int64  `json:"groups"`
	Scopes                []string `json:"scopes"`
	Description           string   `json:"description"`
	Origin                string   `json:"origin"`
	ExpirationDelaySecond float64  `json:"expiration_delay_second"`
}

// APIConsumer is a user granted from a JWT token. It can be a service, a worker, a hatchery or a user
type APIConsumer struct {
	Name       string
	Groups     []Group
	OnBehalfOf AuthentifiedUser
}

func (g *APIConsumer) IsGranted() bool {
	granted := g != nil
	log.Debug("APIConsumer.IsGranted> granted: %t", granted)
	return granted
}

func (g *APIConsumer) GetGroups() []Group {
	if !g.IsGranted() {
		return nil
	}
	return g.Groups
}

func (g *APIConsumer) Admin() bool {
	if !g.IsGranted() {
		return false
	}
	admin := g.OnBehalfOf.Admin()
	log.Debug("APIConsumer.Admin> consumer on behalf of user %s is admin: %t", g.OnBehalfOf.GetFullname(), admin)
	return admin
}

func (g *APIConsumer) Maintainer() bool {
	if !g.IsGranted() {
		return false
	}
	return g.OnBehalfOf.Maintainer()
}

func (g *APIConsumer) GetConsumerName() string {
	return g.Name
}

func (g *APIConsumer) GetUsername() string {
	return g.OnBehalfOf.Username
}

func (g *APIConsumer) GetFullname() string {
	return g.OnBehalfOf.Fullname
}

func (g *APIConsumer) GetEmail() string {
	return g.OnBehalfOf.GetEmail()
}

func (g *APIConsumer) GetDEPRECATEDUserStruct() *User {
	if g.IsGranted() {
		return nil
	}
	if g.OnBehalfOf.OldUserStruct == nil {
		return nil
	}
	return g.OnBehalfOf.OldUserStruct
}

// AccessToken is either a Personnal Access Token or a Group Access Token
type AccessToken struct {
	ID                 string      `json:"id" cli:"id,key" db:"id"`
	Name               string      `json:"name" cli:"name" db:"name"`
	AuthentifiedUserID string      `json:"user_id,omitempty" db:"user_id"`
	ExpireAt           time.Time   `json:"expired_at,omitempty" cli:"expired_at" db:"expired_at"`
	Created            time.Time   `json:"created" cli:"created" db:"created"`
	Status             string      `json:"status" cli:"status" db:"status"`
	Origin             string      `json:"-" cli:"-" db:"origin"`
	Groups             Groups      `json:"groups" cli:"groups" db:"-"`
	Scopes             StringSlice `json:"scopes" cli:"scopes" db:"scopes"`
	// aggregates
	AuthentifiedUser *AuthentifiedUser `json:"user" db:"-"`
}

// AccessTokensToIDs returns ids of given access tokens.
func AccessTokensToIDs(ats []*AccessToken) []string {
	ids := make([]string, len(ats))
	for i := range ats {
		ids[i] = ats[i].ID
	}
	return ids
}

// AccessTokensToIDs returns ids of given access tokens.
func AccessTokensToAuthentifiedUserIDs(ats []*AccessToken) []string {
	ids := make([]string, len(ats))
	for i := range ats {
		ids[i] = ats[i].AuthentifiedUserID
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

// AccessTokenJWTClaims is the specific claims format for JWT Tokens
type AccessTokenJWTClaims struct {
	ID     string
	Groups []int64
	Scopes []string
	jwt.StandardClaims
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
