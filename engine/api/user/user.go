package user

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

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

// deprecatedLoadUserWithoutAuthByID load user information without secret
func deprecatedLoadUserWithoutAuthByID(db gorp.SqlExecutor, userID int64) (*sdk.User, error) {
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

// CountUser Count user table
func CountUser(db gorp.SqlExecutor) (int64, error) {
	return db.SelectInt("SELECT count(id) FROM authentified_user")
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
func insertUser(db gorp.SqlExecutor, u *sdk.User, a *sdk.Auth) error {
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
