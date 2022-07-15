package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"gopkg.in/spacemonkeygo/httpsig.v0"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

// MultiPartData represents the data to send
type MultiPartData struct {
	Reader      io.Reader
	ContentType string
}

// HTTPClient will be set to a default httpclient if not set
var HTTPClient cdsclient.HTTPClient

// HTTPSigner is used to sign requests based on the RFC draft specification https://tools.ietf.org/html/draft-cavage-http-signatures-06
var HTTPSigner *httpsig.Signer

type Client interface {
	// DoJSONRequest performs an http request on a service
	DoJSONRequest(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (http.Header, int, error)
	// DoMultiPartRequest performs an http request on a service with multipart  tar file + json field
	DoMultiPartRequest(ctx context.Context, method, path string, multiPartData *MultiPartData, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (int, error)
	// StreamRequest performs an hhtp request and stream the response
	StreamRequest(ctx context.Context, method, path string, in interface{}, mods ...cdsclient.RequestModifier) (io.Reader, http.Header, int, error)
}

type defaultServiceClient struct {
	db   gorp.SqlExecutor
	srvs []sdk.Service
}

var NewClient func(gorp.SqlExecutor, []sdk.Service) Client = NewDefaultClient

func NewDefaultClient(db gorp.SqlExecutor, srvs []sdk.Service) Client {
	return &defaultServiceClient{
		db:   db,
		srvs: srvs,
	}
}

func (s *defaultServiceClient) StreamRequest(ctx context.Context, method, path string, in interface{}, mods ...cdsclient.RequestModifier) (io.Reader, http.Header, int, error) {
	ctx, end := telemetry.Span(ctx, "services.DoJSONRequest",
		telemetry.Tag("http.method", method),
		telemetry.Tag("http.path", path),
	)
	end()
	return doStreamRequest(ctx, s.srvs, method, path, in, mods...)
}

// doStreamRequest performs an http request on a service
func doStreamRequest(ctx context.Context, srvs []sdk.Service, method, path string, in interface{}, mods ...cdsclient.RequestModifier) (io.Reader, http.Header, int, error) {
	var b = []byte{}
	var err error
	if in != nil {
		b, err = json.Marshal(in)
		if err != nil {
			return nil, nil, 0, sdk.WrapError(err, "Unable to marshal input")
		}
	}

	var lastErr = sdk.WithStack(errors.New("unable to call service: service not found"))
	var lastCode int
	var attempts int64
	for attempts = 0; attempts < 5; attempts++ {
		for i := range srvs {
			srv := &srvs[i]
			reader, headers, code, err := doStreamFromURL(ctx, srv, method, path, bytes.NewReader(b), mods...)
			if err != nil {
				lastErr = err
				lastCode = code
				continue
			}
			return reader, headers, code, nil
		}
		if lastCode < 409 {
			break
		}
	}

	log.Error(ctx, "unable to call service: maximum attempt exceed: %+v lastCode:%d attempts:%d", lastErr, lastCode, attempts)
	return nil, nil, lastCode, sdk.WithStack(lastErr)
}

func (s *defaultServiceClient) DoMultiPartRequest(ctx context.Context, method, path string, multiPartData *MultiPartData, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (int, error) {
	return doMultiPartRequest(ctx, s.srvs, method, path, multiPartData, in, out, mods...)
}

// doMultiPartRequest performs an http request on a service with multipart  tar file + json field
func doMultiPartRequest(ctx context.Context, srvs []sdk.Service, method, path string, multiPartData *MultiPartData, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (int, error) {
	ctx, end := telemetry.Span(ctx, "services.doMultiPartRequest",
		telemetry.Tag("http.method", method),
		telemetry.Tag("http.path", path),
	)
	defer end()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create tar part
	dataFileHeader := make(textproto.MIMEHeader)
	dataFileHeader.Set("Content-Type", multiPartData.ContentType)
	dataFileHeader.Set("Content-Disposition", "form-data; name=\"dataFiles\"; filename=\"data\"")
	dataPart, err := writer.CreatePart(dataFileHeader)
	if err != nil {
		return 0, sdk.WrapError(err, "unable to create data part")
	}
	if _, err := io.Copy(dataPart, multiPartData.Reader); err != nil {
		return 0, sdk.WrapError(err, "unable to write into data part")
	}

	jsonData, errM := json.Marshal(in)
	if errM != nil {
		return 0, sdk.WrapError(errM, "unable to marshal data")
	}
	if err := writer.WriteField("dataJSON", string(jsonData)); err != nil {
		return 0, sdk.WrapError(err, "unable to add field dataJSON")
	}

	// Close writer
	if err := writer.Close(); err != nil {
		return 0, sdk.WrapError(err, "unable to close writer")
	}

	mods = append(mods, cdsclient.SetHeader("Content-Type", writer.FormDataContentType()))
	var lastErr error
	var lastCode int
	var attempt int

	bodyToSend := body.Bytes()
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			res, _, code, err := doRequest(ctx, srv, method, path, bodyToSend, mods...)
			if err != nil {
				lastErr = err
				lastCode = code
				continue
			}

			if out != nil {
				if err := sdk.JSONUnmarshal(res, out); err != nil {
					return code, sdk.WrapError(err, "Unable to marshal output")
				}
			}
			return code, nil

		}
		if lastErr != nil || attempt > 5 {
			break
		}
	}
	return lastCode, lastErr
}

func (s *defaultServiceClient) DoJSONRequest(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (http.Header, int, error) {
	ctx, end := telemetry.Span(ctx, "services.DoJSONRequest",
		telemetry.Tag("http.method", method),
		telemetry.Tag("http.path", path),
	)
	defer end()

	return doJSONRequest(ctx, s.srvs, method, path, in, out, mods...)
}

// doJSONRequest performs an http request on a service
func doJSONRequest(ctx context.Context, srvs []sdk.Service, method, path string, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (http.Header, int, error) {
	var lastErr = sdk.WithStack(errors.New("unable to call service: service not found"))
	var lastCode int
	var attempts int64
	for attempts = 0; attempts < 5; attempts++ {
		for i := range srvs {
			srv := &srvs[i]
			headers, code, err := _doJSONRequest(ctx, srv, method, path, in, out, mods...)
			if err != nil {
				lastErr = err
				lastCode = code
				continue
			}
			return headers, code, nil
		}
		if lastCode < 409 {
			break
		}
	}

	log.Error(ctx, "unable to call service: maximum attempt exceed: %+v lastCode:%d attempts:%d", lastErr, lastCode, attempts)
	return nil, lastCode, sdk.WithStack(lastErr)
}

// _doJSONRequest is a low level function that performs an http request on service
func _doJSONRequest(ctx context.Context, srv *sdk.Service, method, path string, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (http.Header, int, error) {
	var b = []byte{}
	var err error

	if in != nil {
		b, err = json.Marshal(in)
		if err != nil {
			return nil, 0, sdk.WrapError(err, "Unable to marshal input")
		}
	}

	mods = append(mods, cdsclient.SetHeader("Content-Type", "application/json"))
	res, headers, code, err := doRequest(ctx, srv, method, path, b, mods...)
	if err != nil {
		return headers, code, sdk.ErrorWithFallback(err, sdk.ErrUnknownError, "unable to perform request on service %s (%s)", srv.Name, srv.Type)
	}

	if out != nil {
		if err := sdk.JSONUnmarshal(res, out); err != nil {
			return headers, code, sdk.WrapError(err, "unable to unmarshal output")
		}
	}

	return headers, code, nil
}

// PostBinary
func PostBinary(ctx context.Context, srvs []sdk.Service, path string, r io.Reader, out interface{}, mods ...cdsclient.RequestModifier) (int, error) {
	// No retry because if http call failed, reader is closed
	seed := rand.NewSource(time.Now().UnixNano())
	ra := rand.New(seed)
	srv := &srvs[ra.Intn(len(srvs))]
	callURL, err := url.ParseRequestURI(srv.HTTPURL + path)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	res, _, code, err := doRequestFromURL(ctx, "POST", callURL, r, mods...)
	if err != nil {
		return code, err
	}
	if out != nil {
		if err := sdk.JSONUnmarshal(res, out); err != nil {
			return code, sdk.WithStack(err)
		}
	}
	return code, nil
}

// DoRequest performs an http request on a service
func DoRequest(ctx context.Context, srvs []sdk.Service, method, path string, args []byte, mods ...cdsclient.RequestModifier) ([]byte, http.Header, int, error) {
	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			btes, headers, code, err := doRequest(ctx, srv, method, path, args, mods...)
			if err == nil {
				return btes, headers, code, nil
			}
			lastErr = err
			lastCode = code
		}
		if lastErr != nil || attempt > 5 {
			break
		}
	}
	return nil, nil, lastCode, lastErr
}

// doRequest performs an http request on service
func doRequest(ctx context.Context, srv *sdk.Service, method, path string, args []byte, mods ...cdsclient.RequestModifier) ([]byte, http.Header, int, error) {
	callURL, err := url.ParseRequestURI(srv.HTTPURL + path)
	if err != nil {
		return nil, nil, 0, sdk.WithStack(err)
	}
	var r io.Reader
	if args != nil {
		r = bytes.NewReader(args)
	}
	return doRequestFromURL(ctx, method, callURL, r, mods...)
}

func doStreamFromURL(ctx context.Context, srv *sdk.Service, method string, path string, reader io.Reader, mods ...cdsclient.RequestModifier) (io.Reader, http.Header, int, error) {
	callURL, err := url.ParseRequestURI(srv.HTTPURL + path)
	if err != nil {
		return nil, nil, 0, sdk.WithStack(err)
	}

	if HTTPClient == nil {
		HTTPClient = cdsclient.NewHTTPClient(60*time.Second, false)
	}

	if HTTPSigner == nil {
		HTTPSigner = httpsig.NewRSASHA256Signer(authentication.GetIssuerName(), authentication.GetSigningKey(), []string{"(request-target)", "host", "date"})
	}

	req, requestError := http.NewRequest(method, callURL.String(), reader)
	if requestError != nil {
		return nil, nil, 0, sdk.WithStack(requestError)
	}

	req = req.WithContext(ctx)

	spanCtx, ok := telemetry.ContextToSpanContext(ctx)
	if ok {
		telemetry.DefaultFormat.SpanContextToRequest(spanCtx, req)
	}

	req.Header.Set("Connection", "close")
	for i := range mods {
		if mods[i] != nil {
			mods[i](req)
		}
	}

	requestID := cdslog.ContextValue(ctx, cdslog.RequestID)
	if requestID != "" {
		req.Header.Set(cdslog.HeaderRequestID, requestID)
	}

	// If body is not *bytes.Buffer, *bytes.Reader or *strings.Reader Content-Length is not set. (
	// Here we force Content-Length.
	// cf net/http/request.go  NewRequestWithContext
	if req.Header.Get("Content-Length") != "" {
		s, err := strconv.Atoi(req.Header.Get("Content-Length"))
		if err != nil {
			return nil, nil, 0, sdk.WithStack(err)
		}
		req.ContentLength = int64(s)
	}

	// Sign the http request with API private RSA Key
	if err := HTTPSigner.Sign(req); err != nil {
		return nil, nil, 0, sdk.WrapError(err, "request signature failed")
	}

	log.Debug(ctx, "services.DoRequest> request %s %v (%s)", req.Method, req.URL, req.Header.Get("Authorization"))

	//Do the request
	resp, err := HTTPClient.Do(req)
	if err != nil {
		if resp != nil && resp.StatusCode > 0 {
			return nil, nil, resp.StatusCode, sdk.WrapError(err, "request failed with resp status code: %d", resp.StatusCode)
		}
		return nil, nil, 500, sdk.WrapError(err, "request failed - use code 500")
	}

	// if everything is fine, return body
	if resp.StatusCode < 400 {
		return resp.Body, resp.Header, resp.StatusCode, nil
	}

	defer resp.Body.Close()
	
	// Try to catch the CDS Error
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, resp.StatusCode, sdk.WrapError(err, "unable to read body")
	}
	if cdserr := sdk.DecodeError(body); cdserr != nil {
		return nil, resp.Header, resp.StatusCode, cdserr
	}
	return nil, resp.Header, resp.StatusCode, sdk.WithStack(fmt.Errorf("request failed with status code: %d", resp.StatusCode))
}

func doRequestFromURL(ctx context.Context, method string, callURL *url.URL, reader io.Reader, mods ...cdsclient.RequestModifier) ([]byte, http.Header, int, error) {
	if HTTPClient == nil {
		HTTPClient = cdsclient.NewHTTPClient(60*time.Second, false)
	}

	if HTTPSigner == nil {
		HTTPSigner = httpsig.NewRSASHA256Signer(authentication.GetIssuerName(), authentication.GetSigningKey(), []string{"(request-target)", "host", "date"})
	}

	req, requestError := http.NewRequest(method, callURL.String(), reader)
	if requestError != nil {
		return nil, nil, 0, sdk.WithStack(requestError)
	}

	req = req.WithContext(ctx)

	spanCtx, ok := telemetry.ContextToSpanContext(ctx)
	if ok {
		telemetry.DefaultFormat.SpanContextToRequest(spanCtx, req)
	}

	req.Header.Set("Connection", "close")
	for i := range mods {
		if mods[i] != nil {
			mods[i](req)
		}
	}

	requestID := cdslog.ContextValue(ctx, cdslog.RequestID)
	if requestID != "" {
		req.Header.Set(cdslog.HeaderRequestID, requestID)
	}

	// If body is not *bytes.Buffer, *bytes.Reader or *strings.Reader Content-Length is not set. (
	// Here we force Content-Length.
	// cf net/http/request.go  NewRequestWithContext
	if req.Header.Get("Content-Length") != "" {
		s, err := strconv.Atoi(req.Header.Get("Content-Length"))
		if err != nil {
			return nil, nil, 0, sdk.WithStack(err)
		}
		req.ContentLength = int64(s)
	}

	// Sign the http request with API private RSA Key
	if err := HTTPSigner.Sign(req); err != nil {
		return nil, nil, 0, sdk.WrapError(err, "request signature failed")
	}

	log.Debug(ctx, "services.DoRequest> request %s %v (%s)", req.Method, req.URL, req.Header.Get("Authorization"))

	//Do the request
	resp, err := HTTPClient.Do(req)
	if err != nil {
		if resp != nil && resp.StatusCode > 0 {
			return nil, nil, resp.StatusCode, sdk.WrapError(err, "request failed with resp status code: %d", resp.StatusCode)
		}
		return nil, nil, 500, sdk.WrapError(err, "request failed - use code 500")
	}
	defer resp.Body.Close()

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, resp.StatusCode, sdk.WrapError(err, "unable to read body")
	}

	log.Debug(ctx, "services.DoRequest> response code:%d", resp.StatusCode)

	// if everything is fine, return body
	if resp.StatusCode < 400 {
		return body, resp.Header, resp.StatusCode, nil
	}

	// Try to catch the CDS Error
	if cdserr := sdk.DecodeError(body); cdserr != nil {
		return nil, resp.Header, resp.StatusCode, cdserr
	}

	return nil, resp.Header, resp.StatusCode, sdk.WithStack(fmt.Errorf("request failed with status code: %d", resp.StatusCode))
}
