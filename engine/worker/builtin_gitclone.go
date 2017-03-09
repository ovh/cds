package main

import "github.com/ovh/cds/sdk"

func runGitClone(a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {

	url := sdk.ParameterFind(pbJob.Parameters, "url")
	privateKey := sdk.ParameterFind(pbJob.Parameters, "privateKey")
	user := sdk.ParameterFind(pbJob.Parameters, "user")
	password := sdk.ParameterFind(pbJob.Parameters, "password")
	branch := sdk.ParameterFind(pbJob.Parameters, "branch")
	commit := sdk.ParameterFind(pbJob.Parameters, "commit")
	directory := sdk.ParameterFind(pbJob.Parameters, "directory")

	if url == nil {
		res := sdk.Result{
			Status: sdk.StatusFail,
			Reason: "Git repository URL is not set. Nothing to perform.",
		}
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	return sdk.Result{}
}
