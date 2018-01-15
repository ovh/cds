package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runCheckoutApplication(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		branch := sdk.ParameterFind(a.Parameters, "git.branch")
		defaultBranch := sdk.ParameterValue(*params, "git.default_branch")
		commit := sdk.ParameterFind(a.Parameters, "git.commit")
		directory := sdk.ParameterFind(a.Parameters, "directory")

		// Get connection type
		connetionType := sdk.ParameterFind(a.Parameters, "git.connection.type")
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
			keyName := sdk.ParameterFind(a.Parameters, "git.ssh.key")
			if keyName == nil || keyName.Value == "" {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "Git ssh key is not set. Nothing to perform.",
				}
				sendLog(res.Reason)
				return res
			}

			privateKey := sdk.ParameterFind(a.Parameters, "cds.key."+keyName.Value+".priv")
			if privateKey == nil || privateKey.Value != "" {
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
			p := filepath.Join(keysDirectory, keyName.Value)
			b, err := ioutil.ReadFile(p)
			if err != nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Unable to setup ssh key. %s", err),
				}
				sendLog(res.Reason)
				return res
			}
			auth = new(git.AuthOpts)
			auth.PrivateKey = vcs.SSHKey{Filename: p, Content: b}

			url := sdk.ParameterFind(a.Parameters, "git.url")
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
			user := sdk.ParameterFind(a.Parameters, "git.http.user")
			password := sdk.ParameterFind(a.Parameters, "git.http.password")

			if user != nil || password != nil {
				auth = new(git.AuthOpts)
				if user != nil {
					auth.Username = user.Value
				}
				if password != nil {
					auth.Password = password.Value
				}
			}

			url := sdk.ParameterFind(a.Parameters, "git.http_url")
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
