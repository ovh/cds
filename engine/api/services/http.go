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

	"github.com/ovh/cds/engine/api/accesstoken"

	"gopkg.in/spacemonkeygo/httpsig.v0"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/tracingutils"
)

// MultiPartData represents the data to send
type MultiPartData struct {
	Reader      io.Reader
	ContentType string
}

// HTTPClient will be set to a default httpclient if not set
var HTTPClient sdk.HTTPClient

// HTTPSigner is used to sign requests based on the RFC draft specification https://tools.ietf.org/html/draft-cavage-http-signatures-06
var HTTPSigner *httpsig.Signer

// DoMultiPartRequest performs an http request on a service with multipart  tar file + json field
func DoMultiPartRequest(ctx context.Context, db gorp.SqlExecutor, srvs []sdk.Service, method, path string, multiPartData *MultiPartData, in interface{}, out interface{}, mods ...sdk.RequestModifier) (int, error) {
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

	mods = append(mods, sdk.SetHeader("Content-Type", writer.FormDataContentType()))
	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			res, code, err := doRequest(ctx, db, srv, method, path, body.Bytes(), mods...)
			if err != nil {
				return code, sdk.WrapError(err, "Unable to perform request on service %s (%s)", srv.Name, srv.Type)
			}
			if out != nil {
				if err := json.Unmarshal(res, out); err != nil {
					return code, sdk.WrapError(err, "Unable to marshal output")
				}
			}
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

// DoJSONRequest performs an http request on a service
func DoJSONRequest(ctx context.Context, db gorp.SqlExecutor, srvs []sdk.Service, method, path string, in interface{}, out interface{}, mods ...sdk.RequestModifier) (int, error) {
	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			code, err := doJSONRequest(ctx, db, srv, method, path, in, out, mods...)
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
func doJSONRequest(ctx context.Context, db gorp.SqlExecutor, srv *sdk.Service, method, path string, in interface{}, out interface{}, mods ...sdk.RequestModifier) (int, error) {
	var b = []byte{}
	var err error

	if in != nil {
		b, err = json.Marshal(in)
		if err != nil {
			return 0, sdk.WrapError(err, "Unable to marshal input")
		}
	}

	mods = append(mods, sdk.SetHeader("Content-Type", "application/json"))
	res, code, err := doRequest(ctx, db, srv, method, path, b, mods...)
	if err != nil {
		return code, sdk.ErrorWithFallback(err, sdk.ErrUnknownError, "Unable to perform request on service %s (%s)", srv.Name, srv.Type)
	}

	if out != nil {
		if err := json.Unmarshal(res, out); err != nil {
			return code, sdk.WrapError(err, "Unable to marshal output")
		}
	}

	return code, nil
}

// PostMultipart post a file content through multipart upload
func PostMultipart(ctx context.Context, db gorp.SqlExecutor, srvs []sdk.Service, path string, filename string, fileContents []byte, out interface{}, mods ...sdk.RequestModifier) (int, error) {
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

	mods = append(mods, sdk.SetHeader("Content-Type", "multipart/form-data"))

	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			res, code, err := doRequest(ctx, db, srv, "POST", path, body.Bytes(), mods...)
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
func DoRequest(ctx context.Context, db gorp.SqlExecutor, srvs []sdk.Service, method, path string, args []byte, mods ...sdk.RequestModifier) ([]byte, int, error) {
	var lastErr error
	var lastCode int
	var attempt int
	for {
		attempt++
		for i := range srvs {
			srv := &srvs[i]
			btes, code, err := doRequest(ctx, db, srv, method, path, args, mods...)
			if err == nil {
				return btes, code, nil
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

// doRequest performs an http request on service
func doRequest(ctx context.Context, db gorp.SqlExecutor, srv *sdk.Service, method, path string, args []byte, mods ...sdk.RequestModifier) ([]byte, int, error) {
	if HTTPClient == nil {
		HTTPClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	if HTTPSigner == nil {
		HTTPSigner = httpsig.NewRSASHA256Signer(accesstoken.LocalIssuer, accesstoken.GetSigningKey(), []string{"(request-target)", "host", "date"})
	}

	callURL, err := url.ParseRequestURI(srv.HTTPURL + path)
	if err != nil {
		return nil, 0, err
	}

	var requestError error
	var req *http.Request
	if args != nil {
		req, requestError = http.NewRequest(method, callURL.String(), bytes.NewReader(args))
	} else {
		req, requestError = http.NewRequest(method, callURL.String(), nil)
	}
	if requestError != nil {
		return nil, 0, requestError
	}

	req = req.WithContext(ctx)

	spanCtx, ok := tracingutils.ContextToSpanContext(ctx)
	if ok {
		tracingutils.DefaultFormat.SpanContextToRequest(spanCtx, req)
	}

	req.Header.Set("Connection", "close")
	req.Header.Add(sdk.RequestedWithHeader, sdk.RequestedWithValue)
	for i := range mods {
		if mods[i] != nil {
			mods[i](req)
		}
	}

	// Sign the http request with API private RSA Key
	if err := HTTPSigner.Sign(req); err != nil {
		return nil, 0, sdk.WrapError(err, "services.DoRequest> Request signature failed")
	}

	log.Debug("services.DoRequest> request %v (%s)", req.URL, req.Header.Get("Authorization"))

	//Do the request
	resp, errDo := HTTPClient.Do(req)
	if errDo != nil {
		return nil, 0, sdk.WrapError(errDo, "services.DoRequest> Request failed")
	}
	defer resp.Body.Close()

	// Read the body
	body, errBody := ioutil.ReadAll(resp.Body)
	if errBody != nil {
		return nil, resp.StatusCode, sdk.WrapError(errBody, "services.DoRequest> Unable to read body")
	}

	log.Debug("services.DoRequest> response code:%d body:%s", resp.StatusCode, string(body))

	// if everything is fine, return body
	if resp.StatusCode < 400 {
		return body, resp.StatusCode, nil
	}

	// Try to catch the CDS Error
	if cdserr := sdk.DecodeError(body); cdserr != nil {
		return nil, resp.StatusCode, cdserr
	}

	return nil, resp.StatusCode, fmt.Errorf("Request Failed")
}
