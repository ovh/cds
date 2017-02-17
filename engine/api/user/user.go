package user

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Verify verify user token
func Verify(u *sdk.User, token string) (string, string, error) {
	if u.Auth.EmailVerified && u.Auth.DateReset == 0 {
		return "", "", errors.New("Account already verified")
	}

	if u.Auth.DateReset != 0 && time.Since(time.Unix(u.Auth.DateReset, 0)).Minutes() > 30 {
		return "", "", fmt.Errorf("Reset operation expired")
	}

	err := checkToken(u, token)
	if err != nil {
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

// LoadUserWithoutAuthByID load user information without secret
func LoadUserWithoutAuthByID(db gorp.SqlExecutor, userID int64) (*sdk.User, error) {
	query := `SELECT username, admin, data, origin FROM "user" WHERE id = $1`

	var jsonUser []byte
	var username, origin string
	var admin bool
	err := db.QueryRow(query, userID).Scan(&username, &admin, &jsonUser, &origin)
	if err != nil {
		return nil, err
	}

	// Load user
	u, err := sdk.NewUser(username).FromJSON(jsonUser)
	if err != nil {
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
	err := db.QueryRow(query, name).Scan(&id, &admin, &jsonUser, &origin)
	if err != nil {
		return nil, err
	}

	// Load user
	u, err := sdk.NewUser(name).FromJSON(jsonUser)
	if err != nil {
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
	err := db.QueryRow(query, name).Scan(&id, &admin, &jsonUser, &jsonAuth, &origin)
	if err != nil {
		return nil, err
	}

	// Load user
	u, err := sdk.NewUser(name).FromJSON(jsonUser)
	if err != nil {
		return nil, err
	}

	// Load Auth
	a, err := sdk.NewAuth("").FromJSON(jsonAuth)
	if err != nil {
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
	err := db.QueryRow(query, name).Scan(&id)
	if err != nil {
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
	query := `UPDATE "user" SET username=$1, admin=$2, data=$3 WHERE id=$4`
	u.Groups = nil
	_, err := db.Exec(query, u.Username, u.Admin, u.JSON(), u.ID)
	return err
}

// UpdateUserAndAuth update given user
func UpdateUserAndAuth(db gorp.SqlExecutor, u sdk.User) error {
	query := `UPDATE "user" SET username=$1, admin=$2, data=$3, auth=$4 WHERE id=$5`
	u.Groups = nil
	_, err := db.Exec(query, u.Username, u.Admin, u.JSON(), u.Auth.JSON(), u.ID)
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

	err := deleteUserFromUserGroup(db, u)
	if err != nil {
		log.Warning("DeleteUserWithDependencies>User cannot be removed from group_user table: %s", err)
		return err
	}

	err = deleteUser(db, u)
	if err != nil {
		log.Warning("DeleteUserWithDependencies> User cannot be removed from user table: %s", err)
		return err
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
	query := `INSERT INTO "user" (username, admin, data, auth, created, origin) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`
	err := db.QueryRow(query, u.Username, u.Admin, u.JSON(), a.JSON(), time.Now(), u.Origin).Scan(&u.ID)
	return err
}
