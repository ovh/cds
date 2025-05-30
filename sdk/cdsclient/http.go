package cdsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/telemetry"

	jwt "github.com/golang-jwt/jwt/v4"

	"github.com/ovh/cds/sdk"
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

// WithQueryParameter add query parameters to your http.Request
func WithQueryParameter(key, value string) RequestModifier {
	return func(req *http.Request) {
		q := req.URL.Query()
		q.Set(key, value)
		req.URL.RawQuery = q.Encode()
	}
}

// PostJSON post the *in* struct as json. If set, it unmarshalls the response to *out*
func (c *client) PostJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(ctx, http.MethodPost, path, in, out, mods...)
	return code, err
}

// PutJSON ut the *in* struct as json. If set, it unmarshalls the response to *out*
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
			return nil, nil, 0, newError(err)
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

	if code == 204 {
		return res, header, code, nil
	}

	if out != nil {
		if err := sdk.JSONUnmarshal(res, out); err != nil {
			return res, nil, code, newError(err)
		}
	}
	return res, header, code, nil
}

// Request executes an authentificated HTTP request on $path given $method and $args
func (c *client) Request(ctx context.Context, method string, path string, body io.Reader, mods ...RequestModifier) ([]byte, http.Header, int, error) {
	respBody, respHeader, code, err := c.Stream(ctx, c.httpClient, method, path, body, mods...)
	if err != nil {
		return nil, nil, code, err
	}
	defer func() {
		// Drain and close the body to let the Transport reuse the connection
		_, _ = io.Copy(io.Discard, respBody)
		_ = respBody.Close()
	}()

	var bodyBtes []byte
	bodyBtes, err = io.ReadAll(respBody)
	if err != nil {
		return nil, nil, code, newTransportError(err)
	}

	if c.config.Verbose {
		if len(bodyBtes) > 0 {
			log.Printf("Response Body: %s\n", bodyBtes)
		}
	}

	if code >= 400 {
		if err := sdk.DecodeError(bodyBtes); err != nil {
			return bodyBtes, nil, code, newAPIError(err)
		}
		return bodyBtes, nil, code, newAPIError(fmt.Errorf("HTTP %d", code))
	}
	return bodyBtes, respHeader, code, nil
}

// signin route pattern

var signinRouteRegexp = regexp.MustCompile(`\/auth\/consumer\/.*\/signin`)

func extractBodyErrorFromResponse(r *http.Response) error {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close() // nolint
	if err := sdk.DecodeError(body); err != nil {
		return newAPIError(err)
	}
	return newAPIError(fmt.Errorf("HTTP %d", r.StatusCode))
}

func (c *client) StreamNoRetry(ctx context.Context, httpClient HTTPClient, method string, path string, body io.Reader, mods ...RequestModifier) (io.ReadCloser, http.Header, int, error) {
	// Checks that current session_token is still valid
	// If not, challenge a new one against the authenticationToken
	var checkToken = !strings.Contains(path, "/auth/consumer/builtin/signin") &&
		!strings.Contains(path, "/auth/consumer/local/signin") &&
		!strings.Contains(path, "/auth/consumer/local/signup") &&
		!strings.Contains(path, "/auth/consumer/local/verify") &&
		!strings.Contains(path, "/auth/consumer/worker/signin") &&
		!strings.Contains(path, "/v2/auth/consumer/hatchery/signin")

	if checkToken && !c.config.HasValidSessionToken() && c.config.BuiltinConsumerAuthenticationToken != "" {
		if c.config.Verbose {
			log.Printf("session token invalid: (%s). Relogin...\n", c.config.SessionToken)
		}
		var req interface{}
		if c.signinRequest != nil {
			req = c.signinRequest
		} else {
			switch c.GetConsumerType() {
			case sdk.ConsumerHatchery:
				req = sdk.AuthConsumerHatcherySigninRequest{
					Token: c.config.BuiltinConsumerAuthenticationToken,
				}
			default:
				req = sdk.AuthConsumerSigninRequest{"token": c.config.BuiltinConsumerAuthenticationToken}
			}
		}
		resp, err := c.AuthConsumerSignin(c.GetConsumerType(), req)
		if err != nil {
			return nil, nil, -1, err
		}
		if c.config.Verbose {
			log.Println("jwt: ", sdk.StringFirstN(resp.Token, 12))
		}
		c.config.SessionToken = resp.Token
	}
	labels := pprof.Labels("path", path, "method", method)
	ctx = pprof.WithLabels(ctx, labels)
	pprof.SetGoroutineLabels(ctx)

	var url string
	if strings.HasPrefix(path, "http") {
		url = path
	} else {
		url = c.config.Host + path
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, nil, 0, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to create request: %v", err)
	}
	req = req.WithContext(ctx)
	date := sdk.FormatDateRFC5322(time.Now())
	req.Header.Set("Date", date)
	req.Header.Set("X-CDS-RemoteTime", date)

	if c.config.Verbose {
		log.Printf("Stream > context> %s\n", telemetry.DumpContext(ctx))
	}
	spanCtx, ok := telemetry.ContextToSpanContext(ctx)
	if ok {
		telemetry.DefaultFormat.SpanContextToRequest(spanCtx, req)
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

	//No auth on signing routes or on url that is not cds configured in config.Host
	if strings.HasPrefix(url, c.config.Host) && !signinRouteRegexp.MatchString(path) {
		if _, _, err := new(jwt.Parser).ParseUnverified(c.config.SessionToken, &sdk.AuthSessionJWTClaims{}); err == nil {
			if c.config.Verbose {
				log.Println("JWT recognized")
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, 500, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to execute request: %v", err)
	}
	if c.config.Verbose {
		log.Println(cli.Yellow("********RESPONSE**********"))
		dmp, _ := httputil.DumpResponse(resp, true)
		log.Printf("%s", string(dmp))
		log.Println(cli.Yellow("**************************"))
	}

	if resp.StatusCode == 401 {
		c.config.SessionToken = ""
	}

	if resp.StatusCode >= 400 {
		err := extractBodyErrorFromResponse(resp)
		return nil, nil, resp.StatusCode, err
	}
	return resp.Body, resp.Header, resp.StatusCode, nil
}

// Stream makes an authenticated http request and return io.ReadCloser
func (c *client) Stream(ctx context.Context, httpClient HTTPClient, method string, path string, body io.Reader, mods ...RequestModifier) (io.ReadCloser, http.Header, int, error) {
	// Checks that current session_token is still valid
	// If not, challenge a new one against the authenticationToken
	var checkToken = !strings.Contains(path, "/auth/consumer/builtin/signin") &&
		!strings.Contains(path, "/auth/consumer/local/signin") &&
		!strings.Contains(path, "/auth/consumer/local/signup") &&
		!strings.Contains(path, "/auth/consumer/local/verify") &&
		!strings.Contains(path, "/auth/consumer/worker/signin") &&
		!strings.Contains(path, "/v2/auth/consumer/hatchery/signin")

	if checkToken && !c.config.HasValidSessionToken() && c.config.BuiltinConsumerAuthenticationToken != "" {
		if c.config.Verbose {
			log.Printf("session token invalid: (%s). Relogin...\n", c.config.SessionToken)
		}
		switch c.GetConsumerType() {
		case sdk.ConsumerHatchery:
			var req interface{}
			if c.signinRequest != nil {
				req = c.signinRequest
			} else {
				req = sdk.AuthConsumerHatcherySigninRequest{
					Token: c.config.BuiltinConsumerAuthenticationToken,
				}
			}
			resp, err := c.AuthConsumerHatcherySigninV2(req)
			if err != nil {
				return nil, nil, -1, err
			}
			if c.config.Verbose {
				log.Println("jwt: ", sdk.StringFirstN(resp.Token, 12))
			}
			c.config.SessionToken = resp.Token
		default:
			var req interface{}
			if c.signinRequest != nil {
				req = c.signinRequest
			} else {
				req = sdk.AuthConsumerSigninRequest{"token": c.config.BuiltinConsumerAuthenticationToken}
			}
			resp, err := c.AuthConsumerSignin(c.GetConsumerType(), req)
			if err != nil {
				return nil, nil, -1, err
			}
			if c.config.Verbose {
				log.Println("jwt: ", sdk.StringFirstN(resp.Token, 12))
			}
			c.config.SessionToken = resp.Token
		}

	}

	labels := pprof.Labels("path", path, "method", method)
	ctx = pprof.WithLabels(ctx, labels)
	pprof.SetGoroutineLabels(ctx)

	// In case where the given reader is not a ReadSeeker we should store the body in ram to retry http request
	var bodyBytes []byte
	var err error
	if _, ok := body.(io.ReadSeeker); !ok && body != nil {
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, nil, 0, newTransportError(err)
		}
	}

	var url string
	if strings.HasPrefix(path, "http") {
		url = path
	} else {
		url = c.config.Host + path
	}

	var savederror error
	var savedCodeError int
	for i := 0; i <= c.config.Retry; i++ {
		var req *http.Request
		if rs, ok := body.(io.ReadSeeker); ok {
			if _, err := rs.Seek(0, 0); err != nil {
				return nil, nil, 0, newError(fmt.Errorf("request failed after %d retries: %v. Original error: %v", i, err, savederror))
			}
			req, err = http.NewRequest(method, url, body)
			if err != nil {
				return nil, nil, 0, newError(err)
			}
		} else {
			req, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
			if err != nil {
				return nil, nil, 0, newError(err)
			}
		}

		req = req.WithContext(ctx)
		date := sdk.FormatDateRFC5322(time.Now())
		req.Header.Set("Date", date)
		req.Header.Set("X-CDS-RemoteTime", date)

		if c.config.Verbose {
			log.Printf("Stream > context> %s\n", telemetry.DumpContext(ctx))
		}
		spanCtx, ok := telemetry.ContextToSpanContext(ctx)
		if ok {
			telemetry.DefaultFormat.SpanContextToRequest(spanCtx, req)
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

		//No auth on signing routes or on url that is not cds configured in config.Host
		if strings.HasPrefix(url, c.config.Host) && !signinRouteRegexp.MatchString(path) {
			if _, _, err := new(jwt.Parser).ParseUnverified(c.config.SessionToken, &sdk.AuthSessionJWTClaims{}); err == nil {
				if c.config.Verbose {
					log.Println("JWT recognized")
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

		resp, err := httpClient.Do(req)
		if err != nil {
			savederror = newTransportError(err)
			continue
		}

		savedCodeError = resp.StatusCode

		if c.config.Verbose {
			log.Println(cli.Yellow("********RESPONSE**********"))
			dmp, _ := httputil.DumpResponse(resp, true)
			log.Printf("%s", string(dmp))
			log.Println(cli.Yellow("**************************"))
		}

		if resp.StatusCode == 401 {
			c.config.SessionToken = ""
		}

		if resp.StatusCode == 409 || resp.StatusCode >= 500 {
			time.Sleep(250 * time.Millisecond)
			savederror = extractBodyErrorFromResponse(resp)
			continue
		}
		return resp.Body, resp.Header, resp.StatusCode, nil
	}

	if savedCodeError == 409 {
		return nil, nil, savedCodeError, savederror
	}
	return nil, nil, savedCodeError, newError(fmt.Errorf("request failed after %d retries: %v", c.config.Retry, savederror))
}

// UploadMultiPart upload multipart
func (c *client) UploadMultiPart(method string, path string, body *bytes.Buffer, mods ...RequestModifier) ([]byte, int, error) {
	// Checks that current session_token is still valid
	// If not, challenge a new one against the authenticationToken
	if !c.config.HasValidSessionToken() && c.config.BuiltinConsumerAuthenticationToken != "" {
		var req interface{}
		if c.signinRequest != nil {
			req = c.signinRequest
		} else {
			switch c.GetConsumerType() {
			case sdk.ConsumerHatchery:
				req = sdk.AuthConsumerHatcherySigninRequest{
					Token: c.config.BuiltinConsumerAuthenticationToken,
				}
			default:
				req = sdk.AuthConsumerSigninRequest{"token": c.config.BuiltinConsumerAuthenticationToken}
			}
		}
		resp, err := c.AuthConsumerSignin(c.GetConsumerType(), req)
		if err != nil {
			return nil, -1, err
		}
		c.config.SessionToken = resp.Token
	}

	var req *http.Request
	req, errRequest := http.NewRequest(method, c.config.Host+path, body)
	if errRequest != nil {
		return nil, 0, newError(errRequest)
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

	resp, err := c.HTTPNoTimeoutClient().Do(req)
	if err != nil {
		return nil, 0, newTransportError(err)
	}
	defer resp.Body.Close()

	if c.config.Verbose {
		fmt.Printf("Response Status: %s\n", resp.Status)
		fmt.Printf("Request path: %s\n", c.config.Host+path)
		fmt.Printf("Request Headers: %s\n", req.Header)
		fmt.Printf("Response Headers: %s\n", resp.Header)
	}

	if resp.StatusCode == 401 {
		c.config.SessionToken = ""
	}

	var respBody []byte
	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, newTransportError(err)
	}

	if c.config.Verbose {
		if len(body.Bytes()) > 0 {
			fmt.Printf("Response Body: %s\n", body.String())
		}
	}

	return respBody, resp.StatusCode, nil
}
