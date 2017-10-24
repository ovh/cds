package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"
)

// HTTPClient will be set to a default httpclient if not set
var HTTPClient sdk.HTTPClient

// DoJSONRequest performs an http request on a service
func DoJSONRequest(srvs []sdk.Service, method, path string, in interface{}, out interface{}, mods ...sdk.RequestModifier) (int, error) {
	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			code, err := doJSONRequest(srv, "GET", "/vcs", in, out, mods...)
			if err == nil {
				return code, nil
			}
			lastErr = err
			lastCode = code
		}
		if lastErr != nil || attempt > 5 {
			break
		}
	}
	return lastCode, lastErr
}

// DoJSONRequest performs an http request on service
func doJSONRequest(srv *sdk.Service, method, path string, in interface{}, out interface{}, mods ...sdk.RequestModifier) (int, error) {
	var b = []byte{}
	var err error

	if in != nil {
		b, err = json.Marshal(in)
		if err != nil {
			return 0, err
		}
	}

	mods = append(mods, sdk.SetHeader("Content-Type", "application/json"))
	res, code, err := DoRequest(srv, method, path, b, mods...)
	if err != nil {
		return code, err
	}

	if out != nil {
		if err := json.Unmarshal(res, out); err != nil {
			return code, err
		}
	}

	return code, nil
}

// DoRequest performs an http request on service
func DoRequest(srv *sdk.Service, method, path string, args []byte, mods ...sdk.RequestModifier) ([]byte, int, error) {
	if HTTPClient == nil {
		HTTPClient = &http.Client{
			Timeout: 2 * time.Second,
		}
	}

	var requestError error
	var req *http.Request
	if args != nil {
		req, requestError = http.NewRequest(method, srv.HTTPURL+path, bytes.NewReader(args))
	} else {
		req, requestError = http.NewRequest(method, srv.HTTPURL+path, nil)
	}
	if requestError != nil {
		return nil, 0, requestError
	}

	req.Header.Set("Connection", "close")
	req.Header.Add(sdk.RequestedWithHeader, sdk.RequestedWithValue)
	for i := range mods {
		if mods[i] != nil {
			mods[i](req)
		}
	}

	// Authentify the request with the hash
	basedHash := base64.StdEncoding.EncodeToString([]byte(srv.Hash))
	req.Header.Set(sdk.AuthHeader, basedHash)

	//Do the request
	resp, errDo := HTTPClient.Do(req)
	if errDo != nil {
		return nil, resp.StatusCode, sdk.WrapError(errDo, "services.DoRequest> Request failed")
	}
	defer resp.Body.Close()

	// Read the body
	body, errBody := ioutil.ReadAll(resp.Body)
	if errBody != nil {
		return nil, resp.StatusCode, sdk.WrapError(errBody, "services.DoRequest> Unable to read body")
	}

	// if everything is fine, return body
	if resp.StatusCode < 500 {
		return body, resp.StatusCode, nil
	}

	// Try to catch the CDS Error
	if cdserr := sdk.DecodeError(body); cdserr != nil {
		return nil, resp.StatusCode, cdserr
	}

	return nil, resp.StatusCode, fmt.Errorf("Request Failed")
}
