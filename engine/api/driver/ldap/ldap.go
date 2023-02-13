package ldap

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/ovh/cds/sdk"
	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"gopkg.in/ldap.v2"
	"strings"
	"text/template"
)

const errUserNotFound = "ldap::user not found"

type ldapDriver struct {
	conf Config
	conn *ldap.Conn
}

// Config handles all config to connect to the LDAP.
type Config struct {
	Host            string // 192.168.1.32
	Port            int    // 636
	SSL             bool   // true
	RootDN          string // dc=ejnserver,dc=fr
	UserSearchBase  string // ou=people
	UserSearch      string // uid={{.search}}
	UserFullname    string // {{.givenName}} {{.sn}}
	ManagerDN       string // cn=admin,dc=ejnserver,dc=fr
	ManagerPassword string // SECRET_PASSWORD_MANAGER
}

// NewLdapDriver returns a new ldap auth driver.
func NewLdapDriver(ctx context.Context, cfg Config) (sdk.Driver, error) {
	var d = &ldapDriver{
		conf: cfg,
	}

	if err := d.openLDAP(ctx, cfg); err != nil {
		return nil, fmt.Errorf("unable to open LDAP connection to %s:%d : %v", cfg.Host, cfg.Port, err)
	}
	return d, nil
}

func (l *ldapDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	if bind, ok := req["bind"]; !ok || bind == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid bind term for ldap signin")
	}
	if password, ok := req["password"]; !ok || password == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid password for ldap signin")
	}
	return nil
}

func (l *ldapDriver) GetUserInfoFromDriver(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var userInfo sdk.AuthDriverUserInfo
	var bind = req.String("bind")
	var password = req.String("password")

	if err := l.bind(ctx, bind, password); err != nil {
		return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	entry, err := l.search(ctx, bind, "uid", "dn", "cn", "ou", "givenName", "sn", "mail", "memberOf", "company")
	if err != nil && err.Error() != errUserNotFound {
		return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	if len(entry) > 1 {
		return userInfo, fmt.Errorf("LDAP Search error multiple values")
	}

	//If user doesn't exist and search was'nt successful => exist
	if err != nil {
		log.Warn(ctx, "LDAP> Search error %s", err)
		return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	//Execute template to compute fullname
	tmpl, err := template.New("userfullname").Parse(l.conf.UserFullname)
	if err != nil {
		log.Error(ctx, "LDAP> Error with user fullname template %s : %s", l.conf.UserFullname, err)
		tmpl, _ = template.New("userfullname").Parse("{{.givenName}}")
	}
	bufFullname := new(bytes.Buffer)
	if err := tmpl.Execute(bufFullname, entry[0].Attributes); err != nil {
		return userInfo, sdk.WithStack(err)
	}

	userInfo.Fullname = bufFullname.String()
	userInfo.Email = entry[0].Attributes["mail"]
	userInfo.ExternalID = entry[0].Attributes["uid"]
	userInfo.Username = req.String("bind")
	userInfo.Organization = req.String("company")

	return userInfo, nil
}

func (l *ldapDriver) openLDAP(ctx context.Context, conf Config) error {
	if l.conn != nil {
		l.conn.Close()
	}
	var err error
	l.conf = conf

	address := fmt.Sprintf("%s:%d", l.conf.Host, l.conf.Port)

	log.Info(ctx, "Auth> Preparing connection to LDAP server: %s", address)
	if !l.conf.SSL {
		l.conn, err = ldap.Dial("tcp", address)
		if err != nil {
			log.Error(ctx, "Auth> Cannot dial %s : %s", address, err)
			return sdk.WithStack(sdk.ErrLDAPConn)
		}

		//Reconnect with TLS
		err = l.conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			log.Error(ctx, "Auth> Cannot start TLS %s : %s", address, err)
			return sdk.WithStack(sdk.ErrLDAPConn)
		}
	} else {
		log.Info(ctx, "Auth> Connecting to LDAP server")
		l.conn, err = ldap.DialTLS("tcp", address, &tls.Config{
			ServerName:         l.conf.Host,
			InsecureSkipVerify: false,
		})
		if err != nil {
			log.Error(ctx, "Auth> Cannot dial TLS (InsecureSkipVerify=false) %s : %s", address, err)
			return sdk.WithStack(sdk.ErrLDAPConn)
		}
	}

	if l.conf.ManagerDN != "" {
		log.Info(ctx, "LDAP> bind manager %s", l.conf.ManagerDN)
		if err := l.conn.Bind(l.conf.ManagerDN, l.conf.ManagerPassword); err != nil {
			if shoudRetry(ctx, err) {
				if err := l.openLDAP(ctx, l.conf); err != nil {
					return err
				}
				if err := l.conn.Bind(l.conf.ManagerDN, l.conf.ManagerPassword); err != nil {
					return sdk.WithStack(err)
				}
			} else {
				return err
			}
		}
	}

	return nil
}

// bind binds
func (l *ldapDriver) bind(ctx context.Context, term, password string) error {
	bindRequest := strings.Replace(l.conf.UserSearch, "{0}", ldap.EscapeFilter(term), 1) + "," + l.conf.UserSearchBase + "," + l.conf.RootDN
	log.Debug(ctx, "LDAP> bind user %s", bindRequest)

	if err := l.conn.Bind(bindRequest, password); err != nil {
		if !shoudRetry(ctx, err) {
			return sdk.WithStack(err)
		}
		if err := l.openLDAP(ctx, l.conf); err != nil {
			return err
		}
		if err := l.conn.Bind(bindRequest, password); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}

// Search search
func (l *ldapDriver) search(ctx context.Context, term string, attributes ...string) ([]Entry, error) {
	userSearch := strings.Replace(l.conf.UserSearch, "{0}", ldap.EscapeFilter(term), 1)
	filter := fmt.Sprintf("(%s)", userSearch)

	log.Debug(ctx, "LDAP> Search user %s", filter)
	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		l.conf.RootDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1,
		0,
		false,
		filter,
		attributes,
		nil,
	)

	sr, err := l.conn.Search(searchRequest)
	if err != nil {
		if !shoudRetry(ctx, err) {
			return nil, err
		}
		if err := l.openLDAP(ctx, l.conf); err != nil {
			return nil, err
		}
		sr, err = l.conn.Search(searchRequest)
		if err != nil {
			return nil, sdk.WithStack(err)
		}
	}

	if len(sr.Entries) < 1 {
		return nil, errors.New(errUserNotFound)
	}

	entries := []Entry{}
	for _, e := range sr.Entries {
		entry := Entry{
			DN:         e.DN,
			Attributes: make(map[string]string),
		}

		for _, a := range attributes {
			entry.Attributes[a] = e.GetAttributeValue(a)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
