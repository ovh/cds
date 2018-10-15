package auth

import (
	"context"
	"errors"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type contextKey int

const (
	ContextUser contextKey = iota
	ContextHatchery
	ContextWorker
	ContextService
	ContextUserSession
	ContextProvider
)

//Driver is an interface to all auth method (local, ldap and beyond...)
type Driver interface {
	Init(options interface{}) error
}

type Authentifier interface {
	Authentify(username, password string) (bool, error)
}

type RemoteAuthentifier interface {
	AuthentificationURL() (string, error)
	Callback(token string) error
}

//GetDriver is a factory
func GetDriver(c context.Context, mode string, options interface{}, DBFunc func() *gorp.DbMap) (Driver, error) {
	log.Info("Auth> Initializing driver (%s)", mode)
	var d Driver
	switch mode {
	case "ldap":
		d = &LDAPClient{
			dbFunc: DBFunc,
		}
	case "github":
		d = &GithubClient{}
	default:
		d = &LocalClient{
			dbFunc: DBFunc,
		}
	}

	if d == nil {
		return nil, errors.New("GetDriver> Unable to get AuthDriver (nil)")
	}
	if err := d.Init(options); err != nil {
		return nil, sdk.WrapError(err, "GetDriver> Unable to get AuthDriver")
	}
	return d, nil
}

// ContextValues retuns auth values of a context
func ContextValues(ctx context.Context) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		ContextHatchery: ctx.Value(ContextHatchery),
		ContextService:  ctx.Value(ContextService),
		ContextWorker:   ctx.Value(ContextWorker),
		ContextUser:     ctx.Value(ContextUser),
	}
}
