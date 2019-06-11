package localauthentication

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/nbutton23/zxcvbn-go"

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

	ok, err := Authentify(ctx, db, username, password)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, sdk.WithStack(sdk.ErrUnauthorized)
	}

	u, err := user.LoadByUsername(ctx, db, username)
	if err != nil {
		return nil, err
	}

	return u, nil
}

const minPasswordStrength = 3

func CheckPasswordIsValid(password string) error {
	passwordStrength := zxcvbn.PasswordStrength(password, nil).Score
	if passwordStrength < minPasswordStrength {
		return sdk.WithStack(sdk.ErrWrongRequest)
	}
	return nil
}

func GenerateVerifyToken(username string) (string, error) {
	/*token := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.StandardClaims{
		Issuer:    accesstoken.LocalIssuer,
		Subject:   username,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	return accesstoken.Sign(token)*/
	return "", nil
}
