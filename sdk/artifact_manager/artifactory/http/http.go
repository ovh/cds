package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/rockbears/log"
)

// RequestModifier is used to modify behavior of Request and Steam functions
type RequestModifier func(req *http.Request)

// HTTPClient is a interface for HTTPClient mock
type HTTPClient interface {
	RequestJSON(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...RequestModifier) ([]byte, http.Header, int, error)
	PostJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	PutJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	GetJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error)
	DeleteJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error)
}

type client struct {
	Config struct {
		Host  string
		Token string
	}
	*http.Client
}

func NewClient(host, token string) HTTPClient {
	c := &client{Client: http.DefaultClient}
	c.Config.Host = host
	c.Config.Token = token
	return c
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

// DeleteJSON deletes the requested path If set, it unmarshalls the response to *out*
func (c *client) DeleteJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error) {
	_, _, code, err := c.RequestJSON(ctx, http.MethodDelete, path, nil, out, mods...)
	return code, err
}

type APIError struct {
	Errors []struct {
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"errors"`
}

func (e APIError) Error() error {
	var msg []string
	for _, m := range e.Errors {
		msg = append(msg, m.Message)
	}
	return errors.New(strings.Join(msg, ", "))
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
	if code >= 400 {
		var err = errors.Errorf("HTTP %d", code)
		var apiError APIError
		if errX := json.Unmarshal(res, &apiError); errX == nil {
			err = errors.Wrap(apiError.Error(), fmt.Sprintf("HTTP %d", code))
		}
		return res, nil, code, err
	}

	if err != nil {
		return res, nil, code, err
	}

	if code == 204 {
		return res, header, code, nil
	}

	if out != nil {
		if header.Get("Content-Type") == "application/gzip" {
			zreader, err := zip.NewReader(bytes.NewReader(res), int64(len(res)))
			if err != nil {
				return res, header, code, errors.Wrap(err, "unable to open zip content")
			}

			for _, f := range zreader.File {
				fi, err := f.Open()
				if err != nil {
					return res, header, code, errors.Wrap(err, "unable to open zipped file")
				}
				btes, err := io.ReadAll(fi)
				if err != nil {
					return res, header, code, errors.Wrap(err, "unable to read zip content")
				}
				res = btes
				break // only handle the first "file" of the zip archive
			}
		}

		switch x := out.(type) {
		case *json.RawMessage:
			*x = res
			return res, nil, code, err
		default:
			if err := json.Unmarshal(res, out); err != nil {
				return res, nil, code, err
			}
		}
	}

	return res, header, code, nil
}

// Request executes an authentificated HTTP request on $path given $method and $args
func (c *client) Request(ctx context.Context, method string, path string, body io.Reader, mods ...RequestModifier) ([]byte, http.Header, int, error) {
	respBody, respHeader, code, err := c.Stream(ctx, c.Client, method, path, body, mods...)
	if err != nil {
		return nil, nil, 0, err
	}
	defer func() {
		// Drain and close the body to let the Transport reuse the connection
		_, _ = io.Copy(io.Discard, respBody)
		_ = respBody.Close()
	}()

	var bodyBtes []byte
	bodyBtes, err = io.ReadAll(respBody)
	if err != nil {
		return nil, nil, code, err
	}

	if code >= 400 {
		return bodyBtes, nil, code, errors.WithStack(fmt.Errorf("HTTP %d", code))
	}

	return bodyBtes, respHeader, code, nil
}

func (c *client) Stream(ctx context.Context, httpClient *http.Client, method string, path string, body io.Reader, mods ...RequestModifier) (io.ReadCloser, http.Header, int, error) {
	var url string
	if strings.HasPrefix(path, "http") {
		url = path
	} else {
		url = c.Config.Host + path
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, nil, 0, err
	}

	for i := range mods {
		if mods[i] != nil {
			mods[i](req)
		}
	}

	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Authorization", "Bearer "+c.Config.Token)
	req.Header.Set("Connection", "close")

	log.Debug(ctx, "%s %s", req.Method, req.URL.String())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}

	return resp.Body, resp.Header, resp.StatusCode, nil

}
