package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DeleteUserHandler removes a user
func (api *API) deleteUserHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !getUser(ctx).Admin && username != getUser(ctx).Username {
			return WriteJSON(w, nil, http.StatusForbidden)
		}

		u, errLoad := user.LoadUserWithoutAuth(api.mustDB(), username)
		if errLoad != nil {
			return sdk.WrapError(errLoad, "deleteUserHandler> Cannot load user from db")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "deleteUserHandler> cannot start transaction")
		}
		defer tx.Rollback()

		if err := user.DeleteUserWithDependencies(tx, u); err != nil {
			return sdk.WrapError(err, "deleteUserHandler> cannot delete user")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteUserHandler> cannot commit transaction")
		}

		return nil
	}
}

// GetUserHandler returns a specific user's information
func (api *API) getUserHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !getUser(ctx).Admin && username != getUser(ctx).Username {
			return WriteJSON(w, nil, http.StatusForbidden)
		}

		u, err := user.LoadUserWithoutAuth(api.mustDB(), username)
		if err != nil {
			return sdk.WrapError(err, "getUserHandler: Cannot load user from db")
		}

		if err = loadUserPermissions(api.mustDB(), api.Cache, u); err != nil {
			return sdk.WrapError(err, "getUserHandler: Cannot get user group and project from db")
		}

		return WriteJSON(w, u, http.StatusOK)
	}
}

// getUserGroupsHandler returns groups of the user
func (api *API) getUserGroupsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !getUser(ctx).Admin && username != getUser(ctx).Username {
			return WriteJSON(w, nil, http.StatusForbidden)
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

		return WriteJSON(w, res, http.StatusOK)
	}
}

// UpdateUserHandler modifies user informations
func (api *API) updateUserHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !getUser(ctx).Admin && username != getUser(ctx).Username {
			return WriteJSON(w, nil, http.StatusForbidden)
		}

		userDB, errload := user.LoadUserWithoutAuth(api.mustDB(), username)
		if errload != nil {
			return sdk.WrapError(errload, "getUserHandler: Cannot load user from db")
		}

		var userBody sdk.User
		if err := UnmarshalBody(r, &userBody); err != nil {
			return err
		}

		userBody.ID = userDB.ID

		if !user.IsValidEmail(userBody.Email) {
			return sdk.WrapError(sdk.ErrWrongRequest, "updateUserHandler: Email address %s is not valid", userBody.Email)
		}

		if err := user.UpdateUser(api.mustDB(), userBody); err != nil {
			return sdk.WrapError(err, "updateUserHandler: Cannot update user table")
		}

		return WriteJSON(w, userBody, http.StatusOK)
	}
}

// GetUsers fetches all users from databases
func (api *API) getUsersHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		users, err := user.LoadUsers(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "GetUsers: Cannot load user from db")
		}
		return WriteJSON(w, users, http.StatusOK)
	}
}

// postUserFavoriteHandler post favorite user for workflow or project
func (api *API) postUserFavoriteHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		params := sdk.FavoriteParams{}
		if err := UnmarshalBody(r, &params); err != nil {
			return err
		}

		switch params.Type {
		case "workflow":
			wf, errW := workflow.Load(api.mustDB(), api.Cache, params.ProjectKey, params.WorkflowName, getUser(ctx), workflow.LoadOptions{WithFavorites: true})
			if errW != nil {
				return sdk.WrapError(errW, "postUserFavoriteHandler> Cannot load workflow %s/%s", params.ProjectKey, params.WorkflowName)
			}

			if err := workflow.UpdateFavorite(api.mustDB(), wf.ID, getUser(ctx), !wf.Favorite); err != nil {
				return sdk.WrapError(err, "postUserFavoriteHandler> Cannot change workflow %s/%s favorite", params.ProjectKey, params.WorkflowName)
			}
			wf.Favorite = !wf.Favorite

			return WriteJSON(w, wf, http.StatusOK)
		case "project":
			p, errProj := project.Load(api.mustDB(), api.Cache, params.ProjectKey, getUser(ctx), project.LoadOptions.WithFavorites)
			if errProj != nil {
				return sdk.WrapError(errProj, "postUserFavoriteHandler> Cannot load project %s", params.ProjectKey)
			}

			if err := project.UpdateFavorite(api.mustDB(), p.ID, getUser(ctx), !p.Favorite); err != nil {
				return sdk.WrapError(err, "postUserFavoriteHandler> Cannot change workflow %s favorite", p.Key)
			}
			p.Favorite = !p.Favorite

			return WriteJSON(w, p, http.StatusOK)
		}

		return sdk.ErrInvalidFavoriteType
	}
}

// AddUser creates a new user and generate verification email
func (api *API) addUserHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//returns forbidden if LDAP mode is activated
		if _, ldap := api.Router.AuthDriver.(*auth.LDAPClient); ldap {
			return sdk.ErrForbidden
		}

		createUserRequest := sdk.UserAPIRequest{}
		if err := UnmarshalBody(r, &createUserRequest); err != nil {
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

		go mail.SendMailVerifyToken(createUserRequest.User.Email, createUserRequest.User.Username, tokenVerify, createUserRequest.Callback)

		// If it's the first user, add him to shared.infra group
		if nbUsers == 0 {
			if err := group.AddAdminInGlobalGroup(api.mustDB(), u.ID); err != nil {
				return sdk.WrapError(err, "AddUser: Cannot add user in global group")
			}
		}

		return WriteJSON(w, u, http.StatusCreated)
	}
}

func (api *API) resetUserHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//returns forbidden if LDAP mode is activated
		if _, ldap := api.Router.AuthDriver.(*auth.LDAPClient); ldap {
			return sdk.ErrForbidden
		}

		// Get username in URL
		vars := mux.Vars(r)
		username := vars["username"]

		resetUserRequest := sdk.UserAPIRequest{}
		if err := UnmarshalBody(r, &resetUserRequest); err != nil {
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

		return WriteJSON(w, userDb, http.StatusCreated)
	}
}

//AuthModeHandler returns the auth mode : local ok ldap
func (api *API) authModeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		mode := "local"
		if _, ldap := api.Router.AuthDriver.(*auth.LDAPClient); ldap {
			mode = "ldap"
		}
		res := map[string]string{
			"auth_mode": mode,
		}
		return WriteJSON(w, res, http.StatusOK)
	}
}

// ConfirmUser verify token send via email and mark user as verified
func (api *API) confirmUserHandler() Handler {
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
		return WriteJSON(w, response, http.StatusOK)
	}
}

// LoginUser take user credentials and creates a auth token
func (api *API) loginUserHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		loginUserRequest := sdk.UserLoginRequest{}
		if err := UnmarshalBody(r, &loginUserRequest); err != nil {
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
		if errl != nil && errl == sql.ErrNoRows {
			return sdk.WrapError(sdk.ErrInvalidUser, "Auth> Login error %s: %s", loginUserRequest.Username, errl)
		}
		if errl != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "Auth> Login error %s: %s", loginUserRequest.Username, errl)
		}

		// Prepare response
		response := sdk.UserAPIResponse{
			User: *u,
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
		return WriteJSON(w, response, http.StatusOK)
	}
}

func (api *API) importUsersHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var users = []sdk.User{}
		if err := UnmarshalBody(r, &users); err != nil {
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

		return WriteJSON(w, errors, http.StatusOK)
	}
}

func (api *API) getUserTokenListHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tokens, err := token.LoadTokens(api.mustDB(), getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getUserTokenListHandler> cannot load group for user %s", getUser(ctx).Username)
		}

		return WriteJSON(w, tokens, http.StatusOK)
	}
}

func (api *API) getUserTokenHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		tok, err := token.LoadTokenWithGroup(api.mustDB(), vars["token"])
		if err == sdk.ErrInvalidToken {
			return sdk.ErrTokenNotFound
		}
		if err != nil {
			return sdk.WrapError(err, "getUserTokenHandler> cannot load token for user %s", getUser(ctx).Username)
		}
		tok.Token = ""

		return WriteJSON(w, tok, http.StatusOK)
	}
}
