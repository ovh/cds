package local

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

var _ sdk.AuthDriver = new(localAuthentication)

func NewDriver(allowedDomains string) sdk.AuthDriver {
	var domains []string

	if allowedDomains != "" {
		domains = strings.Split(allowedDomains, ",")
	}

	return &localAuthentication{
		allowedDomains: domains,
	}
}

type localAuthentication struct {
	allowedDomains []string
}

func (l localAuthentication) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:   sdk.ConsumerLocal,
		Method: http.MethodPost,
		URL:    "/auth/consumer/local/signup",
		Fields: []sdk.AuthDriverManifestField{
			{
				Name: "fullname",
				Type: sdk.FieldString,
			},
			{
				Name: "username",
				Type: sdk.FieldString,
			},
			{
				Name: "email",
				Type: sdk.FieldEmail,
			},
			{
				Name: "password",
				Type: sdk.FieldPassword,
			},
		},
	}
}

func (l localAuthentication) CheckRequest(req sdk.AuthDriverRequest) error {
	if err := req.IsValid(l.GetManifest()); err != nil {
		return err
	}

	email := req["email"]
	if !l.IsAllowedDomain(email) {
		return sdk.NewErrorFrom(sdk.ErrInvalidEmailDomain, "email address %s does not have a valid domain", email)
	}

	password := req["password"]
	if err := isPasswordValid(password); err != nil {
		return err
	}

	return nil
}

func (l *localAuthentication) CheckAuthentication(ctx context.Context, db gorp.SqlExecutor, r *http.Request) (*sdk.AuthentifiedUser, error) {
	/*_, end := observability.Span(ctx, "localauthentication.CheckAuthentication")
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

	return u, nil*/

	return nil, nil
}

// IsAllowedDomain return true is email is allowed, false otherwise.
func (l localAuthentication) IsAllowedDomain(email string) bool {
	if len(l.allowedDomains) == 0 {
		return true
	}

	for _, domain := range l.allowedDomains {
		if strings.HasSuffix(email, "@"+domain) && strings.Count(email, "@") == 1 {
			return true
		}
	}
	return false
}
