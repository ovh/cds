package ldap

import (
	"context"

	"github.com/rockbears/log"
	"gopkg.in/ldap.v2"
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

// Entry represents a LDAP entity
type Entry struct {
	DN         string
	Attributes map[string]string
}
