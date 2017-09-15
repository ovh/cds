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
)

// RequestModifier is used to modify behavior of Request and Steam functions
type RequestModifier func(req *http.Request)

// HTTPClient is a interface for HTTPClient mock
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// SetHeader modify headers of http.Request
func SetHeader(key, value string) RequestModifier {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// PostJSON post the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) PostJSON(path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error) {
	return c.RequestJSON(http.MethodPost, path, in, out, mods...)
}

// PostJSON ut the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) PutJSON(path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error) {
	return c.RequestJSON(http.MethodPut, path, in, out, mods...)
}

// GetJSON get the requested path If set, it unmarshalls the response to *out*
func (c *client) GetJSON(path string, out interface{}, mods ...RequestModifier) (int, error) {
	return c.RequestJSON(http.MethodGet, path, nil, out, mods...)
}

// DeleteJSON deletes the requested path If set, it unmarshalls the response to *out*
func (c *client) DeleteJSON(path string, out interface{}, mods ...RequestModifier) (int, error) {
	return c.RequestJSON(http.MethodDelete, path, nil, out, mods...)
}

// RequestJSON does a request with the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) RequestJSON(method, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error) {
	var b = []byte{}
	var err error

	if in != nil {
		b, err = json.Marshal(in)
		if err != nil {
			return 0, err
		}
	}

	res, code, err := c.Request(method, path, b, mods...)
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

// Request executes an authentificated HTTP request on $path given $method and $args
func (c *client) Request(method string, path string, args []byte, mods ...RequestModifier) ([]byte, int, error) {
	respBody, code, err := c.Stream(method, path, args, false, mods...)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		// Drain and close the body to let the Transport reuse the connection
		io.Copy(ioutil.Discard, respBody)
		respBody.Close()
	}()

	var body []byte
	body, err = ioutil.ReadAll(respBody)
	if err != nil {
		return nil, code, err
	}

	if c.config.Verbose {
		if len(body) > 0 {
			log.Printf("Response Body: %s\n", body)
		}
	}

	if err := sdk.DecodeError(body); err != nil {
		return nil, code, err
	}

	return body, code, nil
}

// Stream makes an authenticated http request and return io.ReadCloser
func (c *client) Stream(method string, path string, args []byte, noTimeout bool, mods ...RequestModifier) (io.ReadCloser, int, error) {
	var savederror error

	if c.config.Verbose {
		log.Printf("Request %s Body : %s", c.config.Host+path, string(args))
	}

	for i := 0; i <= c.config.Retry; i++ {
		var requestError error
		var req *http.Request
		if args != nil {
			req, requestError = http.NewRequest(method, c.config.Host+path, bytes.NewReader(args))
		} else {
			req, requestError = http.NewRequest(method, c.config.Host+path, nil)
		}
		if requestError != nil {
			savederror = requestError
			continue
		}

		req.Header.Set("Content-Type", "application/json")
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

		var errDo error
		var resp *http.Response
		if !noTimeout {
			if c.HTTPClientWithoutTimeout == nil {
				return nil, 0, fmt.Errorf("HTTPClientWithoutTimeout is not setted on this client")
			}
			resp, errDo = c.HTTPClientWithoutTimeout.Do(req)
		} else {
			resp, errDo = c.HTTPClient.Do(req)
		}

		// if everything is fine, return body
		if errDo == nil && resp.StatusCode < 500 {
			return resp.Body, resp.StatusCode, nil
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
				return nil, resp.StatusCode, cdserr
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
			return nil, 0, errDo
		}
	}

	return nil, 0, fmt.Errorf("x%d: %s", c.config.Retry, savederror)
}

// UploadMultiPart upload multipart
func (c *client) UploadMultiPart(method string, path string, body *bytes.Buffer, mods ...RequestModifier) ([]byte, int, error) {
	var req *http.Request
	req, errRequest := http.NewRequest(method, c.config.Host+path, body)
	if errRequest != nil {
		return nil, 0, errRequest
	}

	req.Header.Set("Content-Type", "application/json")
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

	resp, err := c.HTTPClient.Do(req)
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
