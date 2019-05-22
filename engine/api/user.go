package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

// DeleteUserHandler removes a user
func (api *API) deleteUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !deprecatedGetUser(ctx).Admin && username != deprecatedGetUser(ctx).Username {
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}

		u, errLoad := user.LoadUserWithoutAuth(api.mustDB(), username)
		if errLoad != nil {
			return sdk.WrapError(errLoad, "Cannot load user from db")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := user.DeleteUserWithDependencies(tx, u); err != nil {
			return sdk.WrapError(err, "cannot delete user")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		return nil
	}
}

// GetUserHandler returns a specific user's information
func (api *API) getUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]
		withFavoritesWorkflows := FormBool(r, "withFavoritesWorkflows")
		withFavoritesProjects := FormBool(r, "withFavoritesProjects")

		if !deprecatedGetUser(ctx).Admin && username != deprecatedGetUser(ctx).Username {
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}

		u, err := user.LoadUserWithoutAuth(api.mustDB(), username)
		if err != nil {
			return sdk.WrapError(err, "getUserHandler: Cannot load user from db")
		}

		if err = loadUserPermissions(api.mustDB(), api.Cache, u); err != nil {
			return sdk.WrapError(err, "getUserHandler: Cannot get user group and project from db")
		}

		if withFavoritesWorkflows {
			favoritesWorkflows, err := workflow.LoadFavorites(ctx, api.mustDB(), api.Cache, deprecatedGetUser(ctx))
			if err != nil {
				return sdk.WrapError(err, "unable to load favorites workflows")
			}
			u.FavoritesWorkflows = favoritesWorkflows
		}

		if withFavoritesProjects {
			favoritesProjects, err := project.LoadFavorites(ctx, api.mustDB(), api.Cache, deprecatedGetUser(ctx))
			if err != nil {
				return sdk.WrapError(err, "unable to load favorites projects")
			}
			u.FavoritesProjects = favoritesProjects
		}

		return service.WriteJSON(w, u, http.StatusOK)
	}
}

// getUserGroupsHandler returns groups of the user
func (api *API) getUserGroupsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !deprecatedGetUser(ctx).Admin && username != deprecatedGetUser(ctx).Username {
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}

		u, errl := user.LoadUserWithoutAuth(api.mustDB(), username)
		if errl != nil {
			return sdk.WrapError(errl, "getUserHandler: Cannot load user from db")
		}

		var groups, groupsAdmin []sdk.Group

		var err1, err2 error
		groups, err1 = group.LoadGroupByUser(api.mustDB(), u.ID)
		if err1 != nil {
			return sdk.WrapError(err1, "getUserGroupsHandler: Cannot load group by user")
		}

		groupsAdmin, err2 = group.LoadGroupByAdmin(api.mustDB(), u.ID)
		if err2 != nil {
			return sdk.WrapError(err2, "getUserGroupsHandler: Cannot load group by admin")
		}

		res := map[string][]sdk.Group{}
		res["groups"] = groups
		res["groups_admin"] = groupsAdmin

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

// UpdateUserHandler modifies user informations
func (api *API) updateUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !deprecatedGetUser(ctx).Admin && username != deprecatedGetUser(ctx).Username {
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}

		userDB, errload := user.LoadUserWithoutAuth(api.mustDB(), username)
		if errload != nil {
			return sdk.WrapError(errload, "getUserHandler: Cannot load user from db")
		}

		var userBody sdk.User
		if err := service.UnmarshalBody(r, &userBody); err != nil {
			return err
		}

		userBody.ID = userDB.ID

		if !user.IsValidEmail(userBody.Email) {
			return sdk.WrapError(sdk.ErrWrongRequest, "updateUserHandler: Email address %s is not valid", userBody.Email)
		}

		if err := user.UpdateUser(api.mustDB(), userBody); err != nil {
			return sdk.WrapError(err, "updateUserHandler: Cannot update user table")
		}

		return service.WriteJSON(w, userBody, http.StatusOK)
	}
}

// GetUsers fetches all users from databases
func (api *API) getUsersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		users, err := user.LoadUsers(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "GetUsers: Cannot load user from db")
		}
		return service.WriteJSON(w, users, http.StatusOK)
	}
}

// getUserLoggedHandler check if the current user is connected
func (api *API) getUserLoggedHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		h := r.Header.Get(sdk.SessionTokenHeader)
		if h == "" {
			return sdk.ErrUnauthorized
		}

		store := api.Router.AuthDriver.Store()
		key := sessionstore.SessionKey(h)
		if ok, _ := store.Exists(key); !ok {
			return sdk.ErrUnauthorized
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getTimelineFilterHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := deprecatedGetUser(ctx)
		filter, err := user.LoadTimelineFilter(api.mustDB(), u)
		if err != nil {
			return sdk.WrapError(err, "getTimelineFilterHandler")
		}
		return service.WriteJSON(w, filter, http.StatusOK)
	}
}

func (api *API) postTimelineFilterHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := deprecatedGetUser(ctx)
		var timelineFilter sdk.TimelineFilter
		if err := service.UnmarshalBody(r, &timelineFilter); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		// Try to load
		count, errLoad := user.CountTimelineFilter(api.mustDB(), u)
		if errLoad != nil {
			return sdk.WrapError(errLoad, "Cannot load filter")
		}
		if count == 0 {
			if err := user.InsertTimelineFilter(api.mustDB(), timelineFilter, u); err != nil {
				return sdk.WrapError(err, "Cannot insert filter")
			}
		} else {
			if err := user.UpdateTimelineFilter(api.mustDB(), timelineFilter, u); err != nil {
				return sdk.WrapError(err, "Unable to update filter")
			}
		}
		return nil
	}
}

// postUserFavoriteHandler post favorite user for workflow or project
func (api *API) postUserFavoriteHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		params := sdk.FavoriteParams{}
		if err := service.UnmarshalBody(r, &params); err != nil {
			return err
		}

		p, err := project.Load(api.mustDB(), api.Cache, params.ProjectKey, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations, project.LoadOptions.WithFavorites)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		switch params.Type {
		case "workflow":
			wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, p, params.WorkflowName, deprecatedGetUser(ctx), workflow.LoadOptions{WithFavorites: true})
			if errW != nil {
				return sdk.WrapError(errW, "postUserFavoriteHandler> Cannot load workflow %s/%s", params.ProjectKey, params.WorkflowName)
			}

			if err := workflow.UpdateFavorite(api.mustDB(), wf.ID, deprecatedGetUser(ctx), !wf.Favorite); err != nil {
				return sdk.WrapError(err, "Cannot change workflow %s/%s favorite", params.ProjectKey, params.WorkflowName)
			}
			wf.Favorite = !wf.Favorite

			return service.WriteJSON(w, wf, http.StatusOK)
		case "project":
			if err := project.UpdateFavorite(api.mustDB(), p.ID, deprecatedGetUser(ctx), !p.Favorite); err != nil {
				return sdk.WrapError(err, "Cannot change workflow %s favorite", p.Key)
			}
			p.Favorite = !p.Favorite

			return service.WriteJSON(w, p, http.StatusOK)
		}

		return sdk.ErrInvalidFavoriteType
	}
}

// AddUser creates a new user and generate verification email
func (api *API) addUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//returns forbidden if LDAP mode is activated
		if _, ldap := api.Router.AuthDriver.(*auth.LDAPClient); ldap {
			return sdk.ErrForbidden
		}

		createUserRequest := sdk.UserAPIRequest{}
		if err := service.UnmarshalBody(r, &createUserRequest); err != nil {
			return err
		}

		if createUserRequest.User.Username == "" {
			return sdk.WrapError(sdk.ErrInvalidUsername, "AddUser: Empty username is invalid")
		}

		if !user.IsValidEmail(createUserRequest.User.Email) {
			return sdk.WrapError(sdk.ErrInvalidEmail, "AddUser: Email address %s is not valid", createUserRequest.User.Email)
		}

		if !user.IsAllowedDomain(api.Config.Auth.Local.SignupAllowedDomains, createUserRequest.User.Email) {
			return sdk.WrapError(sdk.ErrInvalidEmailDomain, "AddUser: Email address %s does not have a valid domain. Allowed domains:%v", createUserRequest.User.Email, api.Config.Auth.Local.SignupAllowedDomains)
		}

		u := createUserRequest.User
		u.Origin = "local"

		// Check that user does not already exists
		query := `SELECT * FROM "user" WHERE username = $1`
		rows, err := api.mustDB().Query(query, u.Username)
		if err != nil {
			return sdk.WrapError(err, "AddUsers: Cannot check if user %s exist", u.Username)
		}
		defer rows.Close()
		if rows.Next() {
			return sdk.WrapError(sdk.ErrUserConflict, "AddUser: User %s already exists", u.Username)
		}

		tokenVerify, hashedToken, errg := user.GeneratePassword()
		if errg != nil {
			return sdk.WrapError(errg, "AddUser: Error while generate Token Verify for new user")
		}

		auth := sdk.NewAuth(hashedToken)

		nbUsers, errc := user.CountUser(api.mustDB())
		if errc != nil {
			return sdk.WrapError(errc, "AddUser: Cannot count user")
		}
		if nbUsers == 0 {
			u.Admin = true
		} else {
			u.Admin = false
		}

		if err := user.InsertUser(api.mustDB(), &u, auth); err != nil {
			return sdk.WrapError(err, "AddUser: Cannot insert user")
		}

		go func() {
			if errM := mail.SendMailVerifyToken(createUserRequest.User.Email, createUserRequest.User.Username, tokenVerify, createUserRequest.Callback); errM != nil {
				log.Warning("addUserHandler.SendMailVerifyToken> Cannot send verify token email for user %s : %v", u.Username, errM)
			}
		}()

		// If it's the first user, add him to shared.infra group
		if nbUsers == 0 {
			if err := group.AddAdminInGlobalGroup(api.mustDB(), u.ID); err != nil {
				return sdk.WrapError(err, "AddUser: Cannot add user in global group")
			}
		}

		return service.WriteJSON(w, u, http.StatusCreated)
	}
}

func (api *API) resetUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//returns forbidden if LDAP mode is activated
		if _, ldap := api.Router.AuthDriver.(*auth.LDAPClient); ldap {
			return sdk.ErrForbidden
		}

		// Get username in URL
		vars := mux.Vars(r)
		username := vars["username"]

		resetUserRequest := sdk.UserAPIRequest{}
		if err := service.UnmarshalBody(r, &resetUserRequest); err != nil {
			return err
		}

		// Load user
		userDb, err := user.LoadUserAndAuth(api.mustDB(), username)
		if err != nil || userDb.Email != resetUserRequest.User.Email {
			return sdk.WrapError(sdk.ErrInvalidResetUser, "Cannot load user: %s", err)
		}

		tokenVerify, hashedToken, err := user.GeneratePassword()
		if err != nil {
			return sdk.WrapError(err, "Error while generate Token Verify for new user")
		}
		userDb.Auth.HashedTokenVerify = hashedToken
		userDb.Auth.DateReset = time.Now().Unix()

		// Update in db
		if err := user.UpdateUserAndAuth(api.mustDB(), *userDb); err != nil {
			return sdk.WrapError(err, "ResetUser: Cannot update user %s", userDb.Username)
		}

		go mail.SendMailVerifyToken(userDb.Email, userDb.Username, tokenVerify, resetUserRequest.Callback)

		return service.WriteJSON(w, userDb, http.StatusCreated)
	}
}

//AuthModeHandler returns the auth mode : local ok ldap
func (api *API) authModeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		mode := "local"
		if _, ldap := api.Router.AuthDriver.(*auth.LDAPClient); ldap {
			mode = "ldap"
		}
		res := map[string]string{
			"auth_mode": mode,
		}
		return service.WriteJSON(w, res, http.StatusOK)
	}
}

// ConfirmUser verify token send via email and mark user as verified
func (api *API) confirmUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//returns forbidden if LDAP mode is activated
		if _, ldap := api.Router.AuthDriver.(*auth.LDAPClient); ldap {
			return sdk.ErrForbidden
		}

		// Get user name in URL
		vars := mux.Vars(r)
		username := vars["username"]
		token := vars["token"]

		if username == "" || token == "" {
			return sdk.ErrInvalidUsername
		}

		// Load user
		u, err := user.LoadUserAndAuth(api.mustDB(), username)
		if err != nil {
			return sdk.ErrInvalidUsername
		}

		// Verify token
		password, hashedPassword, err := user.Verify(u, token)
		if err != nil {
			return sdk.ErrUnauthorized
		}

		u.Auth.EmailVerified = true
		u.Auth.HashedPassword = hashedPassword
		u.Auth.DateReset = 0

		// Update in db
		if err := user.UpdateUserAndAuth(api.mustDB(), *u); err != nil {
			return err
		}

		var response = sdk.UserAPIResponse{
			User: *u,
		}

		var logFromCLI bool
		if r.Header.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
			log.Info("LoginUser> login from CLI")
			logFromCLI = true
		}

		var sessionKey sessionstore.SessionKey
		var errs error
		if !logFromCLI {
			sessionKey, errs = auth.NewSession(api.Router.AuthDriver, u)
			if errs != nil {
				log.Error("Auth> Error while creating new session: %s\n", errs)
			}
		} else {
			//CLI login, generate user key as persistent session
			sessionKey, errs = auth.NewPersistentSession(api.mustDB(), api.Router.AuthDriver, u)
			if errs != nil {
				log.Error("Auth> Error while creating new session: %s\n", errs)
			}
		}

		response.Token = string(sessionKey)
		response.Password = password

		response.User.Auth = sdk.Auth{}
		return service.WriteJSON(w, response, http.StatusOK)
	}
}

// LoginUser take user credentials and creates a auth token
func (api *API) loginUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		loginUserRequest := sdk.UserLoginRequest{}
		if err := service.UnmarshalBody(r, &loginUserRequest); err != nil {
			return err
		}

		var logFromCLI bool
		if r.Header.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
			log.Info("LoginUser> login from CLI")
			logFromCLI = true
		}

		// Authentify user through authDriver
		authOK, erra := api.Router.AuthDriver.Authentify(loginUserRequest.Username, loginUserRequest.Password)
		if erra != nil {
			return sdk.WrapError(sdk.ErrInvalidUser, "Auth> Login error %s: %s", loginUserRequest.Username, erra)
		}
		if !authOK {
			return sdk.WrapError(sdk.ErrInvalidUser, "Auth> Login failed: %s", loginUserRequest.Username)
		}
		// Load user
		u, errl := user.LoadUserWithoutAuth(api.mustDB(), loginUserRequest.Username)
		if errl != nil && sdk.Cause(errl) == sql.ErrNoRows {
			return sdk.WrapError(sdk.ErrInvalidUser, "Auth> Login error %s: %s", loginUserRequest.Username, errl)
		}
		if errl != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "Auth> Login error %s: %s", loginUserRequest.Username, errl)
		}

		// Prepare response
		response := sdk.UserAPIResponse{
			User: *u,
		}

		// If there is a request token, store it (for 30 minutes)
		if loginUserRequest.RequestToken != "" {
			var accessTokenRequest sdk.AccessTokenRequest
			if err := jws.UnsafeParse(loginUserRequest.RequestToken, &accessTokenRequest); err != nil {
				return sdk.WithStack(err)
			}
			token, _, err := api.createNewAccessToken(*u, accessTokenRequest)
			if err != nil {
				return sdk.WithStack(err)
			}
			api.Cache.SetWithTTL("api:loginUserHandler:RequestToken:"+loginUserRequest.RequestToken, token, 30*60)
		}

		if err := group.CheckUserInDefaultGroup(api.mustDB(), u.ID); err != nil {
			log.Warning("Auth> Error while check user in default group:%s\n", err)
		}

		var sessionKey sessionstore.SessionKey
		var errs error
		if !logFromCLI {
			//Standard login, new session
			sessionKey, errs = auth.NewSession(api.Router.AuthDriver, u)
			if errs != nil {
				log.Error("Auth> Error while creating new session: %s\n", errs)
			}
		} else {
			//CLI login, generate user key as persistent session
			sessionKey, errs = auth.NewPersistentSession(api.mustDB(), api.Router.AuthDriver, u)
			if errs != nil {
				log.Error("Auth> Error while creating new session: %s\n", errs)
			}
		}

		if sessionKey != "" {
			w.Header().Set(sdk.SessionTokenHeader, string(sessionKey))
			response.Token = string(sessionKey)
		}

		response.User.Auth = sdk.Auth{}
		response.User.Permissions = sdk.UserPermissions{}
		return service.WriteJSON(w, response, http.StatusOK)
	}
}

func (api *API) importUsersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var users = []sdk.User{}
		if err := service.UnmarshalBody(r, &users); err != nil {
			return err
		}

		_, hashedToken, err := user.GeneratePassword()
		if err != nil {
			return sdk.WrapError(err, "Error while generate Token Verify for new user")
		}

		errors := map[string]string{}
		for _, u := range users {
			if err := user.InsertUser(api.mustDB(), &u, &sdk.Auth{
				EmailVerified:  true,
				DateReset:      0,
				HashedPassword: hashedToken,
			}); err != nil {
				oldU, err := user.LoadUserWithoutAuth(api.mustDB(), u.Username)
				if err != nil {
					errors[u.Username] = err.Error()
					continue
				}
				u.ID = oldU.ID
				u.Auth = sdk.Auth{
					EmailVerified: true,
					DateReset:     0,
				}
				if err := user.UpdateUserAndAuth(api.mustDB(), u); err != nil {
					errors[u.Username] = err.Error()
				}
			}
		}

		return service.WriteJSON(w, errors, http.StatusOK)
	}
}

func (api *API) getUserTokenListHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tokens, err := token.LoadTokens(api.mustDB(), deprecatedGetUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load group for user %s", deprecatedGetUser(ctx).Username)
		}

		return service.WriteJSON(w, tokens, http.StatusOK)
	}
}

func (api *API) getUserTokenHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		tok, err := token.LoadTokenWithGroup(api.mustDB(), vars["token"])
		if sdk.ErrorIs(err, sdk.ErrInvalidToken) {
			return sdk.ErrTokenNotFound
		}
		if err != nil {
			return sdk.WrapError(err, "Cannot load token for user %s", deprecatedGetUser(ctx).Username)
		}
		tok.Token = ""

		return service.WriteJSON(w, tok, http.StatusOK)
	}
}

func (api *API) loginUserCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// api.Cache.SetWithTTL("api:loginUserHandler:RequestToken:"+loginUserRequest.RequestToken, token, 30*60)
		var request sdk.UserLoginCallbackRequest
		if err := service.UnmarshalBody(r, &request); err != nil {
			return sdk.WithStack(err)
		}

		var accessToken sdk.AccessToken
		if !api.Cache.Get("api:loginUserHandler:RequestToken:"+request.RequestToken, &accessToken) {
			return sdk.ErrNotFound
		}

		pk, err := jws.NewPublicKeyFromPEM(request.PublicKey)
		if err != nil {
			log.Debug("unable to read public key: %v", string(request.PublicKey))
			return sdk.WithStack(err)
		}

		var x sdk.AccessTokenRequest
		if err := jws.Verify(pk, request.RequestToken, &x); err != nil {
			return sdk.WithStack(err)
		}

		jwt, err := accesstoken.Regen(&accessToken)
		if err != nil {
			return sdk.WithStack(err)
		}

		w.Header().Add("X-CDS-JWT", jwt)

		return service.WriteJSON(w, accessToken, http.StatusOK)
	}
}
