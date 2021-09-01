package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
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
	deployment_default_config:
		version:
			type: string
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
		"CDS_ENVIRONMENT": "{{.cds.integration}}",
		"CDS_GIT_BRANCH": "{{.git.branch}}",
		"CDS_WORKFLOW": "{{.cds.workflow}}",
		"CDS_PROJECT": "{{.cds.project}}",
		"CDS_VERSION": "{{.cds.version}}",
		"CDS_SEMVER": "{{.cds.semver}}",
		"CDS_GIT_REPOSITORY": "{{.git.repository}}",
		"CDS_GIT_HASH": "{{.git.hash}}"
	}
}`

func (e *arsenalDeploymentPlugin) Run(ctx context.Context, q *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	var application = q.GetOptions()["cds.application"]
	var arsenalHost = q.GetOptions()["cds.integration.deployment.host"]
	var arsenalDeploymentToken = q.GetOptions()["cds.integration.deployment.deployment.token"]
	if arsenalDeploymentToken == "" {
		arsenalDeploymentToken = q.GetOptions()["cds.integration.deployment.token"]
	}
	var maxRetryStr = q.GetOptions()["cds.integration.deployment.retry.max"]
	var delayRetryStr = q.GetOptions()["cds.integration.deployment.retry.delay"]
	maxRetry, err := strconv.Atoi(maxRetryStr)
	if err != nil {
		fmt.Printf("Error parsing cds.integration.deployment.retry.max: %v. Default value will be used\n", err)
		maxRetry = 10
	}
	delayRetry, err := strconv.Atoi(delayRetryStr)
	if err != nil {
		fmt.Printf("Error parsing cds.integration.deployment.retry.max: %v. Default value will be used\n", err)
		delayRetry = 5
	}

	deployData, err := interpolate.Do(deployData, q.GetOptions())
	if err != nil {
		return fail("Error: unable to interpolate data: %v. Please check you integration configuration\n", err)
	}

	httpClient := cdsclient.NewHTTPClient(60*time.Second, false)

	// Prepare the request
	req, err := http.NewRequest(http.MethodPost, arsenalHost+"/deploy", strings.NewReader(deployData))
	if err != nil {
		return fail("Error: unable to prepare request on %s/deploy: %v", arsenalHost, err)
	}
	req.Header.Add("X-Arsenal-Deployment-Token", arsenalDeploymentToken)

	fmt.Printf("Deploying %s on Arsenal at %s...\n", application, arsenalHost)

	// Do the request
	res, err := httpClient.Do(req)
	if err != nil {
		return fail("Error: Post %s/deploy failed: %v. Please check you integration configuration", arsenalHost, err)
	}
	defer res.Body.Close()

	//Check the result
	body, _ := ioutil.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		fmt.Println("Body: ", string(body))
		return fail("deployment failure (HTTP Status Code: %d)", res.StatusCode)
	}

	//Read the followUp token
	bodyResult := map[string]string{}
	if err := sdk.JSONUnmarshal(body, &bodyResult); err != nil {
		return fail("Error: Unable to read body: %v", err)
	}
	var followUpToken = bodyResult["followup_token"]

	//Retry loop to follow the deployment status
	var retry = 0
	var success bool
	for retry < maxRetry {
		if retry > 0 {
			fmt.Printf("Retrying in %s seconds...\n", delayRetryStr)
			time.Sleep(time.Duration(delayRetry) * time.Second)
		}

		fmt.Println("Fetching followup status on deployment...")
		req, err := http.NewRequest(http.MethodGet, arsenalHost+"/follow", nil)
		if err != nil {
			return fail("Error: unable to prepare request on %s/follow: %v", arsenalHost, err)
		}
		req.Header.Add("X-Arsenal-Followup-Token", followUpToken)

		res, err := httpClient.Do(req)
		if err != nil {
			return fail("Deployment failed: %v. Please check you integration configuration", err)
		}
		defer res.Body.Close()

		body, _ := ioutil.ReadAll(res.Body)
		if res.StatusCode == http.StatusServiceUnavailable {
			retry++
			fmt.Println("Arsenal service unavailable, waiting for next retry")
			continue
		}
		if res.StatusCode != http.StatusOK {
			fmt.Println("Body: ", string(body))
			return fail("deployment failure")
		}

		//Read the followUp token
		bodyResult := map[string]interface{}{}
		if err := sdk.JSONUnmarshal(body, &bodyResult); err != nil {
			return fail("Error: Unable to read body: %v", err)
		}

		doneB, doneIsBool := bodyResult["done"].(bool)
		doneS, doneIsString := bodyResult["done"].(string)
		if (doneIsBool && doneB) || (doneIsString && doneS == "true") {
			success = true
			break
		} else {
			fmt.Println("Not done yet")
		}
		retry++
	}

	if !success {
		return fail("deployment failed")
	}

	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	e := arsenalDeploymentPlugin{}
	if err := integrationplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
}

func fail(format string, args ...interface{}) (*integrationplugin.RunResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &integrationplugin.RunResult{
		Details: msg,
		Status:  sdk.StatusFail,
	}, nil
}
