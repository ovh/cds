package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
	"github.com/ovh/cds/sdk/interpolate"
)

/*
This plugin have to be used as a deployment integration plugin

Arsenal deployment plugin must configured as following:
	name: arsenal-deployment-plugin
	type: integration
	author: "François Samin"
	description: "OVH Arsenal Deployment Plugin"

$ cdsctl admin plugins import arsenal-deployment-plugin.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add arsenal-deployment-plugin arsenal-deployment-plugin-bin.yml <path-to-binary-file>

Arsenal integration must configured as following
	name: Arsenal
	default_config:
		host:
			type: string
	deployment: true
	additional_default_config:
		version:
			type: string
		alternative.config:
			type: text
		deployment.token:
			type: password
		retry.max:
			type: string
			value: 10
		retry.delay:
			type: string
			value 5
	plugin: arsenal-deployment-plugin
*/

// alternativeConfig represents an alternative to a deployment.
type alternativeConfig struct {
	Name    string                 `json:"name"`
	From    string                 `json:"from,omitempty"`
	Config  map[string]interface{} `json:"config"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// deployRequest represents a deploy request to arsenal.
type deployRequest struct {
	Version     string            `json:"version"`
	Alternative string            `json:"alternative,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

// String returns a string representation of a deploy request. Omits metadata.
func (r *deployRequest) String() string {
	s := "Version: " + r.Version
	if r.Alternative != "" {
		s += "; Alternative: " + r.Alternative
	}
	return s
}

// followupState is the followup status of a deploy request.
type followupState struct {
	Done     bool    `json:"done"`
	Progress float64 `json:"progress"`
}

type arsenalDeploymentPlugin struct {
	integrationplugin.Common
}

func (e *arsenalDeploymentPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "OVH Arsenal Deployment Plugin",
		Author:      "François Samin",
		Description: "OVH Arsenal Deployment Plugin",
		Version:     sdk.VERSION,
	}, nil
}

const deployData = `{
	"version": "{{.cds.integration.deployment.version}}",
	"metadata": {
		"CDS_APPLICATION": "{{.cds.application}}",
		"CDS_RUN": "{{.cds.run}}",
		"CDS_ENVIRONMENT": "{{.cds.integration.deployment}}",
		"CDS_GIT_BRANCH": "{{.git.branch}}",
		"CDS_WORKFLOW": "{{.cds.workflow}}",
		"CDS_PROJECT": "{{.cds.project}}",
		"CDS_VERSION": "{{.cds.version}}",
		"CDS_GIT_REPOSITORY": "{{.git.repository}}",
		"CDS_GIT_HASH": "{{.git.hash}}"
	}
}`

func (e *arsenalDeploymentPlugin) Run(ctx context.Context, q *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	// Read and check inputs
	var (
		application     = getStringOption(q, "cds.application")
		arsenalHost     = getStringOption(q, "cds.integration.deployment.host")
		deploymentToken = getStringOption(q, "cds.integration.deployment.deployment.token", "cds.integration.deployment.token")
		alternative     = getStringOption(q, "cds.integration.deployment.alternative.config")
	)
	maxRetry, err := getIntOption(q, "cds.integration.deployment.retry.max")
	if err != nil {
		fmt.Printf("Error parsing cds.integration.deployment.retry.max: %v. Default value will be used\n", err)
		maxRetry = 10
	}
	delayRetry, err := getIntOption(q, "cds.integration.deployment.retry.delay")
	if err != nil {
		fmt.Printf("Error parsing cds.integration.deployment.retry.max: %v. Default value will be used\n", err)
		delayRetry = 5
	}
	if arsenalHost == "" {
		return fail("missing arsenal host")
	}
	if deploymentToken == "" {
		return fail("missing arsenal deployment token")
	}

	arsenalClient := newArsenalClient(arsenalHost, deploymentToken)

	// Read alternative if configured.
	var altConfig *alternativeConfig
	if len(alternative) > 0 {
		// Resolve alternative.
		altTmpl, err := template.New("alternative").Delims("[[", "]]").Funcs(interpolate.InterpolateHelperFuncs).Parse(alternative)
		if err != nil {
			return fail("failed to resolve alternative config: %v\n", err)
		}
		var altBuf bytes.Buffer
		if err = altTmpl.Execute(&altBuf, q.GetOptions()); err != nil {
			return fail("failed to interpolate alternative config: %v", err)
		}

		// Create alternative if anything was resolved.
		if altBuf.Len() > 0 {
			if err = json.Unmarshal(altBuf.Bytes(), &altConfig); err != nil {
				fmt.Println("Resolved alternative:", altBuf.String())
				return fail("failed to unmarshal alternative config: %v", err)
			}

			// Create alternative on /alternative
			fmt.Printf("Creating alternative %s on Arsenal...\n", altConfig.Name)
			if err = arsenalClient.upsertAlternative(altConfig); err != nil {
				return failErr(err)
			}
		}
	}

	// Build deploy request
	deployData, err := interpolate.Do(string(deployData), q.GetOptions())
	if err != nil {
		return fail("unable to interpolate data: %v\n", err)
	}
	deployReq := &deployRequest{}
	err = json.Unmarshal([]byte(deployData), deployReq)
	if err != nil {
		return fail("unable to create deploy request: %v\n", err)
	}
	if altConfig != nil {
		deployReq.Alternative = altConfig.Name
	}

	// Do deploy request
	fmt.Printf("Deploying %s (%s) on Arsenal at %s...\n", application, deployReq, arsenalHost)
	followUpToken, err := arsenalClient.deploy(deployReq)
	if err != nil {
		return fail("deploy failed: %v", err)
	}

	// Retry loop to follow the deployment status
	var retry int
	var success bool
	var lastProgress float64
	for retry < maxRetry {
		if retry > 0 {
			time.Sleep(time.Duration(delayRetry) * time.Second)
		}

		fmt.Println("Fetching followup status on deployment...")
		state, err := arsenalClient.follow(followUpToken)
		if err != nil {
			return failErr(err)
		}
		if state == nil {
			retry++
			fmt.Println("Arsenal service unavailable, waiting for next retry")
			continue
		}
		if state.Done {
			success = true
			break
		}
		// If the progress is back to 0 after subsequent call to follows, it means
		// it was probably cancelled on the platform side.
		if state.Progress < lastProgress && state.Progress == 0 {
			fmt.Println("Deployment cancelled.")
			break
		}
		lastProgress = state.Progress

		fmt.Printf("Deployment still in progress (%.1f%%)...\n", lastProgress*100)
		retry++
	}

	if !success {
		return fail("deployment failed after %d retries", retry)
	}

	fmt.Println("Deployment succeeded.")
	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

const (
	arsenalDeploymentTokenHeader = "X-Arsenal-Deployment-Token"
	arsenalFollowupTokenHeader   = "X-Arsenal-Followup-Token"
)

// arsenalClient is a helper client to call arsenal public API.
type arsenalClient struct {
	client          *http.Client
	host            string
	deploymentToken string
}

func newArsenalClient(host, deploymentToken string) *arsenalClient {
	return &arsenalClient{
		client:          cdsclient.NewHTTPClient(60*time.Second, false),
		host:            host,
		deploymentToken: deploymentToken,
	}
}

// deploy makes a deploy request and returns a followup token if successful.
func (ac *arsenalClient) deploy(deployRequest *deployRequest) (string, error) {
	req, err := ac.newRequest(http.MethodPost, "/deploy", deployRequest)
	if err != nil {
		return "", err
	}
	req.Header.Add(arsenalDeploymentTokenHeader, ac.deploymentToken)

	deployResult := make(map[string]string)
	statusCode, rawBody, err := ac.doRequest(req, &deployResult)
	if err != nil {
		return "", err
	}
	if statusCode != http.StatusOK {
		if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
			return "", fmt.Errorf("deploy request failed (HTTP status %d): %s", statusCode, rawBody)
		}
		return "", fmt.Errorf("cannot reach Arsenal service (HTTP status %d)", statusCode)
	}
	token, exists := deployResult["followup_token"]
	if !exists {
		return "", fmt.Errorf("no followup token returned")
	}
	return token, nil
}

// follow makes a followup request with a followup token.
func (ac *arsenalClient) follow(followupToken string) (*followupState, error) {
	req, err := ac.newRequest(http.MethodGet, "/follow", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create follow request: %w", err)
	}
	req.Header.Add(arsenalFollowupTokenHeader, followupToken)

	state := &followupState{}
	statusCode, rawBody, err := ac.doRequest(req, state)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		if statusCode == http.StatusServiceUnavailable {
			return nil, nil
		}
		return nil, fmt.Errorf("failed follow request (HTTP status %d): %s", statusCode, rawBody)
	}
	return state, nil
}

// upsertAlternative creates or updates an alternative.
func (ac *arsenalClient) upsertAlternative(altConfig *alternativeConfig) error {
	req, err := ac.newRequest(http.MethodPost, "/alternative", altConfig)
	if err != nil {
		return fmt.Errorf("failed to create upsert alternative request: %w", err)
	}
	req.Header.Add(arsenalDeploymentTokenHeader, ac.deploymentToken)

	statusCode, rawBody, err := ac.doRequest(req, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
			return fmt.Errorf("failed upsert alternative request (HTTP status %d): %s", statusCode, rawBody)
		}
		return fmt.Errorf("cannot reach Arsenal service (HTTP status %d)", statusCode)
	}
	return nil
}

func (ac *arsenalClient) newRequest(method, uri string, obj interface{}) (*http.Request, error) {
	var body io.ReadCloser
	if obj != nil {
		objData, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("unable to encode request body: %w", err)
		}
		body = ioutil.NopCloser(bytes.NewReader(objData))
	}

	req, err := http.NewRequest(method, ac.host+uri, body)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare request on %s %s: %v", method, uri, err)
	}
	return req, nil
}

func (ac *arsenalClient) doRequest(req *http.Request, respObject interface{}) (int, []byte, error) {
	resp, err := ac.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s %s failed: %w", req.Method, req.URL, err)
	}
	defer resp.Body.Close()

	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read body from %s %s: %w", req.Method, req.URL, err)
	}
	if resp.StatusCode == http.StatusOK && respObject != nil {
		err = sdk.JSONUnmarshal(rawBody, respObject)
		if err != nil {
			return resp.StatusCode, nil, fmt.Errorf("failed to decode body from %s %s: %w", req.Method, req.URL, err)
		}
	}

	return resp.StatusCode, rawBody, nil
}

func getStringOption(q *integrationplugin.RunQuery, keys ...string) string {
	for _, k := range keys {
		if v, exists := q.GetOptions()[k]; exists {
			return v
		}
	}
	return ""
}

func getIntOption(q *integrationplugin.RunQuery, keys ...string) (int, error) {
	return strconv.Atoi(getStringOption(q, keys...))
}

func fail(format string, args ...interface{}) (*integrationplugin.RunResult, error) {
	return failErr(fmt.Errorf(format, args...))
}

func failErr(err error) (*integrationplugin.RunResult, error) {
	fmt.Println("Error:", err)
	return &integrationplugin.RunResult{
		Details: err.Error(),
		Status:  sdk.StatusFail,
	}, nil
}

func main() {
	e := arsenalDeploymentPlugin{}
	if err := integrationplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
}
