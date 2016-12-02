package user

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
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
func LoadUserWithoutAuthByID(db *sql.DB, userID int64) (*sdk.User, error) {
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
func LoadUserWithoutAuth(db database.Querier, name string) (*sdk.User, error) {
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
func LoadUserAndAuth(db *sql.DB, name string) (*sdk.User, error) {
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
func FindUserIDByName(db *sql.DB, name string) (int64, error) {
	query := `SELECT id FROM "user" WHERE username = $1`

	var id int64
	err := db.QueryRow(query, name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// LoadUsers load all users from database
func LoadUsers(db *sql.DB) ([]*sdk.User, error) {
	users := []*sdk.User{}

	query := `SELECT "user".username, "user".data, origin FROM "user" WHERE 1 = 1 ORDER BY "user".username`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var username, data, origin string
		err := rows.Scan(&username, &data, &origin)
		if err != nil {
			return nil, err
		}

		uTemp, err := sdk.NewUser(username).FromJSON([]byte(data))
		if err != nil {
			return nil, err
		}

		u := &sdk.User{
			Username: username,
			Fullname: uTemp.Fullname,
			Origin:   origin,
		}

		users = append(users, u)
	}
	return users, nil
}

// CountUser Count user table
func CountUser(db *sql.DB) (int64, error) {
	query := `SELECT count(id) FROM "user" WHERE 1 = 1`

	var countResult int64
	err := db.QueryRow(query).Scan(&countResult)
	if err != nil {
		return 1, err
	}
	return countResult, nil
}

// UpdateUser update given user
func UpdateUser(db *sql.DB, u sdk.User) error {
	query := `UPDATE "user" SET username=$1, admin=$2, data=$3 WHERE id=$4`
	u.Groups = nil
	_, err := db.Exec(query, u.Username, u.Admin, u.JSON(), u.ID)
	return err
}

// UpdateUserAndAuth update given user
func UpdateUserAndAuth(db *sql.DB, u sdk.User) error {
	query := `UPDATE "user" SET username=$1, admin=$2, data=$3, auth=$4 WHERE id=$5`
	u.Groups = nil
	_, err := db.Exec(query, u.Username, u.Admin, u.JSON(), u.Auth.JSON(), u.ID)
	return err
}

// DeleteUserWithDependencies Delete user and all his dependencies
func DeleteUserWithDependencies(db database.Executer, u *sdk.User) error {

	err := deleteUserFromUserGroup(db, u)
	if err != nil {
		log.Warning("DeleteUserWithDependencies>User cannot be removed from group_user table: %s", err)
		return err
	}

	err = deleteUserKey(db, u)
	if err != nil {
		log.Warning("DeleteUserWithDependencies>Cannot remove user key: %s", err)
		return err
	}

	err = deleteUser(db, u)
	if err != nil {
		log.Warning("DeleteUserWithDependencies> User cannot be removed from user table: %s", err)
		return err
	}
	return nil
}

func deleteUserKey(db database.Executer, u *sdk.User) error {
	query := `DELETE FROM "user_key" WHERE user_id=$1`
	_, err := db.Exec(query, u.ID)
	return err
}

func deleteUserFromUserGroup(db database.Executer, u *sdk.User) error {
	query := `DELETE FROM "group_user" WHERE user_id=$1`
	_, err := db.Exec(query, u.ID)
	return err
}

func deleteUser(db database.Executer, u *sdk.User) error {
	query := `DELETE FROM "user" WHERE id=$1`
	_, err := db.Exec(query, u.ID)
	return err
}

// InsertUser Insert new user
func InsertUser(db database.QueryExecuter, u *sdk.User, a *sdk.Auth) error {
	query := `INSERT INTO "user" (username, admin, data, auth, created, origin) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`
	err := db.QueryRow(query, u.Username, u.Admin, u.JSON(), a.JSON(), time.Now(), u.Origin).Scan(&u.ID)
	return err
}

// LoadUserPermissions retrieves all group memberships
func LoadUserPermissions(db *sql.DB, user *sdk.User) error {
	user.Groups = nil
	query := `SELECT "group".id, "group".name, "group_user".group_admin FROM "group"
	 		  JOIN group_user ON group_user.group_id = "group".id
	 		  WHERE group_user.user_id = $1 ORDER BY "group".name ASC`

	rows, err := db.Query(query, user.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var admin bool
		err = rows.Scan(&group.ID, &group.Name, &admin)
		if err != nil {
			return err
		}

		err = project.LoadProjectByGroup(db, &group)
		if err != nil {
			return err
		}

		err = pipeline.LoadPipelineByGroup(db, &group)
		if err != nil {
			return err
		}

		err = application.LoadApplicationByGroup(db, &group)
		if err != nil {
			return err
		}

		err = environment.LoadEnvironmentByGroup(db, &group)
		if err != nil {
			return err
		}

		if admin {
			group.Admins = append(group.Admins, *user)
		}

		user.Groups = append(user.Groups, group)
	}
	return nil
}

// LoadGroupPermissions retrieves all group memberships
func LoadGroupPermissions(db *sql.DB, groupID int64) (*sdk.Group, error) {
	query := `SELECT "group".name FROM "group" WHERE "group".id = $1`

	group := &sdk.Group{ID: groupID}
	err := db.QueryRow(query, groupID).Scan(&group.Name)
	if err != nil {
		return nil, fmt.Errorf("no group with id %d: %s", groupID, err)
	}

	err = project.LoadProjectByGroup(db, group)
	if err != nil {
		return nil, err
	}

	err = pipeline.LoadPipelineByGroup(db, group)
	if err != nil {
		return nil, err
	}

	err = application.LoadApplicationByGroup(db, group)
	if err != nil {
		return nil, err
	}

	err = environment.LoadEnvironmentByGroup(db, group)
	if err != nil {
		return nil, err
	}

	return group, nil
}
