package localauthentication

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

var _ authentication.Driver = new(localAuthentication)

func New() authentication.Driver {
	return &localAuthentication{}
}

type localAuthentication struct{}

func (l *localAuthentication) CheckAuthentication(ctx context.Context, db gorp.SqlExecutor, r *http.Request) (*sdk.AuthentifiedUser, error) {
	_, end := observability.Span(ctx, "localauthentication.CheckAuthentication")
	defer end()

	vars := mux.Vars(r)
	username := vars["username"]
	password := vars["password"]

	ok, err := Authentify(db, username, password)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, sdk.ErrUnauthorized
	}

	u, err := user.LoadUserByUsername(db, username)
	if err != nil {
		return nil, err
	}

	return u, nil
}
