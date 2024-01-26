package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/contrib/integrations/arsenal"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/interpolate"
)

/*
This plugin have to be used to deploy an application.
*/

type deployArsenalPlugin struct {
	actionplugin.Common
}

func (e *deployArsenalPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "OVH Deploy Plugin",
		Author:      "Steven Guiheux",
		Description: "OVH Deploy Plugin",
		Version:     sdk.VERSION,
	}, nil
}

const deployData = `{
	"version": "%s",
	"metadata": {
		"CDS_RUN": "${{cds.run_number}}.${{cds.run_attempt}}",
		"CDS_GIT_BRANCH": "${{git.ref_name}}",
		"CDS_WORKFLOW": "${{cds.workflow}}",
		"CDS_PROJECT": "${{cds.project_key}}",
		"CDS_VERSION": "${{git.semver_current}}",
		"CDS_GIT_REPOSITORY": "${{git.repository}}",
		"CDS_GIT_HASH": "${{git.Sha}}"
	}
}`

func (e *deployArsenalPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	// Read and check inputs
	var (
		token             = q.GetOptions()["token"]
		version           = q.GetOptions()["version"]
		alternativeConfig = q.GetOptions()["alternative-config"]
	)

	if token == "" {
		return nil, fmt.Errorf("missing deployment token")
	}
	if version == "" {
		return nil, fmt.Errorf("missing deployment version")
	}

	maxRetry, err := strconv.Atoi(q.GetOptions()["retry-max"])
	if err != nil {
		fmt.Printf("Error parsing retry-max: %v. Default value will be used\n", err)
		maxRetry = 30
	}
	delayRetry, err := strconv.Atoi(q.GetOptions()["retry-delay"])
	if err != nil {
		fmt.Printf("Error parsing retry-delay: %v. Default value will be used\n", err)
		delayRetry = 10
	}

	jobRun, err := grpcplugins.GetJobRun(ctx, &e.Common)
	if err != nil {
		grpcplugins.Error(err.Error())
		return nil, err
	}
	contexts, err := grpcplugins.GetJobContext(ctx, &e.Common)
	if err != nil {
		grpcplugins.Error(err.Error())
		return nil, err
	}
	contextsBts, _ := json.Marshal(contexts)
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(contextsBts, &mapContexts); err != nil {
		return nil, err
	}

	var deploymentIntgration *sdk.ProjectIntegration
	for _, integrationName := range jobRun.Job.Integrations {
		integ, err := grpcplugins.GetIntegrationByName(ctx, &e.Common, integrationName)
		if err != nil {
			return nil, err
		}
		if integ.Model.Deployment {
			deploymentIntgration = integ
			break
		}
	}

	if deploymentIntgration == nil {
		return nil, fmt.Errorf("unable to retrieve a deployment integration")
	}
	if deploymentIntgration.Model.Name != "Arsenal" {
		return nil, fmt.Errorf("deploymnet integration is not Arsenal")
	}

	host := deploymentIntgration.Config["host"]
	if host.Value == "" {
		return nil, fmt.Errorf("missing arsenal host")
	}
	arsenalClient := arsenal.NewClient(host.Value, token)
	altConfig, err := createAlternative(ctx, arsenalClient, alternativeConfig, *contexts, mapContexts)
	if err != nil {
		return nil, err
	}

	deployData := fmt.Sprintf(deployData, version)
	ap := sdk.NewActionParser(mapContexts, nil)
	deploymentPayload, err := ap.InterpolateToString(ctx, string(deployData))
	if err != nil {
		return nil, err
	}

	deployReq := &arsenal.DeployRequest{}
	err = json.Unmarshal([]byte(deploymentPayload), deployReq)
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

		fmt.Printf("Deploying (%s) on Arsenal at %s...\n", deployReq, host.Value)
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
	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func createAlternative(ctx context.Context, arsenalClient *arsenal.Client, alternativeConfig string, contexts sdk.WorkflowRunJobsContext, mapContexts map[string]interface{}) (*arsenal.Alternative, error) {
	var createdConfig *arsenal.Alternative
	if len(alternativeConfig) > 0 {
		alternativeParser := sdk.NewActionParser(mapContexts, nil)
		altConfigInterpolated, err := alternativeParser.InterpolateToString(ctx, alternativeConfig)
		if err != nil {
			return nil, err
		}

		// Resolve alternative.
		altTmpl, err := template.New("alternative").Delims("[[", "]]").Funcs(interpolate.InterpolateHelperFuncs).Parse(altConfigInterpolated)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve alternative config: %v", err)
		}
		var altBuf bytes.Buffer
		if err = altTmpl.Execute(&altBuf, nil); err != nil {
			return nil, fmt.Errorf("failed to interpolate alternative config: %v", err)
		}

		// Create alternative if anything was resolved.
		if altBuf.Len() > 0 {
			if err = json.Unmarshal(altBuf.Bytes(), &createdConfig); err != nil {
				fmt.Println("Resolved alternative:", altBuf.String())
				return nil, fmt.Errorf("failed to unmarshal alternative config: %v", err)
			}

			// Add references for later processing.
			if createdConfig.Options == nil {
				createdConfig.Options = make(map[string]interface{})
			}
			createdConfig.Options["cds_run"] = contexts.CDS.RunNumber

			// Create alternative on /alternative
			rawAltConfig, _ := json.MarshalIndent(createdConfig, "", "  ")
			fmt.Printf("Creating/Updating alternative: %s\n", rawAltConfig)
			if err = arsenalClient.UpsertAlternative(createdConfig); err != nil {
				return nil, err
			}
		}
	}
	return createdConfig, nil
}

func fail(format string, args ...interface{}) (*actionplugin.ActionResult, error) {
	return failErr(fmt.Errorf(format, args...))
}

func failErr(err error) (*actionplugin.ActionResult, error) {
	fmt.Println("Error:", err)
	return &actionplugin.ActionResult{
		Details: err.Error(),
		Status:  sdk.StatusFail,
	}, nil
}

func main() {
	dp := deployArsenalPlugin{}
	if err := actionplugin.Start(context.Background(), &dp); err != nil {
		panic(err)
	}
	return
}
