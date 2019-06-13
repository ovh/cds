package localauthentication

import (
	"context"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/nbutton23/zxcvbn-go"
	"golang.org/x/crypto/bcrypt"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

var _ sdk.AuthDriver = new(localAuthentication)

func New(allowedDomains string) sdk.AuthDriver {
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
	if err := l.IsPasswordValid(password); err != nil {
		return err
	}

	return nil
}

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

func (l localAuthentication) IsPasswordValid(password string) error {
	passwordStrength := zxcvbn.PasswordStrength(password, nil).Score
	if passwordStrength < 3 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "given password is not strong enough")
	}
	return nil
}

func GenerateVerifyToken(consumerID string) (string, error) {
	issuedAt := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.StandardClaims{
		Issuer:    accesstoken.IssuerName,
		Subject:   consumerID,
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: issuedAt.Add(24 * time.Hour).Unix(),
	})
	return accesstoken.SignJWT(jwtToken)
}

// IsAllowedDomain return true is email is allowed, false otherwise.
func (l localAuthentication) IsAllowedDomain(email string) bool {
	for _, domain := range l.allowedDomains {
		if strings.HasSuffix(email, "@"+domain) && strings.Count(email, "@") == 1 {
			return true
		}
	}
	return false
}

// HashPassword returns a hash from given password.
func HashPassword(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrUnknownError, "cannot generate hash for given password"))
	}
	return hash, nil
}

// CompareHashAndPassword returns an error if given hash and password don't match.
func CompareHashAndPassword(hash []byte, password string) error {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return sdk.NewErrorWithStack(err, sdk.WithStack(sdk.ErrInvalidPassword))
	}
	return nil
}
