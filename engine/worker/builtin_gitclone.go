package main

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runGitClone(*currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params []sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		url := sdk.ParameterFind(a.Parameters, "url")
		privateKey := sdk.ParameterFind(a.Parameters, "privateKey")
		user := sdk.ParameterFind(a.Parameters, "user")
		password := sdk.ParameterFind(a.Parameters, "password")
		branch := sdk.ParameterFind(a.Parameters, "branch")
		commit := sdk.ParameterFind(a.Parameters, "commit")
		directory := sdk.ParameterFind(a.Parameters, "directory")

		if url == nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Git repository URL is not set. Nothing to perform.",
			}
			sendLog(res.Reason)
			return res
		}

		if privateKey != nil {
			//Setup the key
			if err := vcs.SetupSSHKey(nil, keysDirectory, privateKey); err != nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Unable to setup ssh key. %s", err),
				}
				sendLog(res.Reason)
				return res
			}
		}

		//Get the key
		key, errK := vcs.GetSSHKey(params, keysDirectory, privateKey)
		if errK != nil && errK != sdk.ErrKeyNotFound {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to setup ssh key. %s", errK),
			}
			sendLog(res.Reason)
			return res
		}

		//If url is not http(s), a key must be found
		if !strings.HasPrefix(url.Value, "http") {
			if errK == sdk.ErrKeyNotFound || key == nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("SSH Key not found. Unable to perform git clone"),
				}
				sendLog(res.Reason)
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
			sendLog(stdOut.String())
		}
		if len(stdErr.Bytes()) > 0 {
			sendLog(stdErr.String())
		}

		if err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to git clone: %s", err),
			}
			sendLog(res.Reason)
			return res
		}

		time.Sleep(5 * time.Second)
		return sdk.Result{Status: sdk.StatusSuccess.String()}
	}
}
