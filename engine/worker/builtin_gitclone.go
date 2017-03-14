package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runGitClone(a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	url := sdk.ParameterFind(a.Parameters, "url")
	privateKey := sdk.ParameterFind(a.Parameters, "privateKey")
	user := sdk.ParameterFind(a.Parameters, "user")
	password := sdk.ParameterFind(a.Parameters, "password")
	branch := sdk.ParameterFind(a.Parameters, "branch")
	commit := sdk.ParameterFind(a.Parameters, "commit")
	directory := sdk.ParameterFind(a.Parameters, "directory")

	if url == nil {
		res := sdk.Result{
			Status: sdk.StatusFail,
			Reason: "Git repository URL is not set. Nothing to perform.",
		}
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		time.Sleep(5 * time.Second)
		return res
	}

	if privateKey != nil {
		//Setup the key
		if err := vcs.SetupSSHKey(nil, keysDirectory, privateKey); err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail,
				Reason: fmt.Sprintf("Unable to setup ssh key. %s", err),
			}
			sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
			time.Sleep(5 * time.Second)
			return res
		}
	}

	//Get the key
	key, errK := vcs.GetSSHKey(pbJob.Parameters, keysDirectory, privateKey)
	if errK != nil && errK != sdk.ErrKeyNotFound {
		res := sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Unable to setup ssh key. %s", errK),
		}
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		time.Sleep(5 * time.Second)
		return res
	}

	//If url is not http(s), a key must be found
	if !strings.HasPrefix(url.Value, "http") {
		if errK == sdk.ErrKeyNotFound || key == nil {
			res := sdk.Result{
				Status: sdk.StatusFail,
				Reason: fmt.Sprintf("SSH Key not found. Unable to perform git clone"),
			}
			sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
			time.Sleep(5 * time.Second)
			return res
		}
	}

	//Prepare all options - credentials
	var auth *git.AuthOpts
	if user != nil || password != nil {
		auth = new(git.AuthOpts)
		if user != nil {
			auth.Username = user.Value
		}
		if password != nil {
			auth.Password = password.Value
		}
	}

	if key != nil {
		if auth == nil {
			auth = new(git.AuthOpts)
		}
		auth.PrivateKey = *key
	}

	//Prepare all options - clone options
	var clone = &git.CloneOpts{
		Recursive:               true,
		NoStrictHostKeyChecking: true,
	}
	if branch != nil {
		clone.Branch = branch.Value
	} else {
		clone.SingleBranch = true
	}

	r, _ := regexp.Compile("{{.*}}")
	if commit != nil && commit.Value != "" && !r.MatchString(commit.Value) {
		clone.CheckoutCommit = commit.Value
	} else {
		clone.Depth = 1
	}

	var dir string
	if directory != nil {
		dir = directory.Value
	}

	//Prepare all options - logs
	stdErr := new(bytes.Buffer)
	stdOut := new(bytes.Buffer)

	output := &git.OutputOpts{
		Stderr: stdErr,
		Stdout: stdOut,
	}

	git.LogFunc = log.Info

	//Perform the git clone
	err := git.Clone(url.Value, dir, auth, clone, output)

	//Send the logs
	if len(stdOut.Bytes()) > 0 {
		sendLog(pbJob.ID, stdOut.String(), pbJob.PipelineBuildID, stepOrder, false)
	}
	if len(stdErr.Bytes()) > 0 {
		sendLog(pbJob.ID, stdErr.String(), pbJob.PipelineBuildID, stepOrder, false)
	}

	if err != nil {
		res := sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Unable to git clone: %s", err),
		}
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		time.Sleep(5 * time.Second)
		return res
	}

	time.Sleep(5 * time.Second)
	return sdk.Result{Status: sdk.StatusSuccess}
}
