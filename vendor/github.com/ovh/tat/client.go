package tat

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/facebookgo/httpcontrol"
)

// Client represents a Client configuration to connect to api
type Client struct {
	username              string
	password              string
	url                   string
	referer               string
	requestTimeout        time.Duration
	maxTries              uint
	sslInsecureSkipVerify bool
}

//Options is a struct to initialize a TAT client
type Options struct {
	Username              string
	Password              string
	URL                   string
	Referer               string
	RequestTimeout        time.Duration
	MaxTries              uint
	SSLInsecureSkipVerify bool
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPClient is HTTClient or testHTTPClient for tests
var HTTPClient httpClient

// DebugLogFunc is a function that logs the provided message with optional fmt.Sprintf-style arguments. By default, logs to the default log.Logger.
var DebugLogFunc = log.Printf //func(string, ...interface{})

// ErrorLogFunc is a function that logs the provided message with optional fmt.Sprintf-style arguments. By default, logs to the default log.Logger.
var ErrorLogFunc = log.Printf

// IsDebug display request / response in ErrorLogFunc if true
var IsDebug = false

//ErrClientNotInitiliazed is a predifined Error
var ErrClientNotInitiliazed = fmt.Errorf("Client is not initialized")

//NewClient initialize a TAT client
func NewClient(opts Options) (*Client, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("Invalid configuration, please check url of Tat Engine")
	}
	c := &Client{
		url:                   opts.URL,
		username:              opts.Username,
		password:              opts.Password,
		referer:               "TAT-SDK-" + Version,
		requestTimeout:        time.Minute,
		maxTries:              5,
		sslInsecureSkipVerify: opts.SSLInsecureSkipVerify,
	}
	if opts.Referer != "" {
		c.referer = opts.Referer
	}
	if opts.RequestTimeout != time.Duration(0) {
		c.requestTimeout = opts.RequestTimeout
	}
	if opts.MaxTries != 0 {
		c.maxTries = opts.MaxTries
	}

	return c, nil
}

func (c *Client) initHeaders(req *http.Request) error {
	if c == nil {
		return ErrClientNotInitiliazed
	}

	req.Header.Set(TatHeaderUsername, c.username)
	req.Header.Set(TatHeaderPassword, c.password)
	req.Header.Set(TatHeaderXTatRefererLower, c.referer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "close")
	return nil
}

// IsHTTPS returns true if url begins with https
func (c *Client) IsHTTPS() bool {
	return strings.HasPrefix(c.url, "https")
}

func (c *Client) reqWant(method string, wantCode int, path string, jsonStr []byte) ([]byte, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	requestPath := c.url + path
	var req *http.Request
	if jsonStr != nil {
		req, _ = http.NewRequest(method, requestPath, bytes.NewReader(jsonStr))
	} else {
		req, _ = http.NewRequest(method, requestPath, nil)
	}

	c.initHeaders(req)

	if HTTPClient == nil {
		HTTPClient = &http.Client{
			Transport: &httpcontrol.Transport{
				RequestTimeout: c.requestTimeout,
				MaxTries:       c.maxTries,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: c.sslInsecureSkipVerify,
				},
			},
		}
	}
	resp, err := HTTPClient.Do(req)

	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp == nil {
		ErrorLogFunc("Invalid response from Tat. Please Check Tat Engine, err:%s", err)
		return []byte{}, fmt.Errorf("Invalid response from Tat. Please Check Tat Engine, err:%s", err)
	}
	if resp.StatusCode != wantCode || IsDebug {
		ErrorLogFunc("Request Username:%s Referer:%s Path:%s", c.username, c.referer, requestPath)
		ErrorLogFunc("Request Body:%s", string(jsonStr))
		ErrorLogFunc("Response Status:%s", resp.Status)
		ErrorLogFunc("Response Headers:%s", resp.Header)
		if resp.StatusCode != wantCode {
			body, errc := ioutil.ReadAll(resp.Body)
			if errc != nil {
				ErrorLogFunc("Error with ioutil.ReadAll (with statusCode %d != wantCode %d) %s", resp.StatusCode, wantCode, errc)
				return []byte{}, fmt.Errorf("Response code:%d (want:%d) err on readAll Body:%s", resp.StatusCode, wantCode, errc)
			}
			ErrorLogFunc("Response Body:%s", string(body))
			return []byte{}, fmt.Errorf("Response code:%d (want:%d) with Body:%s", resp.StatusCode, wantCode, string(body))
		}
	}
	DebugLogFunc("%s %s", method, requestPath)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ErrorLogFunc("Error with ioutil.ReadAll %s", err)
		return nil, fmt.Errorf("Error with ioutil.ReadAll %s", err.Error())
	}
	if IsDebug {
		ErrorLogFunc("Debug Response Body:%s", string(body))
	}
	return body, nil
}

func (c *Client) simpleGetAndGetBytes(url string) ([]byte, error) {
	out, err := c.reqWant("GET", 200, url, nil)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) simplePutAndGetBytes(url string, want int, v interface{}) ([]byte, error) {
	return c.simpleReqAndGetBytes("PUT", url, want, v)
}

func (c *Client) simplePostAndGetBytes(url string, want int, v interface{}) ([]byte, error) {
	return c.simpleReqAndGetBytes("POST", url, want, v)
}

func (c *Client) simpleDeleteAndGetBytes(url string, want int, v interface{}) ([]byte, error) {
	return c.simpleReqAndGetBytes("DELETE", url, want, v)
}

func (c *Client) simpleReqAndGetBytes(method, url string, want int, v interface{}) ([]byte, error) {
	var jsonStr []byte
	var err error
	if v != nil {
		jsonStr, err = json.Marshal(v)
		if err != nil {
			ErrorLogFunc("Error while convert json:%s", err)
			return nil, err
		}
	}

	out, err := c.reqWant(method, want, url, jsonStr)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Sprint return the value in json as a string
func Sprint(v interface{}) ([]byte, error) {
	jsonStr, err := json.Marshal(v)
	if err != nil {
		ErrorLogFunc("Error while convert response from tat:%s", err)
		return []byte("Error while convert json struct from tat api"), err
	}
	return jsonStr, nil
}
