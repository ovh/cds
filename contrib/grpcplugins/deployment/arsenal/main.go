package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/platformplugin"
	"github.com/ovh/cds/sdk/interpolate"
)

/*
This plugin have to be used as a deployment platform plugin

Arsenal deployment plugin must configured as following:
	name: arsenal-deployment-plugin
	type: platform
	author: "François Samin"
	description: "OVH Arsenal Deployment Plugin"

$ cdsctl admin plugins import arsenal-deployment-plugin.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add arsenal-deployment-plugin arsenal-deployment-plugin-bin.yml <path-to-binary-file>

Arsenal platform must configured as following
	name: Arsenal
	default_config:
		host:
			type: string
	deployment: true
	deployment_default_config:
		version:
			type: string
		deployment.token:
			type: string
		retry.max:
			type: string
			value: 10
		retry.delay:
			type: string
			value 5
	plugin: arsenal-deployment-plugin
*/

type arsenalDeploymentPlugin struct {
	platformplugin.Common
}

func (e *arsenalDeploymentPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*platformplugin.PlatformPluginManifest, error) {
	return &platformplugin.PlatformPluginManifest{
		Name:        "OVH Arsenal Deployment Plugin",
		Author:      "François Samin",
		Description: "OVH Arsenal Deployment Plugin",
		Version:     sdk.VERSION,
	}, nil
}

const deployData = `{
	"version": "{{.cds.platform.version}}",
	"metadata": {
		"CDS_APPLICATION": "{{.cds.application}}",
		"CDS_RUN": "{{.cds.run}}",
		"CDS_ENVIRONMENT": "{{.cds.environment}}",
		"CDS_GIT_BRANCH": "{{.git.branch}}",
		"CDS_WORKFLOW": "{{.cds.workflow}}",
		"CDS_PROJECT": "{{.cds.project}}",
		"CDS_VERSION": "{{.cds.version}}",
		"CDS_GIT_REPOSITORY": "{{.git.repository}}",
		"CDS_GIT_HASH": "{{.git.hash}}"
	}
}`

func (e *arsenalDeploymentPlugin) Deploy(ctx context.Context, q *platformplugin.DeployQuery) (*platformplugin.DeployResult, error) {
	var application = q.GetOptions()["cds.application"]
	var arsenalHost = q.GetOptions()["cds.platform.host"]
	var arsenalDeploymentToken = q.GetOptions()["cds.platform.deployment.token"]
	var maxRetryStr = q.GetOptions()["cds.platform.retry.max"]
	var delayRetryStr = q.GetOptions()["cds.platform.retry.delay"]
	maxRetry, err := strconv.Atoi(maxRetryStr)
	if err != nil {
		fmt.Printf("Error parsing cds.platform.retry.max: %v. Default value will be used", err)
		maxRetry = 10
	}
	delayRetry, err := strconv.Atoi(delayRetryStr)
	if err != nil {
		fmt.Printf("Error parsing cds.platform.retry.max: %v. Default value will be used", err)
		delayRetry = 5
	}

	deployData, err := interpolate.Do(deployData, q.GetOptions())
	if err != nil {
		return fail("Error: unable to interpolate data: %v. Please check you platform configuration", err)
	}

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	// Prepare the request
	req, err := http.NewRequest(http.MethodPost, arsenalHost+"/deploy", strings.NewReader(deployData))
	if err != nil {
		return fail("Error: unable to prepare request on %s/deploy: %v", arsenalHost, err)
	}
	req.Header.Add("X-Arsenal-Deployment-Token", arsenalDeploymentToken)

	fmt.Printf("Deploying %s on Arsenal at %s...", application, arsenalHost)

	// Do the request
	res, err := httpClient.Do(req)
	if err != nil {
		return fail("Error: Post %s/deploy failed: %v. Please check you platform configuration", arsenalHost, err)
	}
	defer res.Body.Close()

	//Check the result
	body, _ := ioutil.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		fmt.Println(string(body))
		return fail("deployment failure")
	}

	//Read the followUp token
	bodyResult := map[string]string{}
	if err := json.Unmarshal(body, &bodyResult); err != nil {
		return fail("Error: Unable to read body: %v", err)
	}
	var followUpToken = bodyResult["followup_token"]

	//Retry loop to follow the deployment status
	var retry = 0
	var success bool
	for retry < maxRetry {
		if retry > 0 {
			fmt.Printf("Retrying in %s seconds...", delayRetryStr)
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
			return fail("Deployment failed: %v. Please check you platform configuration", err)
		}
		defer res.Body.Close()

		body, _ := ioutil.ReadAll(res.Body)
		if res.StatusCode != http.StatusOK {
			fmt.Println(string(body))
			return fail("deployment failure")
		}

		//Read the followUp token
		bodyResult := map[string]string{}
		if err := json.Unmarshal(body, &bodyResult); err != nil {
			return fail("Error: Unable to read body: %v", err)
		}

		if bodyResult["done"] == "true" {
			success = true
			break
		} else {
			fmt.Println("Not done yet.")
		}
		retry++
	}

	if !success {
		return fail("deployment failed")
	}

	return &platformplugin.DeployResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func (e *arsenalDeploymentPlugin) DeployStatus(ctx context.Context, q *platformplugin.DeployStatusQuery) (*platformplugin.DeployResult, error) {
	return &platformplugin.DeployResult{
		Details: "none",
		Status:  "success",
	}, nil
}

func main() {
	e := arsenalDeploymentPlugin{}
	if err := platformplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
	return

}

func fail(format string, args ...interface{}) (*platformplugin.DeployResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &platformplugin.DeployResult{
		Details: msg,
		Status:  sdk.StatusFail.String(),
	}, nil
}
