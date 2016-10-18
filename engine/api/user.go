package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

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
func DeleteUserHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	username := vars["name"]

	u, err := user.LoadUserWithoutAuth(db, username)
	if err != nil {
		log.Warning("deleteUserHandler> Cannot load user from db: %s\n", err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteUserHandler> cannot start transaction: %s", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = user.DeleteUserWithDependencies(tx, u)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteUserHandler> cannot commit transaction: %s", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetUserHandler returns a specific user's information
func GetUserHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	username := vars["name"]

	u, err := user.LoadUserWithoutAuth(db, username)
	if err != nil {
		fmt.Printf("getUserHandler: Cannot load user from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = user.LoadUserPermissions(db, u)
	if err != nil {
		fmt.Printf("getUserHandler: Cannot get user group and project from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, u, http.StatusOK)

}

// UpdateUserHandler modifies user informations
func UpdateUserHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	username := vars["name"]

	userDB, err := user.LoadUserWithoutAuth(db, username)
	if err != nil {
		fmt.Printf("getUserHandler: Cannot load user from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var userBody sdk.User
	err = json.Unmarshal(data, &userBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userBody.ID = userDB.ID

	if !user.IsValidEmail(userBody.Email) {
		log.Warning("updateUserHandler: Email address %s is not valid", userBody.Email)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = user.UpdateUser(db, userBody)
	if err != nil {
		log.Warning("updateUserHandler: Cannot update user table: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

}

// GetUsers fetches all users from databases
func GetUsers(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	users, err := user.LoadUsers(db)
	if err != nil {
		log.Warning("GetUsers: Cannot load user from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, users, http.StatusOK)
}

// AddUser creates a new user and generate verification email
func AddUser(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	//returns forbidden if LDAP mode is activated
	if _, ldap := router.authDriver.(*auth.LDAPClient); ldap {
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	createUserRequest := sdk.UserAPIRequest{}
	err = json.Unmarshal(data, &createUserRequest)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	if createUserRequest.User.Username == "" {
		log.Warning("AddUser: Empty username is invalid")
		WriteError(w, r, sdk.ErrInvalidUsername)
		return
	}

	if !user.IsValidEmail(createUserRequest.User.Email) {
		log.Warning("AddUser: Email address %s is not valid", createUserRequest.User.Email)
		WriteError(w, r, sdk.ErrInvalidEmail)
		return
	}

	u := createUserRequest.User
	u.Origin = "local"

	// Check that user does not already exists
	query := `SELECT * FROM "user" WHERE username = $1`
	rows, err := db.Query(query, u.Username)
	if err != nil {
		log.Warning("AddUsers: Cannot check if user %s exist: %s\n", u.Username, err)
		WriteError(w, r, err)
		return
	}
	defer rows.Close()
	if rows.Next() {
		log.Warning("AddUser: User %s already exists\n", u.Username)
		WriteError(w, r, sdk.ErrUserConflict)
		return
	}
	tokenVerify, hashedToken, err := user.GeneratePassword()
	if err != nil {
		log.Warning("AddUser: Error while generate Token Verify for new user %s \n", err)
		WriteError(w, r, err)
		return
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
		WriteError(w, r, err)
		return
	}

	go mail.SendMailVerifyToken(createUserRequest.User.Email, createUserRequest.User.Username, tokenVerify, createUserRequest.Callback)

	// If it's the first user, add him to shared.infra group
	if nbUsers == 0 {
		err = group.AddAdminInGlobalGroup(db, u.ID)
		if err != nil {
			log.Warning("AddUser: Cannot add user in global group: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

// ResetUser deletes auth secret, generates new ones and send them via email
func ResetUser(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	//returns forbidden if LDAP mode is activated
	if _, ldap := router.authDriver.(*auth.LDAPClient); ldap {
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Get user name in URL
	vars := mux.Vars(r)
	username := vars["name"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	resetUserRequest := sdk.UserAPIRequest{}
	err = json.Unmarshal(data, &resetUserRequest)
	if err != nil {
		log.Warning("Cannot unmarshal body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Load user
	userDb, err := user.LoadUserAndAuth(db, username)
	if err != nil || userDb.Email != resetUserRequest.User.Email {
		log.Warning("Cannot load user: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidResetUser)
		return
	}

	tokenVerify, hashedToken, err := user.GeneratePassword()
	if err != nil {
		log.Warning("Error while generate Token Verify for new user %s \n", err)
		WriteError(w, r, err)
		return
	}
	userDb.Auth.HashedTokenVerify = hashedToken
	userDb.Auth.DateReset = time.Now().Unix()

	// Update in db
	query := `UPDATE "user" SET auth = $1 WHERE username = $2`
	_, err = db.Exec(query, userDb.Auth.JSON(), userDb.Username)
	if err != nil {
		log.Warning("ResetUser: Cannot update user %s: %s\n", userDb.Username, err)
		WriteError(w, r, err)
		return
	}

	go mail.SendMailVerifyToken(userDb.Email, userDb.Username, tokenVerify, resetUserRequest.Callback)

	log.Warning("POST /user/%s/reset: User reset OK\n", userDb.Username)
	w.WriteHeader(http.StatusCreated)

}

//AuthModeHandler returns the auth mode : local ok ldap
func AuthModeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	mode := "local"
	if _, ldap := router.authDriver.(*auth.LDAPClient); ldap {
		mode = "ldap"
	}
	res := map[string]string{
		"auth_mode": mode,
	}
	WriteJSON(w, r, res, http.StatusOK)
}

// ConfirmUser verify token send via email and mark user as verified
func ConfirmUser(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	//returns forbidden if LDAP mode is activated
	if _, ldap := router.authDriver.(*auth.LDAPClient); ldap {
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Get user name in URL
	vars := mux.Vars(r)
	name := vars["name"]
	token := vars["token"]

	if name == "" || token == "" {
		WriteError(w, r, sdk.ErrInvalidUsername)
		return
	}

	// Load user
	u, err := user.LoadUserAndAuth(db, name)
	if err != nil {
		fmt.Printf("ConfirmUser: Cannot load %s: %s\n", name, err)
		WriteError(w, r, sdk.ErrInvalidUsername)
		return
	}

	// Verify token
	password, hashedPassword, err := user.Verify(u, token)
	if err != nil {
		WriteError(w, r, sdk.ErrUnauthorized)
		return
	}

	u.Auth.EmailVerified = true
	u.Auth.HashedPassword = hashedPassword
	u.Auth.DateReset = 0

	// Update in db
	query := `UPDATE "user" SET data = $1, auth = $2 WHERE username = $3`
	_, err = db.Exec(query, u.JSON(), u.Auth.JSON(), u.Username)
	if err != nil {
		WriteError(w, r, err)
		return
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
	WriteJSON(w, r, response, http.StatusOK)
}

// LoginUser take user credentials and creates a auth token
func LoginUser(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	loginUserRequest := sdk.UserLoginRequest{}
	err = json.Unmarshal(data, &loginUserRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var logFromCLI bool
	if r.Header.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
		log.Notice("LoginUser> login from CLI")
		logFromCLI = true
	}

	// Authentify user through authDriver
	authOK, err := router.authDriver.Authentify(loginUserRequest.Username, loginUserRequest.Password)
	if err != nil {
		log.Warning("Auth> Login error %s :%s\n", loginUserRequest.Username, err)
		WriteError(w, r, sdk.ErrInvalidUser)
		return
	}
	if !authOK {
		log.Warning("Auth> Login failed: %s\n", loginUserRequest.Username)
		WriteError(w, r, sdk.ErrInvalidUser)
		return
	}
	// Load user
	u, err := user.LoadUserWithoutAuth(db, loginUserRequest.Username)
	if err != nil && err == sql.ErrNoRows {
		log.Warning("Auth> Login error %s :%s\n", loginUserRequest.Username, err)
		WriteError(w, r, sdk.ErrInvalidUser)
		return
	}
	if err != nil {
		log.Warning("Auth> Login error %s :%s\n", loginUserRequest.Username, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Prepare response
	response := sdk.UserAPIResponse{
		User: *u,
	}

	// If "session" mode is activated, generate a new session
	if _, local := router.authDriver.(*auth.LocalClient); !local || localCLientAuthMode != auth.LocalClientBasicAuthMode {
		var sessionKey sessionstore.SessionKey
		if !logFromCLI {
			//Standard login, new session
			sessionKey, err = auth.NewSession(router.authDriver, u)
			if err != nil {
				log.Critical("Auth> Error while creating new session: %s\n", err)
			}
		} else {
			//CLI login, generate user key as persistent session
			sessionKey, err = auth.NewPersistentSession(db, router.authDriver, u)
			if err != nil {
				log.Critical("Auth> Error while creating new session: %s\n", err)
			}
		}

		if sessionKey != "" {
			w.Header().Set(sdk.SessionTokenHeader, string(sessionKey))
			response.Token = string(sessionKey)
		}
	}

	response.User.Auth = sdk.Auth{}
	WriteJSON(w, r, response, http.StatusOK)
}
