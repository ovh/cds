package auth

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk/log"
)

//LocalClient is a auth driver which store all in database
type LocalClient struct {
	dbFunc func() *gorp.DbMap
}

//Open nothing
func (c *LocalClient) Init(options interface{}) error {
	return nil
}

//CheckAuth checks the auth
// func (c *LocalClient) CheckAuth(ctx context.Context, w http.ResponseWriter, req *http.Request) (context.Context, error) {
// 	//Check persistent session
// 	if req.Header.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
// 		var ok bool
// 		ctx, ok = getUserPersistentSession(ctx, c.dbFunc(), c.Store(), req.Header)
// 		if ok {
// 			return ctx, nil
// 		}
// 		return ctx, fmt.Errorf("invalid session")
// 	}
//
// 	//Check other session
// 	sessionToken := req.Header.Get(sdk.SessionTokenHeader)
// 	if sessionToken == "" {
// 		//Accept session in request
// 		sessionToken = req.FormValue("session")
// 	}
// 	if sessionToken == "" {
// 		return ctx, fmt.Errorf("no session header")
// 	}
//
// 	exists, err := c.store.Exists(sessionstore.SessionKey(sessionToken))
// 	if err != nil {
// 		return ctx, err
// 	}
// 	username, err := GetUsername(c.store, sessionToken)
// 	if err != nil {
// 		return ctx, err
// 	}
// 	u, err := user.LoadUserAndAuth(c.dbFunc(), username)
// 	if err != nil {
// 		return ctx, fmt.Errorf("authorization failed for %s: %s", username, err)
// 	}
// 	ctx = context.WithValue(ctx, ContextUser, u)
// 	ctx = context.WithValue(ctx, ContextUserSession, sessionToken)
//
// 	if !exists {
// 		return ctx, fmt.Errorf("invalid session")
// 	}
//
// 	return ctx, nil
// }

//Authentify check username and password
func (c *LocalClient) Authentify(username, password string) (bool, error) {
	// Load user
	u, err := user.LoadUserAndAuth(c.dbFunc(), username)
	if err != nil {
		log.Warning("Auth> Authorization failed")
		return false, err
	}

	b := user.IsCheckValid(password, u.Auth.HashedPassword)
	return b, err
}
