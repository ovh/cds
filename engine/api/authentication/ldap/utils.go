package ldap

import (
	"context"

	"gopkg.in/ldap.v2"

	"github.com/ovh/cds/sdk/log"
)

func shoudRetry(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	ldapErr, ok := err.(*ldap.Error)
	if !ok {
		return false
	}
	if ldapErr.ResultCode == ldap.ErrorNetwork {
		log.Info(ctx, "LDAP> Retry")
		return true
	}
	return false
}

//Entry represents a LDAP entity
type Entry struct {
	DN         string
	Attributes map[string]string
}
