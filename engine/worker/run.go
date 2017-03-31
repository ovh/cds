package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs"
)

func processPipelineBuildJobParameter(pbJob *sdk.PipelineBuildJob, secrets []sdk.Variable) {
	for i := range pbJob.Parameters {
		keepReplacing := true
		for keepReplacing {
			t := pbJob.Parameters[i].Value

			for _, p := range pbJob.Parameters {
				pbJob.Parameters[i].Value = strings.Replace(pbJob.Parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			for _, p := range secrets {
				pbJob.Parameters[i].Value = strings.Replace(pbJob.Parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			// If parameters wasn't updated, consider it done
			if pbJob.Parameters[i].Value == t {
				keepReplacing = false
			}
		}
	}
	return
}

// ProcessActionVariables replaces all placeholders inside action recursively using
// - parent parameters
// - action build arguments
// - Secrets from project, application and environment
//
// This function should be called ONLY from worker
func processActionVariables(a *sdk.Action, parent *sdk.Action, pipBuildJob sdk.PipelineBuildJob, secrets []sdk.Variable) error {
	// replaces placeholder in parameters with ActionBuild variables
	// replaces placeholder in parameters with Parent params
	for i := range a.Parameters {
		keepReplacing := true
		for keepReplacing {
			t := a.Parameters[i].Value

			if parent != nil {
				for _, p := range parent.Parameters {
					a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
				}
			}

			for _, p := range pipBuildJob.Parameters {
				a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			for _, p := range secrets {
				a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			// If parameters wasn't updated, consider it done
			if a.Parameters[i].Value == t {
				keepReplacing = false
			}
		}
	}

	// replaces placeholder in all children recursively
	for i := range a.Actions {
		err := processActionVariables(&a.Actions[i], a, pipBuildJob, secrets)
		if err != nil {
			return nil
		}
	}

	return nil
}

func startAction(a *sdk.Action, pipBuildJob sdk.PipelineBuildJob, stepOrder int, stepName string) sdk.Result {

	// Process action build arguments
	for _, abp := range pipBuildJob.Parameters {
		// Process build variable for root action
		for j := range a.Parameters {
			if abp.Name == a.Parameters[j].Name {
				a.Parameters[j].Value = abp.Value
			}
			a.Parameters[j].Value = strings.Replace(a.Parameters[j].Value,
				"{{.cds.worker}}", pipBuildJob.Job.WorkerName, -1)
		}
	}

	return runJob(a, pipBuildJob, stepOrder, stepName)
}

func replaceBuildVariablesPlaceholder(a *sdk.Action) {
	for i := range a.Parameters {
		for _, v := range buildVariables {
			a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value,
				"{{.cds.build."+v.Name+"}}", v.Value, -1)
		}
	}
}

func runJob(a *sdk.Action, pipBuildJob sdk.PipelineBuildJob, stepOrder int, stepName string) sdk.Result {
	// Replace build variable placeholder that may have been added by last step
	replaceBuildVariablesPlaceholder(a)

	if a.Type == sdk.BuiltinAction {
		return runBuiltin(a, pipBuildJob, stepOrder)
	}
	if a.Type == sdk.PluginAction {
		return runPlugin(a, pipBuildJob, stepOrder)
	}

	if !a.Enabled {
		return sdk.Result{
			Status:  sdk.StatusDisabled,
			BuildID: pipBuildJob.ID,
		}
	}

	// Nothing to do, success !
	if len(a.Actions) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess,
			BuildID: pipBuildJob.ID,
		}
	}

	finalActions := []sdk.Action{}
	noFinalActions := []sdk.Action{}
	for _, child := range a.Actions {
		if child.Final {
			finalActions = append(finalActions, child)
		} else {
			noFinalActions = append(noFinalActions, child)
		}
	}

	r, nDisabled := runSteps(noFinalActions, a, pipBuildJob, stepOrder, stepName, 0)
	//If all steps are disabled, set action status to disabled
	if nDisabled >= (len(a.Actions) - len(finalActions)) {
		r.Status = sdk.StatusDisabled
	}

	rFinal, _ := runSteps(finalActions, a, pipBuildJob, stepOrder, stepName, len(noFinalActions))

	if r.Status == sdk.StatusFail {
		return r
	}
	return rFinal
}

func runSteps(steps []sdk.Action, a *sdk.Action, pipBuildJob sdk.PipelineBuildJob, stepOrder int, stepName string, stepBaseCount int) (sdk.Result, int) {
	var doNotRunChildrenAnymore bool
	var nbDisabledChildren int

	// Nothing to do, success !
	if len(steps) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess,
			BuildID: pipBuildJob.ID,
		}, 0
	}

	r := sdk.Result{
		Status:  sdk.StatusFail,
		BuildID: pipBuildJob.ID,
	}

	for i, child := range steps {
		if stepOrder == -1 {
			currentStep = stepBaseCount + i
		} else {
			currentStep = stepOrder
		}
		childName := fmt.Sprintf("%s/%s-%d", stepName, child.Name, i+1)
		if !child.Enabled {
			// Update step status and continue
			if err := updateStepStatus(pipBuildJob.ID, currentStep, sdk.StatusDisabled.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s\n", currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
			}

			sendLog(pipBuildJob.ID, fmt.Sprintf("End of Step %s [Disabled]\n", childName), pipBuildJob.PipelineBuildID, currentStep, true)
			nbDisabledChildren++
			continue
		}

		if !doNotRunChildrenAnymore {
			log.Debug("Running %s\n", childName)
			// Update step status
			if err := updateStepStatus(pipBuildJob.ID, currentStep, sdk.StatusBuilding.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s\n", currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
			}
			sendLog(pipBuildJob.ID, fmt.Sprintf("Starting step %s", childName), pipBuildJob.PipelineBuildID, currentStep, false)

			r = startAction(&child, pipBuildJob, currentStep, childName)
			if r.Status != sdk.StatusSuccess {
				log.Debug("Stopping %s at step %s", a.Name, childName)
				doNotRunChildrenAnymore = true
			}

			sendLog(pipBuildJob.ID, fmt.Sprintf("End of step %s [%s]", childName, r.Status.String()), pipBuildJob.PipelineBuildID, currentStep, true)

			// Update step status
			if err := updateStepStatus(pipBuildJob.ID, currentStep, r.Status.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s\n", currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
			}
		}
	}
	return r, nbDisabledChildren
}

func updateStepStatus(pbJobID int64, stepOrder int, status string) error {
	step := sdk.StepStatus{
		StepOrder: stepOrder,
		Status:    status,
	}
	body, errM := json.Marshal(step)
	if errM != nil {
		return errM
	}

	path := fmt.Sprintf("/build/%d/step", pbJobID)
	_, code, errReq := sdk.Request("POST", path, body)
	if errReq != nil {
		return errReq
	}
	if code != http.StatusOK {
		return fmt.Errorf("Wrong http code %d", code)
	}
	return nil
}

// creates a working directory in $HOME/PROJECT/APP/PIP/BN
func setupBuildDirectory(wd string) error {
	if err := os.MkdirAll(wd, 0755); err != nil {
		return err
	}

	if err := os.Chdir(wd); err != nil {
		return err
	}

	if err := os.Setenv("HOME", wd); err != nil {
		return err
	}

	return nil
}

// remove the buildDirectory created by setupBuildDirectory
func teardownBuildDirectory(wd string) error {
	return os.RemoveAll(wd)
}

func generateWorkingDirectory() (string, error) {
	size := 16
	bs := make([]byte, size)
	_, err := rand.Read(bs)
	if err != nil {
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	return string(token), nil
}

func workingDirectory(basedir string, jobInfo *worker.PipelineBuildJobInfo) string {
	gen, _ := generateWorkingDirectory()

	dir := path.Join(basedir,
		fmt.Sprintf("%d", jobInfo.PipelineID),
		fmt.Sprintf("%d", jobInfo.PipelineBuildJob.Job.PipelineActionID),
		fmt.Sprintf("%d", jobInfo.BuildNumber),
		gen)

	return dir
}

func run(pbji *worker.PipelineBuildJobInfo) sdk.Result {
	// REPLACE ALL VARIABLE EVEN SECRETS HERE
	processPipelineBuildJobParameter(&pbji.PipelineBuildJob, pbji.Secrets)

	if err := processActionVariables(&pbji.PipelineBuildJob.Job.Action, nil, pbji.PipelineBuildJob, pbji.Secrets); err != nil {
		log.Warning("run> Cannot process action %s parameters: %s\n", pbji.PipelineBuildJob.Job.Action.Name, err)
		return sdk.Result{Status: sdk.StatusFail}
	}

	// Add secrets as string or password in ActionBuild.Args
	// So they can be used by plugins
	for _, s := range pbji.Secrets {
		p := sdk.Parameter{
			Type:  s.Type,
			Name:  s.Name,
			Value: s.Value,
		}
		pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, p)
	}

	// If action is not done within 12 hour, KILL IT WITH FIRE
	doneChan := make(chan bool)
	go func() {
		for {
			select {
			case <-doneChan:
				return
			case <-time.After(12 * time.Hour):
				path := fmt.Sprintf("/queue/%d/result", pbji.PipelineBuildJob.ID)
				body, _ := json.Marshal(sdk.Result{
					Status: sdk.StatusFail,
					Reason: fmt.Sprintf("Error: Action %s running for 12 hour on worker %s, aborting", pbji.PipelineBuildJob.Job.Action.Name, name),
				})
				sdk.Request("POST", path, body)
				time.Sleep(5 * time.Second)
				os.Exit(1)
			}
		}
	}()

	// Setup working directory
	wd := workingDirectory(basedir, pbji)

	if err := setupBuildDirectory(wd); err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup working directory: %s", err),
		}
	}

	// Setup user ssh keys
	keysDirectory = workingDirectory(basedir, pbji)
	if err := os.MkdirAll(keysDirectory, 0755); err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	// DEPRECATED - BEGIN
	if err := setupSSHKey(pbji.Secrets, keysDirectory); err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}
	// DEPRECATED - END

	// The right way to go is :
	if err := vcs.SetupSSHKey(pbji.Secrets, keysDirectory, nil); err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	logsecrets = pbji.Secrets

	// add cds.worker on parameters available
	pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{Name: "cds.worker", Value: pbji.PipelineBuildJob.Job.WorkerName, Type: sdk.StringParameter})

	res := startAction(&pbji.PipelineBuildJob.Job.Action, pbji.PipelineBuildJob, -1, "")
	close(doneChan)
	logsecrets = nil

	if err := teardownBuildDirectory(wd); err != nil {
		fmt.Printf("Cannot remove build directory: %s\n", err)
	}

	fmt.Printf("run> Done.\n")
	return res
}
