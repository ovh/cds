package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/blang/semver"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runGitTag(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
<<<<<<< HEAD
		tagName := sdk.ParameterFind(&a.Parameters, "tagName")
		tagMessage := sdk.ParameterFind(&a.Parameters, "tagMessage")
		path := sdk.ParameterFind(&a.Parameters, "path")
=======
		tagName := sdk.ParameterFind(a.Parameters, "tagName")
		tagMessage := sdk.ParameterFind(a.Parameters, "tagMessage")
		path := sdk.ParameterFind(a.Parameters, "path")
>>>>>>> master

		if tagName == nil || tagName.Value == "" {
			tagName = sdk.ParameterFind(params, "cds.semver")
			if tagName == nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "Tag name is not set. Nothing to perform.",
				}
				sendLog(res.Reason)
				return res
			}
		}

<<<<<<< HEAD
		gitURL, auth, errR := extractVCSInformations(*params)
=======
		gitUrl, auth, errR := extractVCSInformations(*params)
>>>>>>> master
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

		v, errT := semver.Make(tagName.Value)
		if errT != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Tag name is not semver compatible",
			}
			sendLog(res.Reason)
			return res
		}
		v.Build = nil
		v.Pre = nil

		var userTag string
<<<<<<< HEAD
		userTrig := sdk.ParameterFind(params, "cds.triggered_by.username")
		if userTrig != nil && userTrig.Value != "" {
			userTag = userTrig.Value
		} else {
			gitAuthor := sdk.ParameterFind(params, "git.author")
=======
		userTrig := sdk.ParameterFind(*params, "cds.triggered_by.username")
		if userTrig != nil && userTrig.Value != "" {
			userTag = userTrig.Value
		} else {
			gitAuthor := sdk.ParameterFind(*params, "git.author")
>>>>>>> master
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
			Name:     v.String(),
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
<<<<<<< HEAD
		err := git.TagCreate(gitURL, auth, tagOpts, output)
=======
		err := git.TagCreate(gitUrl, auth, tagOpts, output)
>>>>>>> master

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
