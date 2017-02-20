package auth

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-gorp/gorp"
	"gopkg.in/ldap.v2"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

const errUserNotFound = "user not found"

//LDAPConfig handles all config to connect to the LDAP
type LDAPConfig struct {
	Host         string
	Port         int
	Base         string
	DN           string
	SSL          bool
	UserFullname string
}

//LDAPDriver is the LDAP client interface
type LDAPDriver interface {
	Open(c LDAPConfig) error
	Close() error
	Search(filter string, attributes ...string) ([]Entry, error)
}

//LDAPClient enbeddes the LDAP connecion
type LDAPClient struct {
	store sessionstore.Store
	conn  *ldap.Conn
	conf  LDAPConfig
	local *LocalClient
}

//Entry represents a LDAP entity
type Entry struct {
	DN         string
	Attributes map[string]string
}

//Open open a true LDAP connection
func (c *LDAPClient) Open(options interface{}, store sessionstore.Store) error {
	log.Notice("Auth> Connecting to session store")
	c.store = store
	//LDAP Client needs a local client to check local users
	c.local = &LocalClient{}
	c.local.Open(options, store)
	return c.openLDAP(options)
}

func (c *LDAPClient) openLDAP(options interface{}) error {
	conf, ok := options.(LDAPConfig)
	if !ok {
		return sdk.ErrLDAPConn
	}
	var err error
	c.conf = conf

	address := fmt.Sprintf("%s:%d", c.conf.Host, c.conf.Port)

	log.Notice("Auth> Preparing connection to LDAP server: %s", address)
	if !c.conf.SSL {
		c.conn, err = ldap.Dial("tcp", address)
		if err != nil {
			log.Critical("Auth> Cannot dial %s : %s", address, err)
			return sdk.ErrLDAPConn
		}

		// Reconnect with TLS
		err = c.conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			log.Critical("Auth> Cannot start TLS %s : %s", address, err)
			return sdk.ErrLDAPConn
		}
	} else {
		log.Notice("Auth> Connecting to LDAP server")
		c.conn, err = ldap.DialTLS("tcp", address, &tls.Config{
			ServerName:         c.conf.Host,
			InsecureSkipVerify: false,
		})
		if err != nil {
			log.Critical("Auth> Cannot dial TLS (InsecureSkipVerify=false) %s : %s", address, err)
			return sdk.ErrLDAPConn
		}
	}
	return nil
}

func shoudRetry(err error) bool {
	if err == nil {
		return false
	}
	ldapErr, ok := err.(*ldap.Error)
	if !ok {
		return false
	}
	if ldapErr.ResultCode == ldap.ErrorNetwork {
		log.Notice("LDAP> Retry")
		return true
	}
	return false
}

//isCredentialError check if err is LDAPResultInvalidCredentials
func isCredentialError(err error) bool {
	ldapErr, b := err.(*ldap.Error)
	if !b {
		return false
	}
	if ldapErr.ResultCode == ldap.LDAPResultInvalidCredentials {
		return true
	}
	return false
}

//Close the specified client
func (c *LDAPClient) Close() {
	c.conn.Close()
}

//Store returns store
func (c *LDAPClient) Store() sessionstore.Store {
	return c.store
}

//Bind binds
func (c *LDAPClient) Bind(username, password string) error {
	bindRequest := fmt.Sprintf(c.conf.DN, username)
	bindRequest = strings.Replace(bindRequest, "{{.ldap-base}}", c.conf.Base, -1)
	log.Debug("LDAP> Bind user %s", bindRequest)

	if err := c.conn.Bind(bindRequest, password); err != nil {
		if shoudRetry(err) {
			err = c.openLDAP(c.conf)
			if err != nil {
				return err
			}
			err = c.conn.Bind(bindRequest, password)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

//Search search
func (c *LDAPClient) Search(filter string, attributes ...string) ([]Entry, error) {
	entries := []Entry{}
	key := cache.Key("ldap", filter)
	cache.Get(key, &entries)

	if len(entries) == 0 {
		attr := append(attributes, "dn")
		// Search for the given username
		searchRequest := ldap.NewSearchRequest(
			c.conf.Base,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			filter,
			attr,
			nil,
		)

		sr, err := c.conn.Search(searchRequest)
		if err != nil {
			if shoudRetry(err) {
				err = c.openLDAP(c.conf)
				if err != nil {
					return nil, err
				}
				sr, err = c.conn.Search(searchRequest)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		if len(sr.Entries) < 1 {
			return nil, errors.New(errUserNotFound)
		}

		for _, e := range sr.Entries {
			entry := Entry{
				DN:         e.DN,
				Attributes: make(map[string]string),
			}
			for _, a := range attr {
				entry.Attributes[a] = e.GetAttributeValue(a)
			}
			entries = append(entries, entry)
		}
		//Put ldap entries in cache for 5 minutes to avoid LDAP flood
		cache.SetWithTTL(key, entries, 300)
	}

	return entries, nil
}

func (c *LDAPClient) searchAndInsertOrUpdateUser(db gorp.SqlExecutor, username string) (*sdk.User, error) {
	// Search user
	search := fmt.Sprintf("(uid=%s)", username)
	entry, errSearch := c.Search(search, "uid", "cn", "ou", "givenName", "sn", "mail")
	if errSearch != nil && errSearch.Error() != errUserNotFound {
		log.Warning("LDAP> Search error %s: %s", search, errSearch)
		return nil, errSearch
	}

	if len(entry) > 1 {
		log.Critical("LDAP> Search error %s: multiple values", search)
		return nil, fmt.Errorf("LDAP Search error %s: multiple values", search)
	}

	u, err := user.LoadUserAndAuth(db, username)

	//If user exists in database and is set as "local"
	if u != nil && u.Origin == "local" {
		return u, nil
	}

	//If user doesn't exist and search was'nt successfull => exist
	if errSearch != nil {
		log.Warning("LDAP> Search error %s: %s", search, errSearch)
		return nil, errSearch
	}

	//User
	var newUser = false
	if err == sql.ErrNoRows {
		newUser = true
		u = &sdk.User{
			Admin:    false,
			Username: username,
			Origin:   "ldap",
		}
	} else if err != nil {
		log.Warning("Auth> User %s not found : %s", username, err)
		return nil, err
	}

	//Execute template to compute fullname
	tmpl, err := template.New("userfullname").Parse(c.conf.UserFullname)
	if err != nil {
		log.Critical("LDAP> Error with user fullname template %s : %s", c.conf.UserFullname, err)
		tmpl, _ = template.New("userfullname").Parse("{{.givenName}}")
	}
	bufFullname := new(bytes.Buffer)
	tmpl.Execute(bufFullname, entry[0].Attributes)

	u.Fullname = bufFullname.String()
	u.Email = entry[0].Attributes["mail"]

	if newUser {
		a := &sdk.Auth{
			EmailVerified: true,
		}
		if err := user.InsertUser(db, u, a); err != nil {
			log.Critical("LDAP> Error inserting user %s: %s", username, err)
			return nil, err
		}
		u.Auth = *a
	} else {
		if err := user.UpdateUser(db, *u); err != nil {
			log.Critical("LDAP> Unable to update user %s : %s", username, err)
			return nil, err
		}
	}
	return u, nil
}

//Authentify check username and password
func (c *LDAPClient) Authentify(db gorp.SqlExecutor, username, password string) (bool, error) {
	//Bind user
	if err := c.Bind(username, password); err != nil {
		log.Warning("LDAP> Bind error %s %s", username, err)

		if !isCredentialError(err) {
			return false, err
		}
		//Try local auth
		return c.local.Authentify(db, username, password)
	}

	log.Debug("LDAP> Bind sucessfull %s", username)

	//Search user, refresh data and update database
	if _, err := c.searchAndInsertOrUpdateUser(db, username); err != nil {
		return false, err
	}

	return true, nil
}

//AuthentifyUser check password in database
func (c *LDAPClient) AuthentifyUser(db gorp.SqlExecutor, u *sdk.User, password string) (bool, error) {
	return c.Authentify(db, u.Username, password)
}

//GetCheckAuthHeaderFunc returns the func to heck http headers.
//Options is a const to switch from session to basic auth or both
func (c *LDAPClient) GetCheckAuthHeaderFunc(options interface{}) func(db *gorp.DbMap, headers http.Header, ctx *context.Ctx) error {
	return func(db *gorp.DbMap, headers http.Header, ctx *context.Ctx) error {
		//Check if its a worker
		if h := headers.Get(sdk.AuthHeader); h != "" {
			if err := checkWorkerAuth(db, h, ctx); err != nil {
				return err
			}
			return nil
		}
		//Check if its comming from CLI
		if headers.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
			if getUserPersistentSession(db, c.Store(), headers, ctx) {
				return nil
			}
			if reloadUserPersistentSession(db, c.Store(), headers, ctx) {
				return nil
			}
		}

		return c.checkUserSessionAuth(db, headers, ctx)
	}
}

func (c *LDAPClient) checkUserSessionAuth(db *gorp.DbMap, headers http.Header, ctx *context.Ctx) error {
	sessionToken := headers.Get(sdk.SessionTokenHeader)
	if sessionToken == "" {
		return fmt.Errorf("no session header")
	}
	exists, err := c.store.Exists(sessionstore.SessionKey(sessionToken))
	if err != nil {
		return err
	}
	username, err := GetUsername(c.store, sessionToken)
	if err != nil {
		return err
	}
	u, err := c.searchAndInsertOrUpdateUser(db, username)
	if err != nil {
		return err
	}
	ctx.User = u

	if !exists {
		return fmt.Errorf("invalid session")
	}

	return nil
}
