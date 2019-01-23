package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/platformplugin"
	"github.com/ovh/cds/sdk/interpolate"
)

/*
This plugin have to be used as a deployment platform plugin

This is an example, functional and almost complete. You have to add
some code to call you "deployment" system.

You can use the Makefile to build & publish the plugin on you CDS API.
The Makefile use the cdsctl binary, you need to be an administrator of your
CDS API to import plugin and create the deployment platform.

Hello deployment plugin must configured as following (content of hello-deployment-plugin.yml):
	name: hello-deployment-plugin
	type: platform
	author: "Yvonnick Esnault"
	description: "Hello Example Deployment Plugin"

$ cdsctl admin plugins import hello-deployment-plugin.yml

Build the present binaries and import in CDS (content of hello-deployment-plugin-bin.yml):
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add hello-deployment-plugin hello-deployment-plugin-bin.yml <path-to-binary-file>

Hello platform must configured as following (content of hello-platform.yml)
	name: Hello
	author: "Username Lastname"
	default_config: {}
	deployment_default_config:
	deployment.token:
		value: ""
		type: password
	retry.delay:
		value: "10"
		type: string
	retry.max:
		value: "30"
		type: string
	version:
		value: '{{.cds.version}}'
		type: string
	plugin: hello-deployment-plugin
	public_configurations:
	hello-platform-dev:
		host:
		value: http://hello.your-deployment-platform.dev.local
		type: string
	hello-platform-prod:
		host:
		value: http://hello.your-deployment-platform.prod.local
		type: string
	deployment: true
	public: true

$ cdsctl admin platform-model import hello-platform.yml

*/

type helloDeploymentPlugin struct {
	platformplugin.Common
}

func (e *helloDeploymentPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*platformplugin.PlatformPluginManifest, error) {
	return &platformplugin.PlatformPluginManifest{
		Name:        "Hello Example Deployment Plugin",
		Author:      "Yvonnick Esnault",
		Description: "Hello Example Deployment Plugin",
		Version:     sdk.VERSION,
	}, nil
}

// deployData is an example of variable that can be send
// to your "deployment" system. All data will be interpolate
// with the real values below, by calling interpolate.Do func.
const deployData = `{
	"version": "{{.cds.platform.version}}",
	"metadata": {
		"CDS_APPLICATION": "{{.cds.application}}",
		"CDS_RUN": "{{.cds.run}}",
		"CDS_ENVIRONMENT": "{{.cds.platform}}",
		"CDS_GIT_BRANCH": "{{.git.branch}}",
		"CDS_WORKFLOW": "{{.cds.workflow}}",
		"CDS_PROJECT": "{{.cds.project}}",
		"CDS_VERSION": "{{.cds.version}}",
		"CDS_SEMVER": "{{.cds.semver}}",
		"CDS_GIT_REPOSITORY": "{{.git.repository}}",
		"CDS_GIT_HASH": "{{.git.hash}}"
	}
}`

func (e *helloDeploymentPlugin) Deploy(ctx context.Context, q *platformplugin.DeployQuery) (*platformplugin.DeployResult, error) {
	var application = q.GetOptions()["cds.application"]
	var helloHost = q.GetOptions()["cds.platform.host"]
	var maxRetryStr = q.GetOptions()["cds.platform.retry.max"]
	var delayRetryStr = q.GetOptions()["cds.platform.retry.delay"]
	maxRetry, err := strconv.Atoi(maxRetryStr)
	if err != nil {
		fmt.Printf("Error parsing cds.platform.retry.max: %v. Default value (10) will be used\n", err)
		maxRetry = 10
	}
	delayRetry, err := strconv.Atoi(delayRetryStr)
	if err != nil {
		fmt.Printf("Error parsing cds.platform.retry.max: %v. Default value (5) will be used\n", err)
		delayRetry = 5
	}

	deployData, err := interpolate.Do(deployData, q.GetOptions())
	if err != nil {
		return fail("Error: unable to interpolate data: %v. Please check you platform configuration\n", err)
	}

	fmt.Printf("Deploying %s on Hello at %s...\n", application, helloHost)
	fmt.Printf("Retry.max %d\n", maxRetry)
	fmt.Printf("Retry.delay %d\n", delayRetry)
	fmt.Printf("Metadata %v \n", deployData)

	// Here, you should do the request on the deployment" system
	// you can use the deployData to send it some information about current job

	// After doing the request to deploy, you can follow the deployement
	// by request your "deployment" system
	// below is an example on how to use retry and maxRetry options.

	//Retry loop to follow the deployment status
	var retry = 0
	var success bool
	for retry < maxRetry {
		if retry > 0 {
			fmt.Printf("Retrying in %s seconds...\n", delayRetryStr)
			time.Sleep(time.Duration(delayRetry) * time.Second)
		}

		// here, you can request your "deployment" system to have to status
		// of the deploy action.

		// here, a code just to make this example working
		if retry == 2 {
			fmt.Println("Fake deploy on Hello platform done")
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

	return &platformplugin.DeployResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func (e *helloDeploymentPlugin) DeployStatus(ctx context.Context, q *platformplugin.DeployStatusQuery) (*platformplugin.DeployResult, error) {
	return &platformplugin.DeployResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func main() {
	e := helloDeploymentPlugin{}
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
