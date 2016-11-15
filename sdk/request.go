package sdk

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/spf13/viper"
)

var (
	verbose bool
	// Host defines the endpoint for all SDK requests
	Host           string
	user           string
	password       string
	token          string
	hash           string
	skipReadConfig bool
	// AuthHeader is used as HTTP header
	AuthHeader = "X_AUTH_HEADER"
	// RequestedWithHeader is used as HTTP header
	RequestedWithHeader = "X-Requested-With"
	// RequestedWithValue is used as HTTP header
	RequestedWithValue = "X-CDS-SDK"
	//SessionTokenHeader is user as HTTP header
	SessionTokenHeader = "Session-Token"
	// HTTP client
	client HttpClient
	// current agent calling
	agent Agent
	// CDSConfigFile is path to the default config file
	CDSConfigFile = path.Join(os.Getenv("HOME"), ".cds", "config.json")
)

// InitEndpoint force sdk package request to given endpoint
func InitEndpoint(en string) {
	Host = en
}

// Authorization set authorization header for all next call
func Authorization(h string) {
	hash = h
}

// Agent describe the type of authentication method to use
type Agent string

// Different values of agent
const (
	SDKAgent      Agent = "CDS/sdk"
	WorkerAgent         = "CDS/worker"
	HatcheryAgent       = "CDS/hatchery"
)

//SetAgent set a agent value
func SetAgent(a Agent) {
	agent = a
}

// If CDS_SKIP_VERIFY is present, use a specific http client
// with TLS InsecureSkipVerify enabled
func init() {
	agent = SDKAgent

	skip := os.Getenv("CDS_SKIP_VERIFY")
	if skip != "" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
		return
	}

	client = http.DefaultClient
}

func initRequest(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", string(agent))
	req.Header.Set("Connection", "close")
	req.Header.Add(RequestedWithHeader, RequestedWithValue)
}

func readConfig() error {
	if skipReadConfig {
		return nil
	}
	skipReadConfig = true

	viper.SetConfigFile(CDSConfigFile)
	err := viper.ReadInConfig()
	if err == nil {
		if viper.GetString("host") != "" {
			Host = viper.GetString("host")
		}
		if viper.GetString("user") != "" {
			user = viper.GetString("user")
		}
		if viper.GetString("password") != "" {
			password = viper.GetString("password")
		}
		if viper.GetString("token") != "" {
			token = viper.GetString("token")
		}
	}

	if val := os.Getenv("CDS_USER"); val != "" {
		user = val
	}
	if val := os.Getenv("CDS_PASSWORD"); val != "" {
		password = val
	}
	if val := os.Getenv("CDS_TOKEN"); val != "" {
		token = val
	}

	if user != "" && (password != "" || token != "") {
		return nil
	}

	if hash != "" {
		return nil
	}

	if err != nil {
		fmt.Printf("Warning: Invalid configuration file (%s)\n", err)
	}

	return nil
}

// RequestModifier is used to modify behavior of Request and Steam functions
type RequestModifier func(req *http.Request)

// HttpClient is a interface for HttpClient mock
type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

//SetHTTPClient aims to change the default http client of the sdk
func SetHTTPClient(c HttpClient) {
	client = c
}

//Options set authentication data
func Options(h, u, p, t string) {
	Host = h
	user = u
	password = p
	token = t
	skipReadConfig = true
}

// SetHeader modify headers of http.Request
func SetHeader(key, value string) RequestModifier {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// Request executes an authentificated HTTP request on $path given $method and $args
func Request(method string, path string, args []byte, mods ...RequestModifier) ([]byte, int, error) {
	respBody, code, err := Stream(method, path, args, mods...)
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

	if verbose {
		fmt.Fprintf(os.Stderr, "Response Body: %s\n", body)
	}

	err = DecodeError(body)
	if err != nil {
		return nil, code, err
	}

	return body, code, nil
}

// Stream makes an authenticated http request and return io.ReadCloser
func Stream(method string, path string, args []byte, mods ...RequestModifier) (io.ReadCloser, int, error) {
	var savederror error

	err := readConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration: %s\n", err)
		os.Exit(1)
	}

	for i := 0; i < 10; i++ {
		var req *http.Request
		if args != nil {
			req, err = http.NewRequest(method, Host+path, bytes.NewReader(args))
		} else {
			req, err = http.NewRequest(method, Host+path, nil)
		}
		if err != nil {
			savederror = err
			continue
		}
		initRequest(req)

		for i := range mods {
			mods[i](req)
		}

		//No auth on /login route
		if !strings.HasPrefix(path, "/login") {
			if hash != "" {
				basedHash := base64.StdEncoding.EncodeToString([]byte(hash))
				req.Header.Set(AuthHeader, basedHash)
			}
			if user != "" && password != "" {
				req.SetBasicAuth(user, password)
			}
			if user != "" && token != "" {
				req.Header.Add(SessionTokenHeader, token)
				req.SetBasicAuth(user, token)
			}
		}

		resp, err := client.Do(req)

		// if everything is fine, return body
		if err == nil && resp.StatusCode < 500 {
			return resp.Body, resp.StatusCode, nil
		}

		// if no request error by status > 500, check CDS error
		// if there is a CDS errors, return it
		if err == nil && resp.StatusCode == 500 {
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				continue
			}
			cdserr := DecodeError(body)
			if cdserr != nil {
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

		if err != nil && (strings.Contains(err.Error(), "connection reset by peer") ||
			strings.Contains(err.Error(), "unexpected EOF")) {
			savederror = err
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			continue
		}

		if err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, fmt.Errorf("x10: %s", savederror)
}

// UploadMultiPart upload multipart
func UploadMultiPart(method string, path string, body *bytes.Buffer, mods ...RequestModifier) ([]byte, int, error) {

	err := readConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration: %s\n", err)
		os.Exit(1)
	}

	var req *http.Request
	req, _ = http.NewRequest(method, Host+path, body)
	if err != nil {
		return nil, 0, err
	}
	initRequest(req)

	for i := range mods {
		mods[i](req)
	}

	if hash != "" {
		basedHash := base64.StdEncoding.EncodeToString([]byte(hash))
		req.Header.Set(AuthHeader, basedHash)
	}
	if user != "" && password != "" {
		req.SetBasicAuth(user, password)
	}
	if user != "" && token != "" {
		req.Header.Add(SessionTokenHeader, token)
		req.SetBasicAuth(user, token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if verbose {
		fmt.Fprintf(os.Stderr, "Response Status: %s\n", resp.Status)
		fmt.Fprintf(os.Stderr, "Request path: %s\n", Host+path)
		fmt.Fprintf(os.Stderr, "Request Headers: %s\n", req.Header)
		fmt.Fprintf(os.Stderr, "Response Headers: %s\n", resp.Header)
	}

	var respBody []byte
	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Response Body: %s\n", body)
	}

	return respBody, resp.StatusCode, nil
}

// Upload upload content in given io.Reader to given HTTP endpoint
func Upload(method string, path string, body io.ReadCloser, mods ...RequestModifier) ([]byte, int, error) {

	err := readConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration: %s\n", err)
		os.Exit(1)
	}

	var req *http.Request
	req, _ = http.NewRequest(method, Host+path, body)
	if err != nil {
		return nil, 0, err
	}
	initRequest(req)

	for i := range mods {
		mods[i](req)
	}

	if hash != "" {
		basedHash := base64.StdEncoding.EncodeToString([]byte(hash))
		req.Header.Set(AuthHeader, basedHash)
	}
	if user != "" && password != "" {
		req.SetBasicAuth(user, password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if verbose {
		fmt.Fprintf(os.Stderr, "Response Status: %s\n", resp.Status)
		fmt.Fprintf(os.Stderr, "Request path: %s\n", Host+path)
		fmt.Fprintf(os.Stderr, "Request Headers: %s\n", req.Header)
		fmt.Fprintf(os.Stderr, "Response Headers: %s\n", resp.Header)
	}

	var respBody []byte
	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Response Body: %s\n", body)
	}

	return respBody, resp.StatusCode, nil
}

// DisplayStream decode each line from http buffer and print either message or error
func DisplayStream(buffer io.ReadCloser) error {
	reader := bufio.NewReader(buffer)

	for {
		line, err := reader.ReadBytes('\n')
		e := DecodeError(line)
		if e != nil {
			return e
		}
		if err != nil && err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "%s", line)
	}
}
