package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"time"

	"github.com/go-gorp/gorp"
	"gopkg.in/spacemonkeygo/httpsig.v0"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/tracingutils"
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

func (s *defaultServiceClient) DoMultiPartRequest(ctx context.Context, method, path string, multiPartData *MultiPartData, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (int, error) {
	return doMultiPartRequest(ctx, s.db, s.srvs, method, path, multiPartData, in, out, mods...)
}

// doMultiPartRequest performs an http request on a service with multipart  tar file + json field
func doMultiPartRequest(ctx context.Context, db gorp.SqlExecutor, srvs []sdk.Service, method, path string, multiPartData *MultiPartData, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (int, error) {
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
			res, _, code, err := doRequest(ctx, db, srv, method, path, bodyToSend, mods...)
			if err != nil {
				lastErr = err
				lastCode = code
				continue
			}

			if out != nil {
				if err := json.Unmarshal(res, out); err != nil {
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
	return doJSONRequest(ctx, s.db, s.srvs, method, path, in, out, mods...)
}

// doJSONRequest performs an http request on a service
func doJSONRequest(ctx context.Context, db gorp.SqlExecutor, srvs []sdk.Service, method, path string, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (http.Header, int, error) {
	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			headers, code, err := _doJSONRequest(ctx, db, srv, method, path, in, out, mods...)
			if err == nil {
				return headers, code, nil
			}
			lastErr = err
			lastCode = code
		}
		if lastErr != nil || attempt > 5 {
			break
		}
	}
	return nil, lastCode, lastErr
}

// _doJSONRequest is a low level function that performs an http request on service
func _doJSONRequest(ctx context.Context, db gorp.SqlExecutor, srv *sdk.Service, method, path string, in interface{}, out interface{}, mods ...cdsclient.RequestModifier) (http.Header, int, error) {
	var b = []byte{}
	var err error

	if in != nil {
		b, err = json.Marshal(in)
		if err != nil {
			return nil, 0, sdk.WrapError(err, "Unable to marshal input")
		}
	}

	mods = append(mods, cdsclient.SetHeader("Content-Type", "application/json"))
	res, headers, code, err := doRequest(ctx, db, srv, method, path, b, mods...)
	if err != nil {
		return headers, code, sdk.ErrorWithFallback(err, sdk.ErrUnknownError, "Unable to perform request on service %s (%s)", srv.Name, srv.Type)
	}

	if out != nil {
		if err := json.Unmarshal(res, out); err != nil {
			return headers, code, sdk.WrapError(err, "Unable to marshal output")
		}
	}

	return headers, code, nil
}

// PostMultipart post a file content through multipart upload
func PostMultipart(ctx context.Context, db gorp.SqlExecutor, srvs []sdk.Service, path string, filename string, fileContents []byte, out interface{}, mods ...cdsclient.RequestModifier) (int, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return 0, err
	}
	part.Write(fileContents)
	if err := writer.Close(); err != nil {
		return 0, err
	}

	mods = append(mods, cdsclient.SetHeader("Content-Type", "multipart/form-data"))

	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			res, _, code, err := doRequest(ctx, db, srv, "POST", path, body.Bytes(), mods...)
			lastCode = code
			lastErr = err

			if err == nil {
				return code, nil
			}

			if out != nil {
				if err := json.Unmarshal(res, out); err != nil {
					return code, err
				}
			}
		}
		if lastErr == nil {
			break
		}
	}
	return lastCode, lastErr
}

// DoRequest performs an http request on a service
func DoRequest(ctx context.Context, db gorp.SqlExecutor, srvs []sdk.Service, method, path string, args []byte, mods ...cdsclient.RequestModifier) ([]byte, http.Header, int, error) {
	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			btes, headers, code, err := doRequest(ctx, db, srv, method, path, args, mods...)
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
func doRequest(ctx context.Context, db gorp.SqlExecutor, srv *sdk.Service, method, path string, args []byte, mods ...cdsclient.RequestModifier) ([]byte, http.Header, int, error) {
	callURL, err := url.ParseRequestURI(srv.HTTPURL + path)
	if err != nil {
		return nil, nil, 0, err
	}
	return doRequestFromURL(ctx, db, method, callURL, args, mods...)
}

func doRequestFromURL(ctx context.Context, db gorp.SqlExecutor, method string, callURL *url.URL, args []byte, mods ...cdsclient.RequestModifier) ([]byte, http.Header, int, error) {
	if HTTPClient == nil {
		HTTPClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	if HTTPSigner == nil {
		HTTPSigner = httpsig.NewRSASHA256Signer(authentication.IssuerName, authentication.GetSigningKey(), []string{"(request-target)", "host", "date"})
	}

	var requestError error
	var req *http.Request
	if args != nil {
		req, requestError = http.NewRequest(method, callURL.String(), bytes.NewReader(args))
	} else {
		req, requestError = http.NewRequest(method, callURL.String(), nil)
	}
	if requestError != nil {
		return nil, nil, 0, requestError
	}

	req = req.WithContext(ctx)

	spanCtx, ok := tracingutils.ContextToSpanContext(ctx)
	if ok {
		tracingutils.DefaultFormat.SpanContextToRequest(spanCtx, req)
	}

	req.Header.Set("Connection", "close")
	for i := range mods {
		if mods[i] != nil {
			mods[i](req)
		}
	}

	iRequestID := ctx.Value(log.ContextLoggingRequestIDKey)
	if iRequestID != nil {
		if requestID, ok := iRequestID.(string); ok {
			req.Header.Set(log.HeaderRequestID, requestID)
		}
	}

	// Sign the http request with API private RSA Key
	if err := HTTPSigner.Sign(req); err != nil {
		return nil, nil, 0, sdk.WrapError(err, "services.DoRequest> Request signature failed")
	}

	log.Debug("services.DoRequest> request %s %v (%s)", req.Method, req.URL, req.Header.Get("Authorization"))

	//Do the request
	resp, errDo := HTTPClient.Do(req)
	if errDo != nil {
		return nil, nil, 0, sdk.WrapError(errDo, "services.DoRequest> Request failed")
	}
	defer resp.Body.Close()

	// Read the body
	body, errBody := ioutil.ReadAll(resp.Body)
	if errBody != nil {
		return nil, resp.Header, resp.StatusCode, sdk.WrapError(errBody, "services.DoRequest> Unable to read body")
	}

	log.Debug("services.DoRequest> response code:%d", resp.StatusCode)

	// if everything is fine, return body
	if resp.StatusCode < 400 {
		return body, resp.Header, resp.StatusCode, nil
	}

	// Try to catch the CDS Error
	if cdserr := sdk.DecodeError(body); cdserr != nil {
		return nil, resp.Header, resp.StatusCode, cdserr
	}

	return nil, resp.Header, resp.StatusCode, fmt.Errorf("Request Failed")

}
