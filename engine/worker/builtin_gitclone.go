package main

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runGitClone(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
		url := sdk.ParameterFind(&a.Parameters, "url")
		privateKey := sdk.ParameterFind(&a.Parameters, "privateKey")
		user := sdk.ParameterFind(&a.Parameters, "user")
		password := sdk.ParameterFind(&a.Parameters, "password")
		branch := sdk.ParameterFind(&a.Parameters, "branch")
		defaultBranch := sdk.ParameterValue(*params, "git.default_branch")
		commit := sdk.ParameterFind(&a.Parameters, "commit")
		directory := sdk.ParameterFind(&a.Parameters, "directory")

		if url == nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Git repository URL is not set. Nothing to perform.",
			}
			sendLog(res.Reason)
			return res
		}

		deprecatedKey := true

		if privateKey != nil && (strings.HasPrefix(privateKey.Value, "app-") || strings.HasPrefix(privateKey.Value, "proj-") || strings.HasPrefix(privateKey.Value, "env-")) {
			deprecatedKey = false
		}
		var key *vcs.SSHKey
		var errK error
		var privateKeyVar *sdk.Variable
		if privateKey != nil {
			privateKeyVar = sdk.VariableFind(secrets, "cds.key."+privateKey.Value+".priv")
			if deprecatedKey {
				//Setup the key
				if err := vcs.SetupSSHKeyDEPRECATED(nil, keysDirectory, privateKey); err != nil {
					res := sdk.Result{
						Status: sdk.StatusFail.String(),
						Reason: fmt.Sprintf("Unable to setup ssh key. %s", err),
					}
					sendLog(res.Reason)
					return res
				}
			} else if privateKeyVar != nil {
				// TODO: to delete after migration
				if err := vcs.SetupSSHKey(nil, keysDirectory, privateKeyVar); err != nil {
					res := sdk.Result{
						Status: sdk.StatusFail.String(),
						Reason: fmt.Sprintf("Unable to setup ssh key. %s", err),
					}
					sendLog(res.Reason)
					return res
				}
			}

			if deprecatedKey {
				//TODO: to delete
				//Get the key
				key, errK = vcs.GetSSHKeyDEPRECATED(*params, keysDirectory, privateKey)
				if errK != nil && errK != sdk.ErrKeyNotFound {
					res := sdk.Result{
						Status: sdk.StatusFail.String(),
						Reason: fmt.Sprintf("Unable to get ssh key. %s", errK),
					}
					sendLog(res.Reason)
					return res
				}
			} else {
				//Get the key
				key, errK = vcs.GetSSHKey(secrets, keysDirectory, privateKeyVar)
				if errK != nil && errK != sdk.ErrKeyNotFound {
					res := sdk.Result{
						Status: sdk.StatusFail.String(),
						Reason: fmt.Sprintf("Unable to get ssh key. %s", errK),
					}
					sendLog(res.Reason)
					return res
				}
			}
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
		var opts = &git.CloneOpts{
			Recursive:               true,
			NoStrictHostKeyChecking: true,
			Depth: 50,
		}
		if branch != nil {
			opts.Branch = branch.Value
		} else {
			opts.SingleBranch = true
		}

		// if there is no branch, check if there a defaultBranch
		if (opts.Branch == "" || opts.Branch == "{{.git.branch}}") && defaultBranch != "" {
			opts.Branch = defaultBranch
			opts.SingleBranch = false
			sendLog(fmt.Sprintf("branch is empty, using the default branch %s", defaultBranch))
		}

		r, _ := regexp.Compile("{{.*}}")
		if commit != nil && commit.Value != "" && !r.MatchString(commit.Value) {
			opts.CheckoutCommit = commit.Value
		}

		var dir string
		if directory != nil {
			dir = directory.Value
		}
		return gitClone(w, params, url.Value, dir, auth, opts, sendLog)
	}
}

func gitClone(w *currentWorker, params *[]sdk.Parameter, url string, dir string, auth *git.AuthOpts, clone *git.CloneOpts, sendLog LoggerFunc) sdk.Result {
	//Prepare all options - logs
	stdErr := new(bytes.Buffer)
	stdOut := new(bytes.Buffer)

	output := &git.OutputOpts{
		Stderr: stdErr,
		Stdout: stdOut,
	}

	git.LogFunc = log.Info

	//Perform the git clone
	err := git.Clone(url, dir, auth, clone, output)

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

	// extract info only if we git clone the same repo as current application linked to the pipeline
	gitURLSSH := sdk.ParameterValue(*params, "git.url")
	gitURLHTTP := sdk.ParameterValue(*params, "git.http_url")
	if gitURLSSH == url || gitURLHTTP == url {
		extractInfo(w, dir, params, clone.Branch, clone.CheckoutCommit, sendLog)
	}

	stdTaglistErr := new(bytes.Buffer)
	stdTagListOut := new(bytes.Buffer)
	outputGitTag := &git.OutputOpts{
		Stderr: stdTaglistErr,
		Stdout: stdTagListOut,
	}

	errTag := git.TagList(url, dir, auth, outputGitTag)

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

	return sdk.Result{Status: sdk.StatusSuccess.String()}
}

func extractInfo(w *currentWorker, dir string, params *[]sdk.Parameter, branch, commit string, sendLog LoggerFunc) error {
	author := sdk.ParameterValue(*params, "git.author")
	authorEmail := sdk.ParameterValue(*params, "git.author.email")
	message := sdk.ParameterValue(*params, "git.message")

	info := git.ExtractInfo(dir)

	cdsVersion := sdk.ParameterFind(params, "cds.version")
	if cdsVersion == nil || cdsVersion.Value == "" {
		return fmt.Errorf("cds.version is empty")
	}

	var cdsSemver string
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

		smver, errT := semver.Make(info.GitDescribe)
		if errT != nil {
			sendLog(fmt.Sprintf("!! WARNING !! git describe %s is not semver compatible, we can't create cds.semver variable", info.GitDescribe))
		} else {
			// Prerelease versions
			// for 0.31.1-4-g595de235a, smver.Pre = 4-g595de235a
			if len(smver.Pre) == 1 {
				tuple := strings.Split(smver.Pre[0].String(), "-")
				// we split 4-g595de235a, g595de235a is the sha1
				if len(tuple) == 2 {
					cdsSemver = fmt.Sprintf("%d.%d.%d-%s+sha.%s.cds.%s",
						smver.Major,
						smver.Minor,
						smver.Patch,
						tuple[0],
						tuple[1],
						cdsVersion.Value,
					)
				}
			}
		}

		if cdsSemver == "" {
			// here, there is no prerelease version, it's a tag
			cdsSemver = fmt.Sprintf("%d.%d.%d+cds.%s",
				smver.Major,
				smver.Minor,
				smver.Patch,
				cdsVersion.Value,
			)
		}
	} else {
		// default value if there is no tag on repository
		cdsSemver = fmt.Sprintf("0.0.1+cds.%s", cdsVersion.Value)
	}

	if cdsSemver != "" {
		semverVar := sdk.Variable{
			Name:  "cds.semver",
			Type:  sdk.StringVariable,
			Value: cdsSemver,
		}

		if _, err := w.addVariableInPipelineBuild(semverVar, params); err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to save semver variable: %s", err),
			}
			sendLog(res.Reason)
		}
		sendLog(fmt.Sprintf("cds.semver: %s", cdsSemver))
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
