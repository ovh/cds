package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// DeleteUserHandler removes a user
func DeleteUserHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	username := vars["name"]

	u, err := user.LoadUserWithoutAuth(db, username)
	if err != nil {
		log.Warning("deleteUserHandler> Cannot load user from db: %s\n", err)
		return err

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteUserHandler> cannot start transaction: %s", err)
		return err

	}
	defer tx.Rollback()

	err = user.DeleteUserWithDependencies(tx, u)
	if err != nil {
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteUserHandler> cannot commit transaction: %s", err)
		return err

	}

	return nil
}

// GetUserHandler returns a specific user's information
func GetUserHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	username := vars["name"]

	u, err := user.LoadUserWithoutAuth(db, username)
	if err != nil {
		fmt.Printf("getUserHandler: Cannot load user from db: %s\n", err)
		return err
	}

	if err = loadUserPermissions(db, u); err != nil {
		fmt.Printf("getUserHandler: Cannot get user group and project from db: %s\n", err)
		return err
	}

	return WriteJSON(w, r, u, http.StatusOK)
}

// getUserGroupsHandler returns groups of the user
func getUserGroupsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	log.Debug("getUserGroupsHandler> get groups for user %d", c.User.ID)

	var groups, groupsAdmin []sdk.Group

	//Admin are considered as admin of all groups
	if c.User.Admin {
		allgroups, err := group.LoadGroups(db)
		if err != nil {
			return err

		}

		groups = allgroups
		groupsAdmin = allgroups
	} else {
		var err1, err2 error

		groups, err1 = group.LoadGroupByUser(db, c.User.ID)
		if err1 != nil {
			return err1

		}

		groupsAdmin, err2 = group.LoadGroupByAdmin(db, c.User.ID)
		if err2 != nil {
			return err2
		}
	}

	res := map[string][]sdk.Group{}
	res["groups"] = groups
	res["groups_admin"] = groupsAdmin

	return WriteJSON(w, r, res, http.StatusOK)
}

// UpdateUserHandler modifies user informations
func UpdateUserHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	username := vars["name"]

	userDB, err := user.LoadUserWithoutAuth(db, username)
	if err != nil {
		fmt.Printf("getUserHandler: Cannot load user from db: %s\n", err)
		return err

	}

	var userBody sdk.User
	if err := UnmarshalBody(r, &userBody); err != nil {
		return err
	}

	userBody.ID = userDB.ID

	if !user.IsValidEmail(userBody.Email) {
		log.Warning("updateUserHandler: Email address %s is not valid", userBody.Email)
		return sdk.ErrWrongRequest
	}

	err = user.UpdateUser(db, userBody)
	if err != nil {
		log.Warning("updateUserHandler: Cannot update user table: %s", err)
		return err
	}

	return nil
}

// GetUsers fetches all users from databases
func GetUsers(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	users, err := user.LoadUsers(db)
	if err != nil {
		log.Warning("GetUsers: Cannot load user from db: %s\n", err)
		return err
	}

	return WriteJSON(w, r, users, http.StatusOK)
}

// AddUser creates a new user and generate verification email
func AddUser(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	//returns forbidden if LDAP mode is activated
	if _, ldap := router.authDriver.(*auth.LDAPClient); ldap {
		return sdk.ErrForbidden
	}

	createUserRequest := sdk.UserAPIRequest{}
	if err := UnmarshalBody(r, &createUserRequest); err != nil {
		return err
	}

	if createUserRequest.User.Username == "" {
		log.Warning("AddUser: Empty username is invalid")
		return sdk.ErrInvalidUsername
	}

	if !user.IsValidEmail(createUserRequest.User.Email) {
		log.Warning("AddUser: Email address %s is not valid", createUserRequest.User.Email)
		return sdk.ErrInvalidEmail
	}

	u := createUserRequest.User
	u.Origin = "local"

	// Check that user does not already exists
	query := `SELECT * FROM "user" WHERE username = $1`
	rows, err := db.Query(query, u.Username)
	if err != nil {
		log.Warning("AddUsers: Cannot check if user %s exist: %s\n", u.Username, err)
		return err
	}
	defer rows.Close()
	if rows.Next() {
		log.Warning("AddUser: User %s already exists\n", u.Username)
		return sdk.ErrUserConflict

	}
	tokenVerify, hashedToken, err := user.GeneratePassword()
	if err != nil {
		log.Warning("AddUser: Error while generate Token Verify for new user %s \n", err)
		return err
	}

	auth := sdk.NewAuth(hashedToken)

	nbUsers, err := user.CountUser(db)
	if err != nil {
		log.Warning("AddUser: Cannot count user %s \n", err)
	}
	if nbUsers == 0 {
		u.Admin = true
	} else {
		u.Admin = false
	}

	err = user.InsertUser(db, &u, auth)
	if err != nil {
		log.Warning("AddUser: Cannot insert user: %s\n", err)
		return err

	}

	go mail.SendMailVerifyToken(createUserRequest.User.Email, createUserRequest.User.Username, tokenVerify, createUserRequest.Callback)

	// If it's the first user, add him to shared.infra group
	if nbUsers == 0 {
		err = group.AddAdminInGlobalGroup(db, u.ID)
		if err != nil {
			log.Warning("AddUser: Cannot add user in global group: %s\n", err)
			return err

		}
	}

	return WriteJSON(w, r, u, http.StatusCreated)
}

// ResetUser deletes auth secret, generates new ones and send them via email
func ResetUser(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	//returns forbidden if LDAP mode is activated
	if _, ldap := router.authDriver.(*auth.LDAPClient); ldap {
		return sdk.ErrForbidden

	}

	// Get user name in URL
	vars := mux.Vars(r)
	username := vars["name"]

	resetUserRequest := sdk.UserAPIRequest{}
	if err := UnmarshalBody(r, &resetUserRequest); err != nil {
		return err
	}

	// Load user
	userDb, err := user.LoadUserAndAuth(db, username)
	if err != nil || userDb.Email != resetUserRequest.User.Email {
		log.Warning("Cannot load user: %s\n", err)
		return sdk.ErrInvalidResetUser
	}

	tokenVerify, hashedToken, err := user.GeneratePassword()
	if err != nil {
		log.Warning("Error while generate Token Verify for new user %s \n", err)
		return err
	}
	userDb.Auth.HashedTokenVerify = hashedToken
	userDb.Auth.DateReset = time.Now().Unix()

	// Update in db
	query := `UPDATE "user" SET auth = $1 WHERE username = $2`
	_, err = db.Exec(query, userDb.Auth.JSON(), userDb.Username)
	if err != nil {
		log.Warning("ResetUser: Cannot update user %s: %s\n", userDb.Username, err)
		return err
	}

	go mail.SendMailVerifyToken(userDb.Email, userDb.Username, tokenVerify, resetUserRequest.Callback)

	log.Warning("POST /user/%s/reset: User reset OK\n", userDb.Username)
	return WriteJSON(w, r, userDb, http.StatusCreated)
}

//AuthModeHandler returns the auth mode : local ok ldap
func AuthModeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	mode := "local"
	if _, ldap := router.authDriver.(*auth.LDAPClient); ldap {
		mode = "ldap"
	}
	res := map[string]string{
		"auth_mode": mode,
	}
	return WriteJSON(w, r, res, http.StatusOK)
}

// ConfirmUser verify token send via email and mark user as verified
func ConfirmUser(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	//returns forbidden if LDAP mode is activated
	if _, ldap := router.authDriver.(*auth.LDAPClient); ldap {
		return sdk.ErrForbidden

	}

	// Get user name in URL
	vars := mux.Vars(r)
	name := vars["name"]
	token := vars["token"]

	if name == "" || token == "" {
		return sdk.ErrInvalidUsername
	}

	// Load user
	u, err := user.LoadUserAndAuth(db, name)
	if err != nil {
		fmt.Printf("ConfirmUser: Cannot load %s: %s\n", name, err)
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
	query := `UPDATE "user" SET data = $1, auth = $2 WHERE username = $3`
	_, err = db.Exec(query, u.JSON(), u.Auth.JSON(), u.Username)
	if err != nil {
		return err
	}

	var response = sdk.UserAPIResponse{
		User: *u,
	}
	if _, local := router.authDriver.(*auth.LocalClient); !local || localCLientAuthMode != auth.LocalClientBasicAuthMode {
		sessionKey, err := auth.NewSession(router.authDriver, u)
		if err != nil {
			log.Critical("Auth> Error while creating new session: %s\n", err)
		}

		if sessionKey != "" {
			response.Token = string(sessionKey)
		}
	}

	//If authDriver is local, we send the password.
	//BTW forgotten password process should not be available in ldap mode.
	if _, ok := router.authDriver.(*auth.LocalClient); ok {
		response.Password = password
	}

	response.User.Auth = sdk.Auth{}
	return WriteJSON(w, r, response, http.StatusOK)
}

// LoginUser take user credentials and creates a auth token
func LoginUser(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	loginUserRequest := sdk.UserLoginRequest{}
	if err := UnmarshalBody(r, &loginUserRequest); err != nil {
		return err
	}

	var logFromCLI bool
	if r.Header.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
		log.Notice("LoginUser> login from CLI")
		logFromCLI = true
	}

	// Authentify user through authDriver
	authOK, erra := router.authDriver.Authentify(db, loginUserRequest.Username, loginUserRequest.Password)
	if erra != nil {
		log.Warning("Auth> Login error %s :%s\n", loginUserRequest.Username, erra)
		return sdk.ErrInvalidUser

	}
	if !authOK {
		log.Warning("Auth> Login failed: %s\n", loginUserRequest.Username)
		return sdk.ErrInvalidUser

	}
	// Load user
	u, errl := user.LoadUserWithoutAuth(db, loginUserRequest.Username)
	if errl != nil && errl == sql.ErrNoRows {
		log.Warning("Auth> Login error %s :%s\n", loginUserRequest.Username, errl)
		return sdk.ErrInvalidUser
	}
	if errl != nil {
		log.Warning("Auth> Login error %s :%s\n", loginUserRequest.Username, errl)
		return sdk.ErrWrongRequest
	}

	// Prepare response
	response := sdk.UserAPIResponse{
		User: *u,
	}

	if err := group.CheckUserInDefaultGroup(db, u.ID); err != nil {
		log.Warning("Auth> Error while check user in default group:%s\n", err)
	}

	// If "session" mode is activated, generate a new session
	if _, local := router.authDriver.(*auth.LocalClient); !local || localCLientAuthMode != auth.LocalClientBasicAuthMode {
		var sessionKey sessionstore.SessionKey
		var errs error
		if !logFromCLI {
			//Standard login, new session
			sessionKey, errs = auth.NewSession(router.authDriver, u)
			if errs != nil {
				log.Critical("Auth> Error while creating new session: %s\n", errs)
			}
		} else {
			//CLI login, generate user key as persistent session
			sessionKey, errs = auth.NewPersistentSession(db, router.authDriver, u)
			if errs != nil {
				log.Critical("Auth> Error while creating new session: %s\n", errs)
			}
		}

		if sessionKey != "" {
			w.Header().Set(sdk.SessionTokenHeader, string(sessionKey))
			response.Token = string(sessionKey)
		}
	}

	response.User.Auth = sdk.Auth{}
	return WriteJSON(w, r, response, http.StatusOK)
}

func importUsersHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var users = []sdk.User{}
	if err := UnmarshalBody(r, &users); err != nil {
		return err
	}

	_, hashedToken, err := user.GeneratePassword()
	if err != nil {
		log.Warning("Error while generate Token Verify for new user %s \n", err)
		return err
	}

	errors := map[string]string{}
	for _, u := range users {
		if err := user.InsertUser(db, &u, &sdk.Auth{
			EmailVerified:  true,
			DateReset:      0,
			HashedPassword: hashedToken,
		}); err != nil {
			oldU, err := user.LoadUserWithoutAuth(db, u.Username)
			if err != nil {
				errors[u.Username] = err.Error()
				continue
			}
			u.ID = oldU.ID
			u.Auth = sdk.Auth{
				EmailVerified: true,
				DateReset:     0,
			}
			if err := user.UpdateUserAndAuth(db, u); err != nil {
				errors[u.Username] = err.Error()
			}
		}
	}

	return WriteJSON(w, r, errors, http.StatusOK)
}
