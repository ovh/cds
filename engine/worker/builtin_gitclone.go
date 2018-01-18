package main

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runGitClone(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		url := sdk.ParameterFind(a.Parameters, "url")
		privateKey := sdk.ParameterFind(a.Parameters, "privateKey")
		user := sdk.ParameterFind(a.Parameters, "user")
		password := sdk.ParameterFind(a.Parameters, "password")
		branch := sdk.ParameterFind(a.Parameters, "branch")
		defaultBranch := sdk.ParameterValue(*params, "git.default_branch")
		commit := sdk.ParameterFind(a.Parameters, "commit")
		directory := sdk.ParameterFind(a.Parameters, "directory")
		recursive := sdk.ParameterFind(a.Parameters, "recursive")
		cdsVersion := sdk.ParameterFind(*params, "cds.version")

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
		key, errK := vcs.GetSSHKey(*params, keysDirectory, privateKey)
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

		recursiveB, err := strconv.ParseBool(recursive.Value)
		if err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to parse recursive boolean. %s", err),
			}
			sendLog(res.Reason)
			return res
		}

		//Prepare all options - clone options
		var clone = &git.CloneOpts{
			Recursive:               recursiveB,
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

		//Prepare all options - logs
		stdErr := new(bytes.Buffer)
		stdOut := new(bytes.Buffer)

		output := &git.OutputOpts{
			Stderr: stdErr,
			Stdout: stdOut,
		}

		git.LogFunc = log.Info

		//Perform the git clone
		err = git.Clone(url.Value, dir, auth, clone, output)

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

		extractInfo(w, dir, params, clone.Branch, commit.Value, sendLog)

		stdTaglistErr := new(bytes.Buffer)
		stdTagListOut := new(bytes.Buffer)
		outputGitTag := &git.OutputOpts{
			Stderr: stdTaglistErr,
			Stdout: stdTagListOut,
		}

		errTag := git.TagList(url.Value, dir, auth, outputGitTag)

		if len(stdTaglistErr.Bytes()) > 0 {
			sendLog(stdTaglistErr.String())
		}

		if errTag != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to list tag for getting current version: %s", errTag),
			}
			sendLog(res.Reason)
			return res
		}

		v, errorMake := semver.Make("0.0.1")
		if errorMake != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable init semver: %s", errorMake),
			}
			sendLog(res.Reason)
			return res
		}

		//Send the logs
		if len(stdTagListOut.Bytes()) > 0 {
			// search for version
			lines := strings.Split(stdTagListOut.String(), "\n")
			versions := semver.Versions{}
			re := regexp.MustCompile("refs/tags/(.*)")
			for _, l := range lines {
				match := re.FindStringSubmatch(l)
				if len(match) >= 1 {
					tag := match[1]
					if sv, err := semver.Parse(tag); err == nil {
						versions = append(versions, sv)
					}
				}
			}
			semver.Sort(versions)
			if len(versions) > 0 {
				// and we increment the last version found
				v = versions[len(versions)-1]
				v.Patch++
			}
		}

		pr, errPR := semver.NewPRVersion("snapshot")
		if errPR != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable create snapshot version: %s", errTag),
			}
			sendLog(res.Reason)
			return res
		}
		v.Pre = append(v.Pre, pr)

		if cdsVersion != nil {
			v.Build = append(v.Build, cdsVersion.Value, "cds")
		}

		semverVar := sdk.Variable{
			Name:  "cds.semver",
			Type:  sdk.StringVariable,
			Value: v.String(),
		}

		if _, err := w.addVariableInPipelineBuild(semverVar, params); err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to save semver variable: %s", err),
			}
			sendLog(res.Reason)
			return res
		}

		return sdk.Result{Status: sdk.StatusSuccess.String()}
	}
}

func extractInfo(w *currentWorker, dir string, params *[]sdk.Parameter, branch, commit string, sendLog LoggerFunc) error {
	author := sdk.ParameterValue(*params, "git.author")
	authorEmail := sdk.ParameterValue(*params, "git.author.email")
	message := sdk.ParameterValue(*params, "git.message")

	info := git.ExtractInfo(dir)

	if info.GitDescribe != "" {
		gitDescribe := sdk.Variable{
			Name:  "git.describe",
			Type:  sdk.StringVariable,
			Value: info.GitDescribe,
		}

		if _, err := w.addVariableInPipelineBuild(gitDescribe, params); err != nil {
			return fmt.Errorf("Error on addVariableInPipelineBuild (describe): %s", err)
		}
		sendLog(fmt.Sprintf("git.describe: %s", info.GitDescribe))
	}

	if branch == "" || branch == "{{.git.branch}}" {
		if info.Branch != "" {
			gitBranch := sdk.Variable{
				Name:  "git.branch",
				Type:  sdk.StringVariable,
				Value: info.Branch,
			}

			if _, err := w.addVariableInPipelineBuild(gitBranch, params); err != nil {
				return fmt.Errorf("Error on addVariableInPipelineBuild (branch): %s", err)
			}
			sendLog(fmt.Sprintf("git.branch: %s", info.Branch))
		} else {
			sendLog("git.branch: [empty]")
		}
	} else {
		sendLog(fmt.Sprintf("git.branch: %s", branch))
	}

	if commit == "" || commit == "{{.git.hash}}" {
		if info.Hash != "" {
			gitHash := sdk.Variable{
				Name:  "git.hash",
				Type:  sdk.StringVariable,
				Value: info.Hash,
			}

			if _, err := w.addVariableInPipelineBuild(gitHash, params); err != nil {
				return fmt.Errorf("Error on addVariableInPipelineBuild (hash): %s", err)
			}
			sendLog(fmt.Sprintf("git.hash: %s", info.Hash))
		} else {
			sendLog("git.hash: [empty]")
		}
	} else {
		sendLog(fmt.Sprintf("git.hash: %s", commit))
	}

	if message == "" {
		if info.Message != "" {
			gitMessage := sdk.Variable{
				Name:  "git.message",
				Type:  sdk.StringVariable,
				Value: info.Message,
			}

			if _, err := w.addVariableInPipelineBuild(gitMessage, params); err != nil {
				return fmt.Errorf("Error on addVariableInPipelineBuild (message): %s", err)
			}
			sendLog(fmt.Sprintf("git.message: %s", info.Message))
		} else {
			sendLog("git.message: [empty]")
		}
	} else {
		sendLog(fmt.Sprintf("git.message: %s", message))
	}

	if author == "" {
		if info.Author != "" {
			gitAuthor := sdk.Variable{
				Name:  "git.author",
				Type:  sdk.StringVariable,
				Value: info.Author,
			}

			if _, err := w.addVariableInPipelineBuild(gitAuthor, params); err != nil {
				return fmt.Errorf("Error on addVariableInPipelineBuild (author): %s", err)
			}
			sendLog(fmt.Sprintf("git.author: %s", info.Author))
		} else {
			sendLog("git.author: [empty]")
		}
	} else {
		sendLog(fmt.Sprintf("git.author: %s", author))
	}

	if authorEmail == "" {
		if info.AuthorEmail != "" {
			gitAuthorEmail := sdk.Variable{
				Name:  "git.author.email",
				Type:  sdk.StringVariable,
				Value: info.AuthorEmail,
			}

			if _, err := w.addVariableInPipelineBuild(gitAuthorEmail, params); err != nil {
				return fmt.Errorf("Error on addVariableInPipelineBuild (authorEmail): %s", err)
			}
			sendLog(fmt.Sprintf("git.author.email: %s", info.AuthorEmail))
		} else {
			sendLog("git.author.email: [empty]")
		}
	} else {
		sendLog(fmt.Sprintf("git.author.email: %s", authorEmail))
	}
	return nil
}
