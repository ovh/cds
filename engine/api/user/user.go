package user

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Verify verify user token
func Verify(u *sdk.User, token string) (string, string, error) {
	if u.Auth.EmailVerified && u.Auth.DateReset == 0 {
		return "", "", errors.New("Account already verified")
	}

	if u.Auth.DateReset != 0 && time.Since(time.Unix(u.Auth.DateReset, 0)).Minutes() > 30 {
		return "", "", fmt.Errorf("Reset operation expired")
	}

	if err := checkToken(u, token); err != nil {
		return "", "", err
	}
	return regenerateAndStoreAuth()
}

func checkToken(u *sdk.User, token string) error {
	if !IsCheckValid(token, u.Auth.HashedTokenVerify) {
		return fmt.Errorf("Error while checking user %s with given token", u.Username)
	}
	return nil
}

func regenerateAndStoreAuth() (string, string, error) {
	password, hashedPassword, err := GeneratePassword()
	if err != nil {
		return "", "", err
	}
	return password, hashedPassword, err
}

// IsValidEmail  Check if user email address is ok
func IsValidEmail(email string) bool {
	// check email
	regexp := regexp.MustCompile(sdk.UserEmailPattern)
	return regexp.MatchString(email)
}

// IsAllowedDomain return true is email is allowed, false otherwise
func IsAllowedDomain(allowedDomains string, email string) bool {
	if allowedDomains != "" {
		allowedDomains := strings.Split(allowedDomains, ",")
		for _, domain := range allowedDomains {
			if strings.HasSuffix(email, "@"+domain) && strings.Count(email, "@") == 1 {
				return true
			}
		}
		return false
	}
	// no restriction, domain is ok
	return true
}

// LoadUserWithoutAuthByID load user information without secret
func LoadUserWithoutAuthByID(db gorp.SqlExecutor, userID int64) (*sdk.User, error) {
	query := `SELECT username, admin, data, origin FROM "user" WHERE id = $1`

	var jsonUser []byte
	var username, origin string
	var admin bool

	if err := db.QueryRow(query, userID).Scan(&username, &admin, &jsonUser, &origin); err != nil {
		return nil, err
	}

	// Load user
	u := &sdk.User{}
	if err := json.Unmarshal(jsonUser, u); err != nil {
		return nil, err
	}

	u.Admin = admin
	u.ID = userID
	u.Origin = origin
	return u, nil
}

// LoadUserWithoutAuth load user without auth information
func LoadUserWithoutAuth(db gorp.SqlExecutor, name string) (*sdk.User, error) {
	query := `SELECT id, admin, data, origin FROM "user" WHERE username = $1`

	var jsonUser []byte
	var id int64
	var admin bool
	var origin string

	if err := db.QueryRow(query, name).Scan(&id, &admin, &jsonUser, &origin); err != nil {
		return nil, err
	}

	// Load user
	u := &sdk.User{}
	if err := json.Unmarshal(jsonUser, u); err != nil {
		return nil, err
	}

	u.Admin = admin
	u.ID = id
	u.Origin = origin
	return u, nil
}

// LoadUserAndAuth Load user with auth information
func LoadUserAndAuth(db gorp.SqlExecutor, name string) (*sdk.User, error) {
	query := `SELECT id, admin, data, auth, origin FROM "user" WHERE username = $1`

	var jsonUser []byte
	var jsonAuth []byte
	var id int64
	var admin bool
	var origin string

	if err := db.QueryRow(query, name).Scan(&id, &admin, &jsonUser, &jsonAuth, &origin); err != nil {
		return nil, err
	}

	// Load user
	u := &sdk.User{}
	if err := json.Unmarshal(jsonUser, u); err != nil {
		return nil, err
	}

	// Load Auth
	a := &sdk.Auth{}
	if err := json.Unmarshal(jsonAuth, a); err != nil {
		return nil, err
	}

	u.Admin = admin
	u.Auth = *a
	u.ID = id
	u.Origin = origin
	return u, nil
}

// FindUserIDByName retrieves only user ID in database
func FindUserIDByName(db gorp.SqlExecutor, name string) (int64, error) {
	query := `SELECT id FROM "user" WHERE username = $1`

	var id int64
	if err := db.QueryRow(query, name).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// LoadUsers load all users from database
func LoadUsers(db gorp.SqlExecutor) ([]*sdk.User, error) {
	users := []*sdk.User{}

	query := `SELECT "user".username, "user".data, origin, admin FROM "user" ORDER BY "user".username`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var username, data, origin string
		var adminSQL sql.NullBool
		err := rows.Scan(&username, &data, &origin, &adminSQL)
		if err != nil {
			return nil, err
		}

		var admin bool
		if adminSQL.Valid {
			admin = adminSQL.Bool
		}

		uTemp := &sdk.User{}
		if err := json.Unmarshal([]byte(data), uTemp); err != nil {
			log.Warning("LoadUsers> Unable to load user %s : %s", username, err)
			return nil, err
		}

		u := &sdk.User{
			Username: username,
			Fullname: uTemp.Fullname,
			Origin:   origin,
			Email:    uTemp.Email,
			Admin:    admin,
		}

		users = append(users, u)
	}
	return users, nil
}

// CountUser Count user table
func CountUser(db gorp.SqlExecutor) (int64, error) {
	query := `SELECT count(id) FROM "user" WHERE 1 = 1`

	var countResult int64
	err := db.QueryRow(query).Scan(&countResult)
	if err != nil {
		return 1, err
	}
	return countResult, nil
}

// UpdateUser update given user
func UpdateUser(db gorp.SqlExecutor, u sdk.User) error {
	u.Groups = nil
	su, err := json.Marshal(u)
	if err != nil {
		return err
	}
	query := `UPDATE "user" SET username=$1, admin=$2, data=$3 WHERE id=$4`
	_, err = db.Exec(query, u.Username, u.Admin, su, u.ID)
	return err
}

// UpdateUserAndAuth update given user
func UpdateUserAndAuth(db gorp.SqlExecutor, u sdk.User) error {
	u.Groups = nil
	su, err := json.Marshal(u)
	if err != nil {
		return err
	}
	sa, err := json.Marshal(u.Auth)
	if err != nil {
		return err
	}
	query := `UPDATE "user" SET username=$1, admin=$2, data=$3, auth=$4 WHERE id=$5`
	_, err = db.Exec(query, u.Username, u.Admin, su, sa, u.ID)
	return err
}

// DeleteUserWithDependenciesByName Delete user and all his dependencies
func DeleteUserWithDependenciesByName(db gorp.SqlExecutor, s string) error {
	u, err := LoadUserWithoutAuth(db, s)
	if err != nil {
		return err
	}
	return DeleteUserWithDependencies(db, u)
}

// DeleteUserWithDependencies Delete user and all his dependencies
func DeleteUserWithDependencies(db gorp.SqlExecutor, u *sdk.User) error {
	if err := deleteUserFromUserGroup(db, u); err != nil {
		return sdk.WrapError(err, "DeleteUserWithDependencies>User cannot be removed from group_user table")
	}

	if err := deleteUser(db, u); err != nil {
		return sdk.WrapError(err, "DeleteUserWithDependencies> User cannot be removed from user table")
	}
	return nil
}

func deleteUserFromUserGroup(db gorp.SqlExecutor, u *sdk.User) error {
	query := `DELETE FROM "group_user" WHERE user_id=$1`
	_, err := db.Exec(query, u.ID)
	return err
}

func deleteUser(db gorp.SqlExecutor, u *sdk.User) error {
	query := `DELETE FROM "user" WHERE id=$1`
	_, err := db.Exec(query, u.ID)
	return err
}

// InsertUser Insert new user
func InsertUser(db gorp.SqlExecutor, u *sdk.User, a *sdk.Auth) error {
	su, err := json.Marshal(u)
	if err != nil {
		return err
	}
	sa, err := json.Marshal(a)
	if err != nil {
		return err
	}
	query := `INSERT INTO "user" (username, admin, data, auth, created, origin) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`
	return db.QueryRow(query, u.Username, u.Admin, su, sa, time.Now(), u.Origin).Scan(&u.ID)
}

// NewPersistentSession creates a new persistent session token in database
func NewPersistentSession(db gorp.SqlExecutor, u *sdk.User) (sessionstore.SessionKey, error) {
	t, errSession := sessionstore.NewSessionKey()
	if errSession != nil {
		return "", errSession
	}
	newToken := sdk.UserToken{
		Token:              string(t),
		Comment:            fmt.Sprintf("New persistent session for %s", u.Username),
		CreationDate:       time.Now(),
		LastConnectionDate: time.Now(),
		UserID:             u.ID,
	}

	if err := InsertPersistentSessionToken(db, newToken); err != nil {
		return "", err
	}
	return t, nil
}

// LoadPersistentSessionToken load a token from the database
func LoadPersistentSessionToken(db gorp.SqlExecutor, k sessionstore.SessionKey) (*sdk.UserToken, error) {
	tdb := persistentSessionToken{}
	if err := db.SelectOne(&tdb, "select * from user_persistent_session where token = $1", string(k)); err != nil {
		return nil, err
	}
	t := sdk.UserToken(tdb)
	return &t, nil
}

// InsertPersistentSessionToken create a new persistent session
func InsertPersistentSessionToken(db gorp.SqlExecutor, t sdk.UserToken) error {
	tdb := persistentSessionToken(t)
	if err := db.Insert(&tdb); err != nil {
		return sdk.WrapError(err, "InsertPersistentSessionToken> Unable to insert persistent session token for user %d", t.UserID)
	}
	return nil
}

// UpdatePersistentSessionToken updates a persistent session
func UpdatePersistentSessionToken(db gorp.SqlExecutor, t sdk.UserToken) error {
	tdb := persistentSessionToken(t)
	if _, err := db.Update(&tdb); err != nil {
		return sdk.WrapError(err, "UpdatePersistentSessionToken> Unable to update persistent session token for user %d", t.UserID)
	}
	return nil
}

// DeletePersistentSessionToken deletes a persistent session
func DeletePersistentSessionToken(db gorp.SqlExecutor, t sdk.UserToken) error {
	tdb := persistentSessionToken(t)
	if _, err := db.Delete(&tdb); err != nil {
		return sdk.WrapError(err, "DeletePersistentSessionToken> Unable to delete persistent session token for user %d", t.UserID)
	}
	return nil
}
