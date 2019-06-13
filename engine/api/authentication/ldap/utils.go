package ldap

import (
	"errors"

	"gopkg.in/ldap.v2"

	"github.com/ovh/cds/sdk/log"
)

var errUserNotFound = errors.New("user not found")

func shoudRetry(err error) bool {
	if err == nil {
		return false
	}
	ldapErr, ok := err.(*ldap.Error)
	if !ok {
		return false
	}
	if ldapErr.ResultCode == ldap.ErrorNetwork {
		log.Info("LDAP> Retry")
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

//Entry represents a LDAP entity
type Entry struct {
	DN         string
	Attributes map[string]string
}
