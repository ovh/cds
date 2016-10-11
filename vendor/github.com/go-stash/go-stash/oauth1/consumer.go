package oauth1

import (
	"crypto"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Out-Of-Band mode, used for applications that do not have
// a callback URL, such as mobile phones or command-line
// utilities.
const OOB = "oob"

// Consumer represents a website or application that uses the
// OAuth 1.0a protocol to access protected resources on behalf
// of a User.
type Consumer struct {
	// A value used by the Consumer to identify itself
	// to the Service Provider.
	ConsumerKey string

	// A secret used by the Consumer to establish
	// ownership of the Consumer Key.
	ConsumerSecret string

	// A path to the private pem key used to sign request
	ConsumerPrivateKeyPem string

	// An absolute URL to which the Service Provider will redirect
	// the User back when the Obtaining User Authorization step
	// is completed.
	//
	// If the Consumer is unable to receive callbacks or a callback
	// URL has been established via other means, the parameter
	// value MUST be set to oob (case sensitive), to indicate
	// an out-of-band configuration.
	CallbackURL string

	// The URL used to obtain an unauthorized
	// Request Token.
	RequestTokenURL string

	// The URL used to obtain User authorization
	// for Consumer access.
	AccessTokenURL string

	// The URL used to exchange the User-authorized
	// Request Token for an Access Token.
	AuthorizationURL string
}

func (c *Consumer) RequestToken() (*RequestToken, error) {

	// create the http request to fetch a Request Token.
	requestTokenUrl, _ := url.Parse(c.RequestTokenURL)
	req := http.Request{
		URL:        requestTokenUrl,
		Method:     "POST",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Close:      true,
	}

	// sign the request
	err := c.SignParams(&req, nil, map[string]string{"oauth_callback": c.CallbackURL})
	if err != nil {
		return nil, err
	}

	// make the http request and get the response
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}

	// parse the Request's Body
	requestToken, err := ParseRequestToken(resp.Body)
	if err != nil {
		return nil, err
	}

	return requestToken, nil
}

// AuthorizeRedirect constructs the request URL that should be used
// to redirect the User to verify User identify and consent.
func (c *Consumer) AuthorizeRedirect(t *RequestToken) (string, error) {
	redirect, err := url.Parse(c.AuthorizationURL)
	if err != nil {
		return "", err
	}

	params := make(url.Values)
	params.Add("oauth_token", t.token)
	redirect.RawQuery = params.Encode()

	u := redirect.String()
	if strings.HasPrefix(u, "https://bitbucket.org/%21api/") {
		u = strings.Replace(u, "/%21api/", "/!api/", -1)
	}

	return u, nil
}

func (c *Consumer) AuthorizeToken(t *RequestToken, verifier string) (*AccessToken, error) {

	// create the http request to fetch a Request Token.
	accessTokenUrl, err := url.Parse(c.AccessTokenURL)
	if err != nil {
		return nil, err
	}
	req := http.Request{
		URL:        accessTokenUrl,
		Method:     "POST",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Close:      true,
	}

	// sign the request
	if err := c.SignParams(&req, t, map[string]string{"oauth_verifier": verifier}); err != nil {
		return nil, err
	}

	// make the http request and get the response
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}

	// parse the Request's Body
	accessToken, err := ParseAccessToken(resp.Body)
	if err != nil {
		return nil, err
	}

	return accessToken, nil
}

// Sign will sign an http.Request using the provided token.
func (c *Consumer) Sign(req *http.Request, token Token) error {
	return c.SignParams(req, token, nil)
}

// Sign will sign an http.Request using the provided token, and additional
// parameters.
func (c *Consumer) SignParams(req *http.Request, token Token, params map[string]string) error {

	// ensure the parameter map is not nil
	if params == nil {
		params = map[string]string{}
	}

	// ensure default parameters are set
	//params["oauth_token"] = token.Token()
	params["oauth_consumer_key"] = c.ConsumerKey
	params["oauth_nonce"] = nonce()
	params["oauth_signature_method"] = "RSA-SHA1"
	params["oauth_timestamp"] = timestamp()
	params["oauth_version"] = "1.0"

	// we'll need to sign any form values?
	if req.Form != nil {
		for k, _ := range req.Form {
			params[k] = req.Form.Get(k)
		}
	}

	// we'll also need to sign any URL parameter
	queryParams := req.URL.Query()
	for k, _ := range queryParams {
		params[k] = queryParams.Get(k)
	}

	//var tokenSecret string
	if token != nil {
		//tokenSecret = token.Secret()
		params["oauth_token"] = token.Token()
	}

	// create the oauth signature
	//key := escape(c.ConsumerSecret) + "&" + escape(tokenSecret)
	key := c.ConsumerPrivateKeyPem
	url := fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path)
	base := requestString(req.Method, url, params)
	var err error
	params["oauth_signature"], err = sign(base, key)
	if err != nil {
		return err
	}

	//HACK: we were previously including params in the Authorization
	//      header that shouldn't be. so for now, we'll filter
	//authStringParams := map[string]string{}
	//for k,v := range params {
	//	if strings.HasPrefix(k, "oauth_") {
	//		authStringParams[k]=v
	//	}
	//}

	// ensure the http.Request's Header is not nil
	if req.Header == nil {
		req.Header = http.Header{}
	}

	// add the authorization header string
	req.Header.Add("Authorization", authorizationString(params)) //params))

	// ensure the appropriate content-type is set for POST,
	// assuming the field is not populated
	if (req.Method == "POST" || req.Method == "PUT") && len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	return nil
}

// -----------------------------------------------------------------------------
// Private Helper Functions

// Nonce generator, seeded with current time
var nonceGenerator = rand.New(rand.NewSource(time.Now().Unix()))

// Nonce generates a random string. Nonce's are uniquely generated
// for each request.
func nonce() string {
	return strconv.FormatInt(nonceGenerator.Int63(), 10)
}

// Timestamp generates a timestamp, expressed in the number of seconds
// since January 1, 1970 00:00:00 GMT.
func timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// Generates an RSA SHA1 Signature for an OAuth1.0a request.
func sign(message, key string) (string, error) {
	privateKeyBytes, err := ioutil.ReadFile(key)
	if err != nil {
		return "", fmt.Errorf("Issue reading private key: %s", err)
	}
	block, _ := pem.Decode(privateKeyBytes)

	// try to parse private key in PKCS8 format first
	var privateKey *rsa.PrivateKey
	privateInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		var b bool
		privateKey, b = privateInterface.(*rsa.PrivateKey)
		if !b {
			return "", fmt.Errorf("Issue casting key:s %s", err)
		}
	} else {
		// fall back to PKCS1 if it fails
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("Issue parsing private key: %s", err)
		}
	}

	hashfun := sha1.New()
	hashfun.Write([]byte(message))
	rawsignature := hashfun.Sum(nil)

	cipher, err := rsa.SignPKCS1v15(crand.Reader, privateKey, crypto.SHA1,
		rawsignature)
	if err != nil {
		return "", fmt.Errorf("Issue with SignPKCS1v15: %s", err)
	}

	base64signature := make([]byte, base64.StdEncoding.EncodedLen(len(cipher)))
	base64.StdEncoding.Encode(base64signature, cipher)

	return string(base64signature), nil
}

// Gets the default set of OAuth1.0a headers.
func headers(consumerKey string) map[string]string {
	return map[string]string{
		"oauth_consumer_key":     consumerKey,
		"oauth_nonce":            nonce(),
		"oauth_signature_method": "RSA-SHA1",
		"oauth_timestamp":        timestamp(),
		"oauth_version":          "1.0",
	}
}

func requestString(method string, uri string, params map[string]string) string {

	// loop through params, add keys to map
	var keys []string
	for key, _ := range params {
		keys = append(keys, key)
	}

	// sort the array of header keys
	sort.StringSlice(keys).Sort()

	// create the signed string
	result := method + "&" + escape(uri)

	// loop through sorted params and append to the string
	for pos, key := range keys {
		if pos == 0 {
			result += "&"
		} else {
			result += escape("&")
		}

		result += escape(fmt.Sprintf("%s=%s", key, escape(params[key])))
	}

	return result
}

func authorizationString(params map[string]string) string {

	// loop through params, add keys to map
	var keys []string
	for key, _ := range params {
		keys = append(keys, key)
	}

	// sort the array of header keys
	sort.StringSlice(keys).Sort()

	// create the signed string
	var str string
	var cnt = 0

	// loop through sorted params and append to the string
	for _, key := range keys {

		// we previously encoded all params (url params, form data & oauth params)
		// but for the authorization string we should only encode the oauth params
		if !strings.HasPrefix(key, "oauth_") {
			continue
		}

		if cnt > 0 {
			str += ","
		}

		str += fmt.Sprintf("%s=%q", key, escape(params[key]))
		cnt++
	}

	return fmt.Sprintf("OAuth %s", str)
}

func escape(s string) string {
	t := make([]byte, 0, 3*len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isEscapable(c) {
			t = append(t, '%')
			t = append(t, "0123456789ABCDEF"[c>>4])
			t = append(t, "0123456789ABCDEF"[c&15])
		} else {
			t = append(t, s[i])
		}
	}
	return string(t)
}

func isEscapable(b byte) bool {
	return !('A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' || '0' <= b && b <= '9' || b == '-' || b == '.' || b == '_' || b == '~')

}
