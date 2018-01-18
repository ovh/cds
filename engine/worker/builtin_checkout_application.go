package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runCheckoutApplication(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		// Load action param
		directory := sdk.ParameterFind(&a.Parameters, "directory")

		// Load build param
		branch := sdk.ParameterFind(params, "git.branch")
		defaultBranch := sdk.ParameterValue(*params, "git.default_branch")
		commit := sdk.ParameterFind(params, "git.commit")

		gitURL, auth, err := extractVCSInformations(*params)
		if err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: err.Error(),
			}
			sendLog(res.Reason)
			return res
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

		return gitClone(w, params, gitURL, dir, auth, clone, sendLog)
	}
}
