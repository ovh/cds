package local

import (
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
)

var _ sdk.AuthDriver = new(AuthDriver)

// NewDriver returns a new initialized driver for local authentication.
func NewDriver(signupDisabled bool, uiURL, allowedDomains string) sdk.AuthDriver {
	var domains []string

	if allowedDomains != "" {
		domains = strings.Split(allowedDomains, ",")
	}

	return &AuthDriver{
		signupDisabled: signupDisabled,
		allowedDomains: domains,
	}
}

// AuthDriver for local authentication.
type AuthDriver struct {
	signupDisabled bool
	uiURL          string
	allowedDomains []string
}

func (d AuthDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerLocal,
		SignupDisabled: d.signupDisabled,
	}
}

func (d AuthDriver) GetSigninURI(state string) string {
	return "/"
}

func (d AuthDriver) GetSessionDuration() time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

// CheckSignupRequest checks that given driver request is valid for a signup with auth local.
func (d AuthDriver) CheckSignupRequest(req sdk.AuthConsumerSigninRequest) error {
	if fullname, ok := req["fullname"]; !ok || fullname == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing fullname for local signup")
	}
	if username, ok := req["username"]; !ok || username == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing username for local signup")
	}
	if email, ok := req["email"]; !ok || email == "" || !d.isAllowedDomain(email) {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid email for local signup")
	}
	if password, ok := req["password"]; !ok || password == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid password for local signup")
	} else if err := isPasswordValid(password); err != nil {
		return err
	}
	return nil
}

// CheckSigninRequest checks that given driver request is valid for a signin with auth local.
func (d AuthDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	if email, ok := req["email"]; !ok || email == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid email for local signin")
	}
	if password, ok := req["password"]; !ok || password == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid password for local signin")
	}
	return nil
}

// isAllowedDomain return true is email is allowed, false otherwise.
func (d AuthDriver) isAllowedDomain(email string) bool {
	if len(d.allowedDomains) == 0 {
		return true
	}
	for _, domain := range d.allowedDomains {
		if strings.HasSuffix(email, "@"+domain) && strings.Count(email, "@") == 1 {
			return true
		}
	}
	return false
}

func (d AuthDriver) GetUserInfo(req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	// not used for local auth
	return sdk.AuthDriverUserInfo{}, nil
}
