package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runCheckoutApplication(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
		// Load action param
		directory := sdk.ParameterFind(&a.Parameters, "directory")

		// Load build param
		branch := sdk.ParameterFind(params, "git.branch")
		defaultBranch := sdk.ParameterValue(*params, "git.default_branch")
		tag := sdk.ParameterValue(*params, "git.tag")
		commit := sdk.ParameterFind(params, "git.hash")

		gitURL, auth, err := extractVCSInformations(*params, secrets)
		if err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: err.Error(),
			}
			sendLog(res.Reason)
			return res
		}

		//Prepare all options - clone options
		var opts = &git.CloneOpts{
			Recursive:               true,
			NoStrictHostKeyChecking: true,
			Depth:                   50,
			Tag:                     tag,
		}
		if branch != nil {
			opts.Branch = branch.Value
		} else {
			opts.SingleBranch = true
		}

		// if there is no branch, check if there a defaultBranch
		if (opts.Branch == "" || opts.Branch == "{{.git.branch}}") && defaultBranch != "" && tag == "" {
			opts.Branch = defaultBranch
			opts.SingleBranch = false
			sendLog(fmt.Sprintf("branch is empty, using the default branch %s", defaultBranch))
		}

		r := regexp.MustCompile("{{.*}}")
		if commit != nil && commit.Value != "" && !r.MatchString(commit.Value) {
			opts.CheckoutCommit = commit.Value
		}

		var dir string
		if directory != nil {
			dir = directory.Value
		}

		return gitClone(w, params, gitURL, dir, auth, opts, sendLog)
	}
}
