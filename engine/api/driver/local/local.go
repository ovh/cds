package local

import (
	"context"
	"github.com/nbutton23/zxcvbn-go"
	"github.com/ovh/cds/sdk"
	"strings"
)

var _ sdk.Driver = new(LocalDriver)

// NewDriver returns a new initialized driver for local authentication.
func NewLocalDriver(ctx context.Context, allowedDomains string) sdk.Driver {
	var domains []string
	if allowedDomains != "" {
		domains = strings.Split(allowedDomains, ",")
	}
	return &LocalDriver{
		allowedDomains: domains,
	}
}

// AuthDriver for local authentication.
type LocalDriver struct {
	allowedDomains []string
}

// CheckSignupRequest checks that given driver request is valid for a signup with auth local.
func (d LocalDriver) CheckSignupRequest(req sdk.AuthConsumerSigninRequest) error {
	if fullname, err := req.StringE("fullname"); err != nil || fullname == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing fullname for local signup")
	}
	if username, err := req.StringE("username"); err != nil || username == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid username for local signup")
	}
	if email, err := req.StringE("email"); err != nil || !sdk.IsValidEmail(email) || !d.isAllowedDomain(email) {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid email for local signup")
	}
	if password, err := req.StringE("password"); err != nil || password == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid password for local signup")
	} else if err := isPasswordValid(password); err != nil {
		return err
	}
	return nil
}

// CheckSigninRequest checks that given driver request is valid for a signin with auth local.
func (d LocalDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	if username, err := req.StringE("username"); err != nil || username == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid username for local signin")
	}
	if password, err := req.StringE("password"); err != nil || password == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid password for local signin")
	}
	return nil
}

// CheckVerifyRequest checks that given driver request is valid for a verify consumer.
func (d LocalDriver) CheckVerifyRequest(req sdk.AuthConsumerSigninRequest) error {
	if token, err := req.StringE("token"); err != nil || token == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid token for local verify")
	}
	return nil
}

// CheckAskResetRequest checks that given driver request is valid for a ask reset with auth local.
func (d LocalDriver) CheckAskResetRequest(req sdk.AuthConsumerSigninRequest) error {
	if email, err := req.StringE("email"); err != nil || !sdk.IsValidEmail(email) {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid email for local signin")
	}
	return nil
}

// CheckResetRequest checks that given driver request is valid for a reset with auth local.
func (d LocalDriver) CheckResetRequest(req sdk.AuthConsumerSigninRequest) error {
	if token, err := req.StringE("token"); err != nil || token == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid token for local reset")
	}
	if password, err := req.StringE("password"); err != nil || password == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid password for local signup")
	} else if err := isPasswordValid(password); err != nil {
		return err
	}
	return nil
}

// isAllowedDomain return true is email is allowed, false otherwise.
func (d LocalDriver) isAllowedDomain(email string) bool {
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

func (d LocalDriver) GetUserInfoFromDriver(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	return sdk.AuthDriverUserInfo{}, nil
}

func isPasswordValid(password string) error {
	if len(password) > 256 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "given password is not strong enough, level should be >= 3")
	}
	passwordStrength := zxcvbn.PasswordStrength(password, nil).Score
	if passwordStrength < 3 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "given password is not strong enough, level should be >= 3")
	}
	return nil
}
