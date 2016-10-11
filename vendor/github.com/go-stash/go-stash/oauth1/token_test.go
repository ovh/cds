package oauth1

import (
	"net/url"
	"strconv"
	"testing"
)

func TestNewAccessToken(t *testing.T) {
	oauth_token := "o-token"
	oauth_secret := "o-secret"
	oauth_params := map[string]string{}
	oauth_params["test"] = "here"

	n := NewAccessToken(oauth_token, oauth_secret, oauth_params)

	if n.Token() != oauth_token {
		t.Errorf("Expected Token %v, got %v", oauth_token, n.Token())
	}

	if n.Secret() != oauth_secret {
		t.Errorf("Expected Secret %v, got %v", oauth_secret, n.Secret())
	}

	p := n.Params()
	if p["test"] != "here" {
		t.Errorf("Expected Params %v, got %v", oauth_params, n.Params())
	}
}

// Test the ability to parse a URL query string and unmarshal
// to a RequestToken.
func TestParseRequestTokenStr(t *testing.T) {
	oauth_token := "o-token"
	oauth_token_secret := "o-secret"
	oauth_callback_confirmed := true

	values := url.Values{}
	values.Set("oauth_token", oauth_token)
	values.Set("oauth_token_secret", oauth_token_secret)
	values.Set("oauth_callback_confirmed",
		strconv.FormatBool(oauth_callback_confirmed))

	token, err := ParseRequestTokenStr(values.Encode())
	if err != nil {
		t.Errorf("Expected Request Token parsed, got Error %s", err.Error())
	}
	if token.Token() != oauth_token {
		t.Errorf("Expected Request Token %v, got %v", oauth_token,
			token.Token())
	}
	if token.Secret() != oauth_token_secret {
		t.Errorf("Expected Request Token Secret %v, got %v",
			oauth_token_secret, token.Secret())
	}
}

func TestParseRequestTokenStrEmptyToken(t *testing.T) {
	oauth_token := ""
	oauth_token_secret := "o-secret"
	oauth_callback_confirmed := true

	values := url.Values{}
	values.Set("oauth_token", oauth_token)
	values.Set("oauth_token_secret", oauth_token_secret)
	values.Set("oauth_callback_confirmed",
		strconv.FormatBool(oauth_callback_confirmed))

	token, _ := ParseRequestTokenStr(values.Encode())
	if token != nil {
		t.Errorf("Expected Request Token Empty Token Error")
	}
}

func TestParseRequestTokenStrEmptySecret(t *testing.T) {
	oauth_token := "o-token"
	oauth_secret := ""
	oauth_callback_confirmed := true

	values := url.Values{}
	values.Set("oauth_token", oauth_token)
	values.Set("oauth_token_secret", oauth_secret)
	values.Set("oauth_callback_confirmed",
		strconv.FormatBool(oauth_callback_confirmed))

	token, _ := ParseRequestTokenStr(values.Encode())
	if token != nil {
		t.Errorf("Expected Request Token Empty Secret Error")
	}
}

// Test the ability to Encode a RequestToken to a URL query string.
func TestEncodeRequestToken(t *testing.T) {
	token := RequestToken{
		token:             "o-token",
		secret:            "o-secret",
		callbackConfirmed: true,
	}

	tokenStr := token.Encode()
	expectedStr := "oauth_callback_confirmed=true&oauth_token=o-token&oauth_token_secret=o-secret"
	if tokenStr != expectedStr {
		t.Errorf("Expected Request Token Encoded as %v, got %v",
			expectedStr, tokenStr)
	}
}

// Test the ability to parse a URL query string and unmarshal to
// an AccessToken.
func TestEncodeAccessTokenStr(t *testing.T) {
	oauth_token := "o-token"
	oauth_token_secret := "o-secret"
	oauth_callback_confirmed := true

	values := url.Values{}
	values.Set("oauth_token", oauth_token)
	values.Set("oauth_token_secret", oauth_token_secret)
	values.Set("oauth_callback_confirmed",
		strconv.FormatBool(oauth_callback_confirmed))

	token, err := ParseAccessTokenStr(values.Encode())
	if err != nil {
		t.Errorf("Expected Access Token parsed, got Error %s", err.Error())
	}
	if token.token != oauth_token {
		t.Errorf("Expected Access Token %v, got %v", oauth_token, token.token)
	}
	if token.secret != oauth_token_secret {
		t.Errorf("Expected Access Token Secret %v, got %v",
			oauth_token_secret, token.secret)
	}
}

func TestEncodeAccessTokenStrEmptyToken(t *testing.T) {
	oauth_token := ""
	oauth_token_secret := "o-secret"
	oauth_callback_confirmed := true

	values := url.Values{}
	values.Set("oauth_token", oauth_token)
	values.Set("oauth_token_secret", oauth_token_secret)
	values.Set("oauth_callback_confirmed",
		strconv.FormatBool(oauth_callback_confirmed))

	token, _ := ParseAccessTokenStr(values.Encode())
	if token != nil {
		t.Error("Expected Access Token error")
	}
}

func TestEncodeAccessTokenStrEmptySecret(t *testing.T) {
	oauth_token := "o-token"
	oauth_token_secret := ""
	oauth_callback_confirmed := true

	values := url.Values{}
	values.Set("oauth_token", oauth_token)
	values.Set("oauth_token_secret", oauth_token_secret)
	values.Set("oauth_callback_confirmed",
		strconv.FormatBool(oauth_callback_confirmed))

	token, _ := ParseAccessTokenStr(values.Encode())
	if token != nil {
		t.Error("Expected Access Secret error")
	}
}

// Test the ability to Encode an AccessToken to a URL query string.
func TestEncodeAccessToken(t *testing.T) {
	token := AccessToken{
		token:  "o-token",
		secret: "o-secret",
		params: map[string]string{"user": "dr_no"},
	}

	tokenStr := token.Encode()
	expectedStr := "oauth_token=o-token&oauth_token_secret=o-secret&user=dr_no"
	if tokenStr != expectedStr {
		t.Errorf("Expected Access Token Encoded as %v, got %v",
			expectedStr, tokenStr)
	}
}
