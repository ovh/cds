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
	Host           string // 192.168.1.32
	Port           int    // 636
	SSL            bool   // true
	UserSearchBase string // ou=people,dc=ejnserver,dc=fr
	UserSearch     string // uid={{.search}}
	UserDN         string // cn={0},ou=people,dc=ejnserver,dc=fr
	UserFullname   string // {{.givenName}} {{.sn}}
	UserEmail      string // {{.email}}
	UserExternalID string // {{.uid}}
	Attributes     string // uid,dn,cn,ou,givenName,sn,displayName,mail,memberOf
	BindDN         string // cn=admin,dc=ejnserver,dc=fr
	BindPW         string // BIND_PASSWORD
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

func (d AuthDriver) GetSessionDuration() time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

func (d AuthDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	if user, ok := req["user"]; !ok || user == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid user term for ldap signin")
	}
	if password, ok := req["password"]; !ok || password == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid password for ldap signin")
	}
	return nil
}

func (d AuthDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var userInfo sdk.AuthDriverUserInfo
	var user = req["user"]
	var password = req["password"]

	if d.conf.BindDN != "" {
		if err := d.bindDN(ctx, d.conf.BindDN, d.conf.BindPW); err != nil {
			return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
		}
	} else {
		userDN := strings.Replace(d.conf.UserDN, "{0}", ldap.EscapeFilter(user), 1)

		if err := d.bindDN(ctx, userDN, password); err != nil {
			return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
		}
	}

	attributes := strings.Split(d.conf.Attributes, ",")
	entry, err := d.search(ctx, user, attributes...)
	if err != nil && err.Error() != errUserNotFound {
		return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	if len(entry) > 1 {
		return userInfo, fmt.Errorf("LDAP Search error multiple values")
	}

	if d.conf.BindDN != "" {
		if err := d.bindDN(ctx, entry[0].DN, password); err != nil {
			return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
		}
	}

	//If user doesn't exist and search was'nt successful => exist
	if err != nil {
		log.Warning(ctx, "LDAP> Search error %s", err)
		return userInfo, sdk.NewError(sdk.ErrUnauthorized, err)
	}

	//Execute templates to compute user info
	fullname, err := d.computeTemplate(ctx, "fullname", d.conf.UserFullname, "{{.givenName}}", entry[0].Attributes)
	email, err := d.computeTemplate(ctx, "email", d.conf.UserEmail, "{{.mail}}", entry[0].Attributes)
	externalID, err := d.computeTemplate(ctx, "externalid", d.conf.UserExternalID, "{{.uid}}", entry[0].Attributes)

	userInfo.Fullname = fullname
	userInfo.Email = email
	userInfo.ExternalID = externalID
	userInfo.Username = req["user"]

	return userInfo, nil
}

func (d *AuthDriver) computeTemplate(ctx context.Context, name string, parse string, backup string, data map[string]string) (string, error) {
	tmpl, err := template.New(name).Parse(parse)
	if err != nil {
		log.Error(ctx, "LDAP> Error with user %s template %s : %s", name, parse, err)
		tmpl, _ = template.New(name).Parse(backup)
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}

	log.Debug("LDAP> User %s template %s = %s", name, parse, buf.String())
	return buf.String(), err
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

	if d.conf.BindDN != "" {
		if err := d.bindDN(ctx, d.conf.BindDN, d.conf.BindPW); err != nil {
			return err
		}
	}

	return nil
}

// bindDN binds
func (d *AuthDriver) bindDN(ctx context.Context, dn, password string) error {
	log.Info(ctx, "LDAP> Bind DN %s", dn)

	if err := d.conn.Bind(dn, password); err != nil {
		if !shoudRetry(ctx, err) {
			return sdk.WithStack(err)
		}
		if err := d.openLDAP(ctx, d.conf); err != nil {
			return err
		}
		if err := d.conn.Bind(dn, password); err != nil {
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
		d.conf.UserSearchBase,
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
