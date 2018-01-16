package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runCheckoutApplication(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {

		// Load action param
		directory := sdk.ParameterFind(a.Parameters, "directory")

		// Load build param
		branch := sdk.ParameterFind(*params, "git.branch")
		defaultBranch := sdk.ParameterValue(*params, "git.default_branch")
		commit := sdk.ParameterFind(*params, "git.commit")

		// Get connection type
		connetionType := sdk.ParameterFind(*params, "git.connection.type")
		if connetionType == nil || (connetionType.Value != "ssh" && connetionType.Value != "https") {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Git connection type is not set. Nothing to perform.",
			}
			sendLog(res.Reason)
			return res
		}

		var gitUrl string
		var auth *git.AuthOpts

		switch connetionType.Value {
		case "ssh":
			keyName := sdk.ParameterFind(*params, "git.ssh.key")
			if keyName == nil || keyName.Value == "" {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "Git ssh key is not set. Nothing to perform.",
				}
				sendLog(res.Reason)
				return res
			}

			privateKey := sdk.ParameterFind(*params, "cds.key."+keyName.Value+".priv")
			if privateKey == nil || privateKey.Value == "" {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "SSH key not found. Nothing to perform.",
				}
				sendLog(res.Reason)
				return res
			}
			if err := vcs.SetupSSHKey(nil, keysDirectory, privateKey); err != nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Unable to setup ssh key. %s", err),
				}
				sendLog(res.Reason)
				return res
			}
			key, errK := vcs.GetSSHKey(*params, keysDirectory, privateKey)
			if errK != nil && errK != sdk.ErrKeyNotFound {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Unable to setup ssh key. %s", errK),
				}
				sendLog(res.Reason)
				return res
			}
			if key != nil {
				if auth == nil {
					auth = new(git.AuthOpts)
				}
				auth.PrivateKey = *key
			}

			url := sdk.ParameterFind(*params, "git.url")
			if url == nil || url.Value == "" {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "SSH Url (git.url) not found. Nothing to perform.",
				}
				sendLog(res.Reason)
				return res
			}
			gitUrl = url.Value
		case "https":
			user := sdk.ParameterFind(*params, "git.http.user")
			password := sdk.ParameterFind(*params, "git.http.password")

			if user != nil || password != nil {
				auth = new(git.AuthOpts)
				if user != nil {
					auth.Username = user.Value
				}
				if password != nil {
					auth.Password = password.Value
				}
			}

			url := sdk.ParameterFind(*params, "git.http_url")
			if url == nil || url.Value == "" {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "SSH Url (git.http_url) not found. Nothing to perform.",
				}
				sendLog(res.Reason)
				return res
			}
			gitUrl = url.Value
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

		// if there is no branch, check if there a defaultBranch
		if (clone.Branch == "" || clone.Branch == "{{.git.branch}}") && defaultBranch != "" {
			clone.Branch = defaultBranch
			clone.SingleBranch = false
			sendLog(fmt.Sprintf("branch is empty, using the default branch %s", defaultBranch))
		}

		r, _ := regexp.Compile("{{.*}}")
		if commit != nil && commit.Value != "" && !r.MatchString(commit.Value) {
			clone.CheckoutCommit = commit.Value
		}

		var dir string
		if directory != nil {
			dir = directory.Value
		}

		return gitClone(w, params, gitUrl, dir, auth, clone, sendLog)
	}
}
