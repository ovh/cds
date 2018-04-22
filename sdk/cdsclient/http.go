package cdsclient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/ovh/cds/sdk"
)

const (
	//SessionTokenHeader is user as HTTP header
	SessionTokenHeader = "Session-Token"
	// AuthHeader is used as HTTP header
	AuthHeader = "X_AUTH_HEADER"
	// RequestedWithHeader is used as HTTP header
	RequestedWithHeader = "X-Requested-With"
	// RequestedWithValue is used as HTTP header
	RequestedWithValue = "X-CDS-SDK"
	// RequestedNameHeader is used as HTTP header
	RequestedNameHeader = "X-Requested-Name"
	// RequestedIfModifiedSinceHeader is used as HTTP header
	RequestedIfModifiedSinceHeader = "If-Modified-Since"

	// ResponseAPITimeHeader is used as HTTP header
	ResponseAPITimeHeader = "X-Api-Time"
	// ResponseAPINanosecondsTimeHeader is used as HTTP header
	ResponseAPINanosecondsTimeHeader = "X-Api-Nanoseconds-Time"
	// ResponseEtagHeader is used as HTTP header
	ResponseEtagHeader = "Etag"
	// ResponseProcessTimeHeader is used as HTTP header
	ResponseProcessTimeHeader = "X-Api-Process-Time"
)

// RequestModifier is used to modify behavior of Request and Steam functions
type RequestModifier func(req *http.Request)

// HTTPClient is a interface for HTTPClient mock
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// NoTimeout returns a http.DefaultClient from a HTTPClient
func NoTimeout(c HTTPClient) HTTPClient {
	return http.DefaultClient
}

// SetHeader modify headers of http.Request
func SetHeader(key, value string) RequestModifier {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// PostJSON post the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) PostJSON(path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(http.MethodPost, path, in, out, mods...)
	return code, err
}

// PostJSON ut the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) PutJSON(path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(http.MethodPut, path, in, out, mods...)
	return code, err
}

// GetJSON get the requested path If set, it unmarshalls the response to *out*
func (c *client) GetJSON(path string, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(http.MethodGet, path, nil, out, mods...)
	return code, err
}

// GetJSONWithHeaders get the requested path If set, it unmarshalls the response to *out* and return response headers
func (c *client) GetJSONWithHeaders(path string, out interface{}, mods ...RequestModifier) (http.Header, int, error) {
	_, header, code, err := c.RequestJSON(http.MethodGet, path, nil, out, mods...)
	return header, code, err
}

// DeleteJSON deletes the requested path If set, it unmarshalls the response to *out*
func (c *client) DeleteJSON(path string, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(http.MethodDelete, path, nil, out, mods...)
	return code, err
}

// RequestJSON does a request with the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) RequestJSON(method, path string, in interface{}, out interface{}, mods ...RequestModifier) ([]byte, http.Header, int, error) {
	var b = []byte{}
	var err error

	if in != nil {
		b, err = json.Marshal(in)
		if err != nil {
			return nil, nil, 0, err
		}
	}

	var body io.Reader
	if len(b) > 0 {
		body = bytes.NewBuffer(b)
	}

	res, header, code, err := c.Request(method, path, body, mods...)
	if err != nil {
		fmt.Printf("##### out AA : %+v code:%v", err, code)
		return nil, nil, code, err
	}

	if out != nil {
		fmt.Printf("##### ou : %+v", out)
		if err := json.Unmarshal(res, out); err != nil {
			fmt.Printf("##### out err : %+v", err)
			return res, nil, code, err
		}
	}

	if code >= 400 {
		return res, nil, code, fmt.Errorf("HTTP %d", code)
	}
	return res, header, code, nil
}

// Request executes an authentificated HTTP request on $path given $method and $args
func (c *client) Request(method string, path string, body io.Reader, mods ...RequestModifier) ([]byte, http.Header, int, error) {
	respBody, respHeader, code, err := c.Stream(method, path, body, false, mods...)
	if err != nil {
		return nil, nil, 0, err
	}
	defer func() {
		// Drain and close the body to let the Transport reuse the connection
		io.Copy(ioutil.Discard, respBody)
		respBody.Close()
	}()

	var bodyBtes []byte
	bodyBtes, err = ioutil.ReadAll(respBody)
	if err != nil {
		return nil, nil, code, err
	}

	if c.config.Verbose {
		if len(bodyBtes) > 0 {
			log.Printf("Response Body: %s\n", bodyBtes)
		}
	}

	if err := sdk.DecodeError(bodyBtes); err != nil {
		return nil, nil, code, err
	}

	return bodyBtes, respHeader, code, nil
}

// Stream makes an authenticated http request and return io.ReadCloser
func (c *client) Stream(method string, path string, body io.Reader, noTimeout bool, mods ...RequestModifier) (io.ReadCloser, http.Header, int, error) {
	var savederror error

	var bodyContent []byte
	var err error
	if body != nil {
		bodyContent, err = ioutil.ReadAll(body)
		if err != nil {
			return nil, nil, 0, err
		}
	}

	url := c.config.Host + path
	if strings.HasPrefix(path, "http") {
		url = path
	}

	for i := 0; i <= c.config.Retry; i++ {
		req, requestError := http.NewRequest(method, url, bytes.NewBuffer(bodyContent))
		if requestError != nil {
			savederror = requestError
			continue
		}

		for i := range mods {
			if mods[i] != nil {
				mods[i](req)
			}
		}

		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		req.Header.Set("User-Agent", c.config.userAgent)
		req.Header.Set("Connection", "close")
		req.Header.Add(RequestedWithHeader, RequestedWithValue)
		if c.name != "" {
			req.Header.Add(RequestedNameHeader, c.name)
		}

		//No auth on /login route
		if !strings.HasPrefix(path, "/login") {
			if c.config.Hash != "" {
				basedHash := base64.StdEncoding.EncodeToString([]byte(c.config.Hash))
				req.Header.Set(AuthHeader, basedHash)
			}
			if c.config.User != "" && c.config.Token != "" {
				req.Header.Add(SessionTokenHeader, c.config.Token)
				req.SetBasicAuth(c.config.User, c.config.Token)
			}
		}

		if c.config.Verbose {
			log.Println("********REQUEST**********")
			dmp, _ := httputil.DumpRequestOut(req, true)
			log.Printf("%s", string(dmp))
		}

		var errDo error
		var resp *http.Response
		if noTimeout {
			resp, errDo = NoTimeout(c.HTTPClient).Do(req)
		} else {
			resp, errDo = c.HTTPClient.Do(req)
		}

		if errDo == nil && c.config.Verbose {
			log.Println("********RESPONSE**********")
			dmp, _ := httputil.DumpResponse(resp, true)
			log.Printf("%s", string(dmp))
			log.Println("**************************")
		}

		// if everything is fine, return body
		if errDo == nil && resp.StatusCode < 500 {
			return resp.Body, resp.Header, resp.StatusCode, nil
		}

		// if no request error by status > 500, check CDS error
		// if there is a CDS errors, return it
		if errDo == nil && resp.StatusCode == 500 {
			var body []byte
			var errRead error
			body, errRead = ioutil.ReadAll(resp.Body)
			if errRead != nil {
				resp.Body.Close()
				continue
			}
			if cdserr := sdk.DecodeError(body); cdserr != nil {
				resp.Body.Close()
				return nil, resp.Header, resp.StatusCode, cdserr
			}
		}

		if resp != nil && resp.StatusCode >= 500 {
			savederror = fmt.Errorf("HTTP %d", resp.StatusCode)
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			continue
		}

		if errDo != nil && (strings.Contains(errDo.Error(), "connection reset by peer") ||
			strings.Contains(errDo.Error(), "unexpected EOF")) {
			savederror = errDo
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			continue
		}

		if errDo != nil {
			return nil, nil, 0, errDo
		}
	}

	return nil, nil, 0, fmt.Errorf("x%d: %s", c.config.Retry, savederror)
}

// UploadMultiPart upload multipart
func (c *client) UploadMultiPart(method string, path string, body *bytes.Buffer, mods ...RequestModifier) ([]byte, int, error) {
	var req *http.Request
	req, errRequest := http.NewRequest(method, c.config.Host+path, body)
	if errRequest != nil {
		return nil, 0, errRequest
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", c.config.userAgent)
	req.Header.Set("Connection", "close")
	req.Header.Add(RequestedWithHeader, RequestedWithValue)

	for i := range mods {
		mods[i](req)
	}

	//No auth on /login route
	if !strings.HasPrefix(path, "/login") {
		if c.config.Hash != "" {
			basedHash := base64.StdEncoding.EncodeToString([]byte(c.config.Hash))
			req.Header.Set(AuthHeader, basedHash)
		}
		if c.config.User != "" && c.config.Token != "" {
			req.Header.Add(SessionTokenHeader, c.config.Token)
			req.SetBasicAuth(c.config.User, c.config.Token)
		}
	}

	resp, err := NoTimeout(c.HTTPClient).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if c.config.Verbose {
		fmt.Printf("Response Status: %s\n", resp.Status)
		fmt.Printf("Request path: %s\n", c.config.Host+path)
		fmt.Printf("Request Headers: %s\n", req.Header)
		fmt.Printf("Response Headers: %s\n", resp.Header)
	}

	var respBody []byte
	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if c.config.Verbose {
		if len(body.Bytes()) > 0 {
			fmt.Printf("Response Body: %s\n", body.String())
		}
	}

	return respBody, resp.StatusCode, nil
}
