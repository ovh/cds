package cdsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/ovh/cds/cli"

	"github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/tracingutils"
)

const (
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

// SetHeader modify headers of http.Request
func SetHeader(key, value string) RequestModifier {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// PostJSON post the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) PostJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(ctx, http.MethodPost, path, in, out, mods...)
	return code, err
}

// PostJSON ut the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) PutJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(ctx, http.MethodPut, path, in, out, mods...)
	return code, err
}

// GetJSON get the requested path If set, it unmarshalls the response to *out*
func (c *client) GetJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(ctx, http.MethodGet, path, nil, out, mods...)
	return code, err
}

// GetJSONWithHeaders get the requested path If set, it unmarshalls the response to *out* and return response headers
func (c *client) GetJSONWithHeaders(path string, out interface{}, mods ...RequestModifier) (http.Header, int, error) {
	_, header, code, err := c.RequestJSON(context.Background(), http.MethodGet, path, nil, out, mods...)
	return header, code, err
}

// DeleteJSON deletes the requested path If set, it unmarshalls the response to *out*
func (c *client) DeleteJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(ctx, http.MethodDelete, path, nil, out, mods...)
	return code, err
}

// RequestJSON does a request with the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) RequestJSON(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...RequestModifier) ([]byte, http.Header, int, error) {
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

	res, header, code, err := c.Request(ctx, method, path, body, mods...)
	if err != nil {
		return nil, nil, code, err
	}

	if code >= 400 {
		if err := sdk.DecodeError(res); err != nil {
			return res, nil, code, err
		}
		return res, nil, code, fmt.Errorf("HTTP %d", code)
	}

	if out != nil {
		if err := json.Unmarshal(res, out); err != nil {
			return res, nil, code, err
		}
	}

	return res, header, code, nil
}

// Request executes an authentificated HTTP request on $path given $method and $args
func (c *client) Request(ctx context.Context, method string, path string, body io.Reader, mods ...RequestModifier) ([]byte, http.Header, int, error) {
	respBody, respHeader, code, err := c.Stream(ctx, method, path, body, false, mods...)
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

	if code >= 400 {
		if err := sdk.DecodeError(bodyBtes); err != nil {
			return bodyBtes, nil, code, err
		}
		return bodyBtes, nil, code, fmt.Errorf("HTTP %d", code)
	}

	return bodyBtes, respHeader, code, nil
}

// signin route pattern

var signinRouteRegexp = regexp.MustCompile(`\/auth\/consumer\/.*\/signin`)

// Stream makes an authenticated http request and return io.ReadCloser
func (c *client) Stream(ctx context.Context, method string, path string, body io.Reader, noTimeout bool, mods ...RequestModifier) (io.ReadCloser, http.Header, int, error) {
	// Checks that current session_token is still valid
	// If not, challenge a new one against the authenticationToken
	if path != "/auth/consumer/builtin/signin" && !c.config.HasValidSessionToken() && c.config.BuitinConsumerAuthenticationToken != "" {
		if c.config.Verbose {
			fmt.Printf("session token invalid: (%s). Relogin...\n", c.config.SessionToken)
		}
		resp, err := c.AuthConsumerSignin(sdk.ConsumerBuiltin, sdk.AuthConsumerSigninRequest{"token": c.config.BuitinConsumerAuthenticationToken})
		if err != nil {
			return nil, nil, -1, err
		}
		if c.config.Verbose {
			fmt.Println("jwt: ", resp.Token[:12])
		}
		c.config.SessionToken = resp.Token
	}

	labels := pprof.Labels("path", path, "method", method)
	ctx = pprof.WithLabels(ctx, labels)
	pprof.SetGoroutineLabels(ctx)
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

		req = req.WithContext(ctx)
		date := sdk.FormatDateRFC5322(time.Now())
		req.Header.Set("Date", date)
		req.Header.Set("X-CDS-RemoteTime", date)

		if c.config.Verbose {
			log.Printf("Stream > context> %s\n", tracingutils.DumpContext(ctx))
		}
		spanCtx, ok := tracingutils.ContextToSpanContext(ctx)
		if ok {
			tracingutils.DefaultFormat.SpanContextToRequest(spanCtx, req)
		}

		for i := range mods {
			if mods[i] != nil {
				mods[i](req)
			}
		}

		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		req.Header.Set("Connection", "close")

		//No auth on signing routes
		if !signinRouteRegexp.MatchString(path) {
			if _, _, err := new(jwt.Parser).ParseUnverified(c.config.SessionToken, &sdk.AuthSessionJWTClaims{}); err == nil {
				if c.config.Verbose {
					fmt.Println("JWT recognized")
				}
				auth := "Bearer " + c.config.SessionToken
				req.Header.Add("Authorization", auth)
			}
		}

		if c.config.Verbose {
			log.Println(cli.Green("********REQUEST**********"))
			dmp, _ := httputil.DumpRequestOut(req, true)
			log.Printf("%s", string(dmp))
			log.Println(cli.Green("**************************"))

		}

		var errDo error
		var resp *http.Response
		if noTimeout {
			resp, errDo = c.httpSSEClient.Do(req)
		} else {
			resp, errDo = c.httpClient.Do(req)
		}

		if errDo == nil && c.config.Verbose {
			log.Println(cli.Yellow("********RESPONSE**********"))
			dmp, _ := httputil.DumpResponse(resp, true)
			log.Printf("%s", string(dmp))
			log.Println(cli.Yellow("**************************"))
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
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		if errDo != nil {
			return nil, nil, 0, errDo
		}
	}

	return nil, nil, 0, fmt.Errorf("x%d: %s", c.config.Retry, savederror)
}

// UploadMultiPart upload multipart
func (c *client) UploadMultiPart(method string, path string, body *bytes.Buffer, mods ...RequestModifier) ([]byte, int, error) {
	// Checks that current session_token is still valid
	// If not, challenge a new one against the authenticationToken
	if !c.config.HasValidSessionToken() && c.config.BuitinConsumerAuthenticationToken != "" {
		resp, err := c.AuthConsumerSignin(sdk.ConsumerBuiltin, sdk.AuthConsumerSigninRequest{"token": c.config.BuitinConsumerAuthenticationToken})
		if err != nil {
			return nil, -1, err
		}
		c.config.SessionToken = resp.Token
	}

	var req *http.Request
	req, errRequest := http.NewRequest(method, c.config.Host+path, body)
	if errRequest != nil {
		return nil, 0, errRequest
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Connection", "close")

	for i := range mods {
		mods[i](req)
	}

	//No auth on signing routes
	if !signinRouteRegexp.MatchString(path) {
		if _, _, err := new(jwt.Parser).ParseUnverified(c.config.SessionToken, &sdk.AuthSessionJWTClaims{}); err == nil {
			if c.config.Verbose {
				fmt.Println("JWT recognized")
			}
			auth := "Bearer " + c.config.SessionToken
			req.Header.Add("Authorization", auth)
		}
	}

	resp, err := c.httpSSEClient.Do(req)
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
