package bitbucket

import (
	"crypto"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
)

// Sign will sign an http.Request using the provided token.
func (c *bitbucketConsumer) Sign(req *http.Request, token Token) error {
	return c.SignParams(req, token, nil)
}

// Sign will sign an http.Request using the provided token, and additional
// parameters.
func (c *bitbucketConsumer) SignParams(req *http.Request, token Token, params map[string]string) error {
	// ensure the parameter map is not nil
	if params == nil {
		params = map[string]string{}
	}

	// ensure default parameters are set
	params["oauth_consumer_key"] = c.ConsumerKey
	params["oauth_nonce"] = nonce()
	params["oauth_signature_method"] = "RSA-SHA1"
	params["oauth_timestamp"] = timestamp()
	params["oauth_version"] = "1.0"

	// we'll need to sign any form values?
	if req.Form != nil {
		for k := range req.Form {
			params[k] = req.Form.Get(k)
		}
	}

	// we'll also need to sign any URL parameter
	queryParams := req.URL.Query()
	for k := range queryParams {
		params[k] = queryParams.Get(k)
	}

	//var tokenSecret string
	if token != nil {
		params["oauth_token"] = token.Token()
	}

	// create the oauth signature
	key := c.PrivateKey
	url := fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path)
	base := requestString(req.Method, url, params)
	var err error
	params["oauth_signature"], err = sign(base, key)
	if err != nil {
		return err
	}

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

// Generates an RSA SHA1 Signature for an OAuth1.0a request.
func sign(message string, key []byte) (string, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return "", fmt.Errorf("Unable to decode key (length: %d)", len(key))
	}

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

	cipher, err := rsa.SignPKCS1v15(crand.Reader, privateKey, crypto.SHA1, rawsignature)
	if err != nil {
		return "", fmt.Errorf("Issue with SignPKCS1v15: %s", err)
	}

	base64signature := make([]byte, base64.StdEncoding.EncodedLen(len(cipher)))
	base64.StdEncoding.Encode(base64signature, cipher)

	return string(base64signature), nil
}
