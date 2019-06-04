package ldapauthentication

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"gopkg.in/ldap.v2"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var _ authentication.Driver = new(ldapAuthentication)

type ldapAuthentication struct {
	conf LDAPConfig
	conn *ldap.Conn
}

//LDAPConfig handles all config to connect to the LDAP
type LDAPConfig struct {
	Host         string
	Port         int
	Base         string
	DN           string
	SSL          bool
	UserFullname string
	BindDN       string
	BindPwd      string
}

func New(cfg LDAPConfig) authentication.Driver {
	return &ldapAuthentication{
		conf: cfg,
	}
}

func (l *ldapAuthentication) CheckAuthentication(ctx context.Context, db gorp.SqlExecutor, r *http.Request) (*sdk.AuthentifiedUser, error) {
	_, end := observability.Span(ctx, "ldapAuthentication.CheckAuthentication")
	defer end()

	vars := mux.Vars(r)
	username := vars["username"]
	password := vars["password"]

	//Bind user
	if err := l.Bind(username, password); err != nil {
		return nil, sdk.WrapError(sdk.ErrUnauthorized, "LDAP bind failure: %v", err)
	}

	log.Debug("LDAP> Bind successful %s", username)

	// Search user
	search := fmt.Sprintf("(uid=%s)", username)
	entry, errSearch := l.Search(search, "uid", "cn", "ou", "givenName", "sn", "mail")
	if errSearch != nil && errSearch != errUserNotFound {
		log.Warning("LDAP> Search error %s: %s", search, errSearch)
		return nil, sdk.WrapError(sdk.ErrUnauthorized, "LDAP bind failure: %v", errSearch)
	}

	if len(entry) > 1 {
		log.Error("LDAP> Search error %s: multiple values", search)
		return nil, fmt.Errorf("LDAP Search error %s: multiple values", search)
	}

	u, err := user.LoadUserByUsername(ctx, db, username)
	if err != nil {
		return nil, err
	}

	//Execute template to compute fullname
	//tmpl, err := template.New("userfullname").Parse(l.conf.UserFullname)
	//if err != nil {
	//	log.Error("LDAP> Error with user fullname template %s : %s", l.conf.UserFullname, err)
	//	tmpl, _ = template.New("userfullname").Parse("{{.givenName}}")
	//}
	//bufFullname := new(bytes.Buffer)
	//tmpl.Execute(bufFullname, entry[0].Attributes)
	//
	//u.Fullname = bufFullname.String()
	//u.Email = entry[0].Attributes["mail"]

	return u, nil
}

func (l *ldapAuthentication) openLDAP() error {
	address := fmt.Sprintf("%s:%d", l.conf.Host, l.conf.Port)

	log.Info("Auth> Preparing connection to LDAP server: %s", address)
	if !l.conf.SSL {
		var err error
		l.conn, err = ldap.Dial("tcp", address)
		if err != nil {
			log.Error("Auth> Cannot dial %s : %s", address, err)
			return sdk.ErrLDAPConn
		}

		// Reconnect with TLS
		err = l.conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			log.Error("Auth> Cannot start TLS %s : %s", address, err)
			return sdk.ErrLDAPConn
		}
	} else {
		var err error
		log.Info("Auth> Connecting to LDAP server")
		l.conn, err = ldap.DialTLS("tcp", address, &tls.Config{
			ServerName:         l.conf.Host,
			InsecureSkipVerify: false,
		})
		if err != nil {
			log.Error("Auth> Cannot dial TLS (InsecureSkipVerify=false) %s : %s", address, err)
			return sdk.ErrLDAPConn
		}
	}

	if l.conf.BindDN != "" {
		log.Info("LDAP> Bind user %s", l.conf.BindDN)
		if err := l.conn.Bind(l.conf.BindDN, l.conf.BindPwd); err != nil {
			if shoudRetry(err) {
				if err := l.openLDAP(); err != nil {
					return err
				}
				if err := l.conn.Bind(l.conf.BindDN, l.conf.BindPwd); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}

//Bind binds
func (l *ldapAuthentication) Bind(username, password string) error {
	bindRequest := fmt.Sprintf(l.conf.DN, username)
	bindRequest = strings.Replace(bindRequest, "{{.ldapBase}}", l.conf.Base, -1)
	log.Debug("LDAP> Bind user %s", bindRequest)

	if err := l.conn.Bind(bindRequest, password); err != nil {
		if shoudRetry(err) {
			err = l.openLDAP()
			if err != nil {
				return err
			}
			err = l.conn.Bind(bindRequest, password)
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
func (l *ldapAuthentication) Search(filter string, attributes ...string) ([]Entry, error) {
	attr := append(attributes, "dn")

	if l.conf.BindDN != "" {
		log.Debug("LDAP> Bind user %s", l.conf.BindDN)
		if err := l.conn.Bind(l.conf.BindDN, l.conf.BindPwd); err != nil {
			if shoudRetry(err) {
				if err := l.openLDAP(); err != nil {
					return nil, err
				}
				if err := l.conn.Bind(l.conf.BindDN, l.conf.BindPwd); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		l.conf.Base,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		attr,
		nil,
	)

	sr, err := l.conn.Search(searchRequest)
	if err != nil {
		if shoudRetry(err) {
			err = l.openLDAP()
			if err != nil {
				return nil, err
			}
			sr, err = l.conn.Search(searchRequest)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if len(sr.Entries) < 1 {
		return nil, errUserNotFound
	}

	entries := []Entry{}
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

	return entries, nil
}
