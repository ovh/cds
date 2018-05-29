package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"time"

	"github.com/blang/semver"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runGitTag(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		tagPrerelease := sdk.ParameterFind(&a.Parameters, "tagPrerelease")
		tagMetadata := sdk.ParameterFind(&a.Parameters, "tagMetadata")
		tagLevel := sdk.ParameterFind(&a.Parameters, "tagLevel")
		tagMessage := sdk.ParameterFind(&a.Parameters, "tagMessage")
		path := sdk.ParameterFind(&a.Parameters, "path")

		tagLevelValid := true
		if tagLevel == nil || tagLevel.Value == "" {
			tagLevelValid = false
		} else if tagLevel.Value != "major" && tagLevel.Value != "minor" && tagLevel.Value != "patch" {
			tagLevelValid = false
		}

		if !tagLevelValid {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Tag level is mandatory. It must be: 'major' or 'minor' or 'patch'",
			}
			sendLog(res.Reason)
			return res
		}

		gitURL, auth, errR := extractVCSInformations(*params)
		if errR != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: errR.Error(),
			}
			sendLog(res.Reason)
			return res
		}

		var msg = ""
		if tagMessage != nil {
			msg = tagMessage.Value
		}

		cdsSemver := sdk.ParameterFind(params, "cds.semver")
		if cdsSemver == nil || cdsSemver.Value == "" {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("cds.semver is empty"),
			}
			sendLog(res.Reason)
			return res
		}

		smver, errT := semver.Make(cdsSemver.Value)
		if errT != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("cds.version '%s' is not semver compatible", cdsSemver.Value),
			}
			sendLog(res.Reason)
			return res
		}
		smver.Build = nil
		smver.Pre = nil

		switch tagLevel.Value {
		case "major":
			smver.Major++
			smver.Minor = 0
			smver.Patch = 0
		case "minor":
			smver.Minor++
			smver.Patch = 0
		default:
			smver.Patch++
		}

		r, _ := regexp.Compile("^([0-9A-Za-z\\-.]+)$")
		// prerelease version notes: example: alpha, rc-1, ...
		if tagPrerelease != nil && tagPrerelease.Value != "" {
			if !r.MatchString(tagPrerelease.Value) {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("tagPrerelease '%s' must comprise only ASCII alphanumerics and hyphen [0-9A-Za-z-.].", tagPrerelease.Value),
				}
				sendLog(res.Reason)
				return res
			}
			smver.Pre = []semver.PRVersion{{VersionStr: tagPrerelease.Value}}
		}

		// metadata: this content is after '+'
		if tagMetadata != nil && tagMetadata.Value != "" {
			if !r.MatchString(tagMetadata.Value) {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("tagMetadata '%s' must comprise only ASCII alphanumerics and hyphen [0-9A-Za-z-.].", tagMetadata.Value),
				}
				sendLog(res.Reason)
				return res
			}
			smver.Build = []string{tagMetadata.Value}
		}

		var userTag string
		userTrig := sdk.ParameterFind(params, "cds.triggered_by.username")
		if userTrig != nil && userTrig.Value != "" {
			userTag = userTrig.Value
		} else {
			gitAuthor := sdk.ParameterFind(params, "git.author")
			if gitAuthor != nil && gitAuthor.Value != "" {
				userTag = gitAuthor.Value
			}
		}

		if userTag == "" {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "No user find to perform tag",
			}
			sendLog(res.Reason)
			return res
		}

		//Prepare all options - tag options
		var tagOpts = &git.TagOpts{
			Message:  msg,
			Name:     smver.String(),
			Username: userTag,
		}

		if auth.SignKey.ID != "" {
			tagOpts.SignKey = auth.SignKey.Private
			tagOpts.SignID = auth.SignKey.ID

			if err := ioutil.WriteFile("pgp.pub.key", []byte(auth.SignKey.Public), 0600); err != nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "Cannot create pgp key file.",
				}
				sendLog(res.Reason)
				return res
			}
			if err := ioutil.WriteFile("pgp.key", []byte(tagOpts.SignKey), 0600); err != nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "Cannot create pgp key file.",
				}
				sendLog(res.Reason)
				return res
			}
		}

		// Run Git command

		//Prepare all options - logs
		stdErr := new(bytes.Buffer)
		stdOut := new(bytes.Buffer)

		output := &git.OutputOpts{
			Stderr: stdErr,
			Stdout: stdOut,
		}

		git.LogFunc = log.Info

		if path != nil {
			tagOpts.Path = path.Value
		}

		//Perform the git tag
		err := git.TagCreate(gitURL, auth, tagOpts, output)

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
				Reason: fmt.Sprintf("Unable to git tag: %s", err),
			}
			sendLog(res.Reason)
			return res
		}

		semverVar := sdk.Variable{
			Name:  "cds.release.version",
			Type:  sdk.StringVariable,
			Value: tagOpts.Name,
		}
		_, errV := w.addVariableInPipelineBuild(semverVar, params)
		if errV != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to save semver variable: %s", errV),
			}
			sendLog(res.Reason)
			return res
		}
		time.Sleep(5 * time.Second)
		return sdk.Result{Status: sdk.StatusSuccess.String()}
	}
}
