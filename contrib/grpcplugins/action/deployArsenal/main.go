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
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
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

func (p *deployArsenalPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	// Read and check inputs
	var (
		token             = q.GetOptions()["token"]
		version           = q.GetOptions()["version"]
		alternativeConfig = q.GetOptions()["alternative-config"]
	)

	if token == "" {
		return fail(p, "missing deployment token")
	}
	if version == "" {
		return fail(p, "missing deployment version")
	}

	maxRetry, err := strconv.Atoi(q.GetOptions()["retry-max"])
	if err != nil {
		grpcplugins.Errorf(&p.Common, "Error parsing retry-max: %v. Default value will be used\n", err)
		maxRetry = 30
	}
	delayRetry, err := strconv.Atoi(q.GetOptions()["retry-delay"])
	if err != nil {
		grpcplugins.Errorf(&p.Common, "Error parsing retry-delay: %v. Default value will be used\n", err)
		delayRetry = 10
	}

	contexts, err := grpcplugins.GetJobContext(ctx, &p.Common)
	if err != nil {
		return fail(p, err.Error())
	}
	contextsBts, _ := json.Marshal(contexts)
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(contextsBts, &mapContexts); err != nil {
		return fail(p, err.Error())
	}

	if contexts.Integrations == nil || contexts.Integrations.Deployment.Name == "" {
		return fail(p, "unable to retrieve a deployment integration")
	}
	deploymentIntgration := contexts.Integrations.Deployment

	host := deploymentIntgration.Get("host")
	if host == "" {
		return fail(p, "missing arsenal host")
	}
	arsenalClient := arsenal.NewClient(host, token)
	altConfig, err := createAlternative(ctx, &p.Common, arsenalClient, alternativeConfig, *contexts, mapContexts)
	if err != nil {
		return fail(p, err.Error())
	}

	deployData := fmt.Sprintf(deployData, version)
	ap := sdk.NewActionParser(mapContexts, nil)
	deploymentPayload, err := ap.InterpolateToString(ctx, string(deployData))
	if err != nil {
		return fail(p, err.Error())
	}

	deployReq := &arsenal.DeployRequest{}
	err = json.Unmarshal([]byte(deploymentPayload), deployReq)
	if err != nil {
		return fail(p, fmt.Sprintf("unable to create deploy request: %v", err))
	}
	if altConfig != nil {
		deployReq.Alternative = altConfig.Name
	}

	// Retry loop to deploy an application
	// This loop consists of 6 retries (+ the first try), separated by 10 sec
	var retry int
	var deploymentResult *arsenal.DeployResponse
	for retry < 7 {
		if retry > 0 {
			time.Sleep(time.Duration(10) * time.Second)
		}
		grpcplugins.Logf(&p.Common, "Deploying (%s) on Arsenal at %s...\n", deployReq, host)
		deploymentResult, err = arsenalClient.Deploy(deployReq)
		if err != nil {
			if _, ok := err.(*arsenal.RequestError); ok {
				grpcplugins.Error(&p.Common, "Deployment has failed, retrying...")
				retry++
			} else {
				return fail(p, fmt.Sprintf("deploy failed: %v", err))
			}
		}

		if deploymentResult != nil {
			break
		}
	}
	if deploymentResult == nil {
		return fail(p, "deployment failed")
	}

	// Create run result at status "pending"
	var runResultRequest = workerruntime.V2RunResultRequest{
		RunResult: &sdk.V2WorkflowRunResult{
			IssuedAt: time.Now(),
			Type:     sdk.V2WorkflowRunResultTypeArsenalDeployment,
			Status:   sdk.V2WorkflowRunResultStatusPending,
			Detail:   sdk.V2WorkflowRunResultDetail{},
		},
	}
	data := sdk.V2WorkflowRunResultArsenalDeploymentDetail{
		Version:         version,
		DeploymentName:  deploymentResult.DeploymentName,
		DeploymentID:    deploymentResult.DeploymentID,
		StackID:         deploymentResult.StackID,
		StackName:       deploymentResult.StackName,
		StackPlatform:   deploymentResult.StackPlatform,
		Namespace:       deploymentResult.Namespace,
		IntegrationName: deploymentIntgration.Name,
		Alternative:     nil,
	}
	if altConfig != nil {
		data.Alternative = &sdk.ArsenalDeploymentDetailAlternative{
			Name:    altConfig.Name,
			From:    altConfig.From,
			Config:  altConfig.Config,
			Options: altConfig.Options,
		}
	}
	runResultRequest.RunResult.Detail.Data = data

	response, err := grpcplugins.CreateRunResult(ctx, &p.Common, &runResultRequest)
	if err != nil {
		return fail(p, fmt.Sprintf("failed to create run result: %v", err))
	}

	result := response.RunResult

	// Retry loop to follow the deployment status
	retry = 0
	var success bool
	var lastProgress float64
	for retry < maxRetry {
		if retry > 0 {
			time.Sleep(time.Duration(delayRetry) * time.Second)
		}

		grpcplugins.Logf(&p.Common, "Fetching followup status on deployment %s...", deploymentResult.DeploymentName)
		state, err := arsenalClient.Follow(deploymentResult.FollowUpToken)
		if err != nil {
			return fail(p, fmt.Sprintf("failed to check depoloyment status: %v", err))
		}
		if state == nil {
			retry++
			grpcplugins.Error(&p.Common, "Arsenal service unavailable, waiting for next retry")
			continue
		}
		if state.Done {
			success = true
			break
		}
		// If the progress is back to 0 after subsequent call to follows, it means
		// it was probably cancelled on the platform side.
		if state.Progress < lastProgress && state.Progress == 0 {
			grpcplugins.Error(&p.Common, "Deployment cancelled.")
			break
		}
		lastProgress = state.Progress

		grpcplugins.Logf(&p.Common, "Deployment still in progress (%.1f%%)...\n", lastProgress*100)
		retry++
	}

	if !success {
		return fail(p, fmt.Sprintf("deployment failed after %d retries", retry))
	}

	result.Status = sdk.V2WorkflowRunResultStatusCompleted
	if _, err := grpcplugins.UpdateRunResult(ctx, &p.Common, &workerruntime.V2RunResultRequest{RunResult: result}); err != nil {
		return fail(p, fmt.Sprintf("failed to update run result: %v", err))
	}

	grpcplugins.Logf(&p.Common, "Deployment of %s succeeded.", deploymentResult.DeploymentName)

	return stream.Send(&actionplugin.StreamResult{Status: sdk.StatusSuccess})
}

func fail(p *deployArsenalPlugin, err string) error {
	return p.StreamServer.Send(&actionplugin.StreamResult{Status: sdk.StatusFail, Details: err})
}

func (e *deployArsenalPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func createAlternative(ctx context.Context, p *actionplugin.Common, arsenalClient *arsenal.Client, alternativeConfig string, contexts sdk.WorkflowRunJobsContext, mapContexts map[string]interface{}) (*arsenal.Alternative, error) {
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
				grpcplugins.Error(p, "Resolved alternative: "+altBuf.String())
				return nil, fmt.Errorf("failed to unmarshal alternative config: %v", err)
			}

			// Add references for later processing.
			if createdConfig.Options == nil {
				createdConfig.Options = make(map[string]interface{})
			}
			createdConfig.Options["cds_run"] = contexts.CDS.RunNumber

			// Create alternative on /alternative
			rawAltConfig, _ := json.MarshalIndent(createdConfig, "", "  ")
			grpcplugins.Logf(p, "Creating/Updating alternative: %s\n", rawAltConfig)
			if err = arsenalClient.UpsertAlternative(createdConfig); err != nil {
				return nil, err
			}
		}
	}
	return createdConfig, nil
}

func main() {
	dp := deployArsenalPlugin{}
	if err := actionplugin.Start(context.Background(), &dp); err != nil {
		panic(err)
	}
	return
}
