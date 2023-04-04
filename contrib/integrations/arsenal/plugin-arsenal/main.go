package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"text/template"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/integrations/arsenal"
	"github.com/ovh/cds/sdk"
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
		application       = getStringOption(q, "cds.application")
		workflowRunNumber = getStringOption(q, "cds.run.number")
		arsenalHost       = getStringOption(q, "cds.integration.deployment.host")
		deploymentToken   = getStringOption(q, "cds.integration.deployment.deployment.token", "cds.integration.deployment.token")
		alternative       = getStringOption(q, "cds.integration.deployment.alternative.config")
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

	arsenalClient := arsenal.NewClient(arsenalHost, deploymentToken)

	// Read alternative if configured.
	var altConfig *arsenal.Alternative
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

			// Add references for later processing.
			if altConfig.Options == nil {
				altConfig.Options = make(map[string]interface{})
			}
			altConfig.Options["cds_run"] = workflowRunNumber
			altConfig.Options["cds_application"] = application

			// Create alternative on /alternative
			rawAltConfig, _ := json.MarshalIndent(altConfig, "", "  ")
			fmt.Printf("Creating/Updating alternative: %s\n", rawAltConfig)
			if err = arsenalClient.UpsertAlternative(altConfig); err != nil {
				return failErr(err)
			}
		}
	}

	// Build deploy request
	deployData, err := interpolate.Do(string(deployData), q.GetOptions())
	if err != nil {
		return fail("unable to interpolate data: %v\n", err)
	}
	deployReq := &arsenal.DeployRequest{}
	err = json.Unmarshal([]byte(deployData), deployReq)
	if err != nil {
		return fail("unable to create deploy request: %v\n", err)
	}
	if altConfig != nil {
		deployReq.Alternative = altConfig.Name
	}

	// Retry loop to deploy an application
	// This loop consists of 6 retries (+ the first try), separated by 10 sec
	var retry int
	var followUpToken string
	for retry < 7 {
		if retry > 0 {
			time.Sleep(time.Duration(10) * time.Second)
		}

		fmt.Printf("Deploying %s (%s) on Arsenal at %s...\n", application, deployReq, arsenalHost)
		followUpToken, err = arsenalClient.Deploy(deployReq)
		if err != nil {
			if _, ok := err.(*arsenal.RequestError); ok {
				fmt.Println("Deployment has failed, retrying...")
				retry++
			} else {
				return fail("deploy failed: %v", err)
			}
		}

		if followUpToken != "" {
			break
		}
	}

	// Retry loop to follow the deployment status
	retry = 0
	var success bool
	var lastProgress float64
	for retry < maxRetry {
		if retry > 0 {
			time.Sleep(time.Duration(delayRetry) * time.Second)
		}

		fmt.Println("Fetching followup status on deployment...")
		state, err := arsenalClient.Follow(followUpToken)
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
