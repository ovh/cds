package ldap

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	ldap "gopkg.in/ldap.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var _ sdk.AuthDriver = new(AuthDriver)

const errUserNotFound = "ldap::user not found"

type AuthDriver struct {
	signupDisabled bool
	conf           Config
	conn           *ldap.Conn
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

// NewDriver returns a new ldap auth driver.
func NewDriver(ctx context.Context, signupDisabled bool, cfg Config) (sdk.AuthDriver, error) {
	var d = AuthDriver{
		signupDisabled: signupDisabled,
		conf:           cfg,
	}

	if err := d.openLDAP(ctx, cfg); err != nil {
		return nil, fmt.Errorf("unable to open LDAP connection to %s:%d : %v", cfg.Host, cfg.Port, err)
	}

	return d, nil
}

func (d AuthDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerLDAP,
		SignupDisabled: d.signupDisabled,
	}
}

func (d AuthDriver) GetSessionDuration(_ sdk.AuthDriverUserInfo, _ sdk.AuthConsumer) time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

func (d AuthDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	if bind, ok := req["bind"]; !ok || bind == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid bind term for ldap signin")
	}
	if password, ok := req["password"]; !ok || password == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid password for ldap signin")
	}
	return nil
}

func (d AuthDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var userInfo sdk.AuthDriverUserInfo
	var bind = req["bind"]
	var password = req["password"]

	if err := d.bind(ctx, bind, password); err != nil {
		return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	entry, err := d.search(ctx, bind, "uid", "dn", "cn", "ou", "givenName", "sn", "mail", "memberOf")
	if err != nil && err.Error() != errUserNotFound {
		return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	if len(entry) > 1 {
		return userInfo, fmt.Errorf("LDAP Search error multiple values")
	}

	//If user doesn't exist and search was'nt successful => exist
	if err != nil {
		log.Warning(ctx, "LDAP> Search error %s", err)
		return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	//Execute template to compute fullname
	tmpl, err := template.New("userfullname").Parse(d.conf.UserFullname)
	if err != nil {
		log.Error(ctx, "LDAP> Error with user fullname template %s : %s", d.conf.UserFullname, err)
		tmpl, _ = template.New("userfullname").Parse("{{.givenName}}")
	}
	bufFullname := new(bytes.Buffer)
	if err := tmpl.Execute(bufFullname, entry[0].Attributes); err != nil {
		return userInfo, sdk.WithStack(err)
	}

	userInfo.Fullname = bufFullname.String()
	userInfo.Email = entry[0].Attributes["mail"]
	userInfo.ExternalID = entry[0].Attributes["uid"]
	userInfo.Username = req["bind"]

	return userInfo, nil
}

func (d *AuthDriver) openLDAP(ctx context.Context, conf Config) error {
	if d.conn != nil {
		d.conn.Close()
	}
	var err error
	d.conf = conf

	address := fmt.Sprintf("%s:%d", d.conf.Host, d.conf.Port)

	log.Info(ctx, "Auth> Preparing connection to LDAP server: %s", address)
	if !d.conf.SSL {
		d.conn, err = ldap.Dial("tcp", address)
		if err != nil {
			log.Error(ctx, "Auth> Cannot dial %s : %s", address, err)
			return sdk.WithStack(sdk.ErrLDAPConn)
		}

		//Reconnect with TLS
		err = d.conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			log.Error(ctx, "Auth> Cannot start TLS %s : %s", address, err)
			return sdk.WithStack(sdk.ErrLDAPConn)
		}
	} else {
		log.Info(ctx, "Auth> Connecting to LDAP server")
		d.conn, err = ldap.DialTLS("tcp", address, &tls.Config{
			ServerName:         d.conf.Host,
			InsecureSkipVerify: false,
		})
		if err != nil {
			log.Error(ctx, "Auth> Cannot dial TLS (InsecureSkipVerify=false) %s : %s", address, err)
			return sdk.WithStack(sdk.ErrLDAPConn)
		}
	}

	if d.conf.ManagerDN != "" {
		log.Info(ctx, "LDAP> bind manager %s", d.conf.ManagerDN)
		if err := d.conn.Bind(d.conf.ManagerDN, d.conf.ManagerPassword); err != nil {
			if shoudRetry(ctx, err) {
				if err := d.openLDAP(ctx, d.conf); err != nil {
					return err
				}
				if err := d.conn.Bind(d.conf.ManagerDN, d.conf.ManagerPassword); err != nil {
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
func (d *AuthDriver) bind(ctx context.Context, term, password string) error {
	bindRequest := strings.Replace(d.conf.UserSearch, "{0}", ldap.EscapeFilter(term), 1) + "," + d.conf.UserSearchBase + "," + d.conf.RootDN
	log.Debug("LDAP> bind user %s", bindRequest)

	if err := d.conn.Bind(bindRequest, password); err != nil {
		if !shoudRetry(ctx, err) {
			return sdk.WithStack(err)
		}
		if err := d.openLDAP(ctx, d.conf); err != nil {
			return err
		}
		if err := d.conn.Bind(bindRequest, password); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}

//Search search
func (d *AuthDriver) search(ctx context.Context, term string, attributes ...string) ([]Entry, error) {
	userSearch := strings.Replace(d.conf.UserSearch, "{0}", ldap.EscapeFilter(term), 1)
	filter := fmt.Sprintf("(%s)", userSearch)

	log.Debug("LDAP> Search user %s", filter)
	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		d.conf.RootDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1,
		0,
		false,
		filter,
		attributes,
		nil,
	)

	sr, err := d.conn.Search(searchRequest)
	if err != nil {
		if !shoudRetry(ctx, err) {
			return nil, err
		}
		if err := d.openLDAP(ctx, d.conf); err != nil {
			return nil, err
		}
		sr, err = d.conn.Search(searchRequest)
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
