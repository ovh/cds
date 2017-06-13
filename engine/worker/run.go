package main

import (
	"context"
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
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
func (w *currentWorker) processActionVariables(a *sdk.Action, parent *sdk.Action, pipBuildJob sdk.PipelineBuildJob, secrets []sdk.Variable) error {
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
		if err := w.processActionVariables(&a.Actions[i], a, pipBuildJob, secrets); err != nil {
			return nil
		}
	}

	return nil
}

func (w *currentWorker) startAction(ctx context.Context, a *sdk.Action, pipBuildJob sdk.PipelineBuildJob, stepOrder int, stepName string) sdk.Result {
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

	return w.runJob(ctx, a, pipBuildJob, stepOrder, stepName)
}

func (w *currentWorker) replaceBuildVariablesPlaceholder(a *sdk.Action) {
	for i := range a.Parameters {
		for _, v := range w.currentJob.buildVariables {
			a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value,
				"{{.cds.build."+v.Name+"}}", v.Value, -1)
		}
	}
}

func (w *currentWorker) runJob(ctx context.Context, a *sdk.Action, pipBuildJob sdk.PipelineBuildJob, stepOrder int, stepName string) sdk.Result {
	// Replace build variable placeholder that may have been added by last step
	w.replaceBuildVariablesPlaceholder(a)

	if a.Type == sdk.BuiltinAction {
		return w.runBuiltin(ctx, a, pipBuildJob, stepOrder)
	}
	if a.Type == sdk.PluginAction {
		return w.runPlugin(ctx, a, pipBuildJob, stepOrder)
	}

	if !a.Enabled {
		return sdk.Result{
			Status:  sdk.StatusDisabled.String(),
			BuildID: pipBuildJob.ID,
		}
	}

	// Nothing to do, success !
	if len(a.Actions) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess.String(),
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

	r, nDisabled := w.runSteps(ctx, noFinalActions, a, pipBuildJob, stepOrder, stepName, 0)
	//If all steps are disabled, set action status to disabled
	if nDisabled >= (len(a.Actions) - len(finalActions)) {
		r.Status = sdk.StatusDisabled.String()
	}

	rFinal, _ := w.runSteps(ctx, finalActions, a, pipBuildJob, stepOrder, stepName, len(noFinalActions))

	if r.Status == sdk.StatusFail.String() {
		return r
	}
	return rFinal
}

func (w *currentWorker) runSteps(ctx context.Context, steps []sdk.Action, a *sdk.Action, pipBuildJob sdk.PipelineBuildJob, stepOrder int, stepName string, stepBaseCount int) (sdk.Result, int) {
	var doNotRunChildrenAnymore bool
	var nbDisabledChildren int

	// Nothing to do, success !
	if len(steps) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess.String(),
			BuildID: pipBuildJob.ID,
		}, 0
	}

	r := sdk.Result{
		Status:  sdk.StatusFail.String(),
		BuildID: pipBuildJob.ID,
	}

	for i, child := range steps {
		if stepOrder == -1 {
			w.currentJob.currentStep = stepBaseCount + i
		} else {
			w.currentJob.currentStep = stepOrder
		}
		childName := fmt.Sprintf("%s/%s-%d", stepName, child.Name, i+1)
		if !child.Enabled {
			// Update step status and continue
			if err := updateStepStatus(pipBuildJob.ID, w.currentJob.currentStep, sdk.StatusDisabled.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s", w.currentJob.currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
			}

			w.sendLog(pipBuildJob.ID, fmt.Sprintf("End of Step %s [Disabled]\n", childName), pipBuildJob.PipelineBuildID, w.currentJob.currentStep, true)
			nbDisabledChildren++
			continue
		}

		if !doNotRunChildrenAnymore {
			log.Debug("Running %s", childName)
			// Update step status
			if err := updateStepStatus(pipBuildJob.ID, w.currentJob.currentStep, sdk.StatusBuilding.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s\n", w.currentJob.currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
			}
			w.sendLog(pipBuildJob.ID, fmt.Sprintf("Starting step %s", childName), pipBuildJob.PipelineBuildID, w.currentJob.currentStep, false)

			r = w.startAction(ctx, &child, pipBuildJob, w.currentJob.currentStep, childName)
			if r.Status != sdk.StatusSuccess.String() {
				log.Debug("Stopping %s at step %s", a.Name, childName)
				doNotRunChildrenAnymore = true
			}

			w.sendLog(pipBuildJob.ID, fmt.Sprintf("End of step %s [%s]", childName, r.Status), pipBuildJob.PipelineBuildID, w.currentJob.currentStep, true)

			// Update step status
			if err := updateStepStatus(pipBuildJob.ID, w.currentJob.currentStep, r.Status); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s", w.currentJob.currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
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

func (w *currentWorker) processJob(ctx context.Context, jobInfo *worker.WorkflowNodeJobRunInfo) sdk.Result {
	return sdk.Result{}
}

func (w *currentWorker) run(ctx context.Context, pbji *worker.PipelineBuildJobInfo) sdk.Result {
	t0 := time.Now()
	defer func() { log.Info("Run Pipeline Build Job Done (%s)", sdk.Round(time.Since(t0), time.Second).String()) }()
	/*
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
		defer cancel()
	*/
	// Setup working directory
	wd := workingDirectory(w.basedir, pbji)

	if err := setupBuildDirectory(wd); err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup working directory: %s", err),
		}
	}

	//Add working directory as job parameter
	pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{
		Name:  "cds.workspace",
		Type:  sdk.StringParameter,
		Value: wd,
	})

	// REPLACE ALL VARIABLE EVEN SECRETS HERE
	processPipelineBuildJobParameter(&pbji.PipelineBuildJob, pbji.Secrets)

	if err := w.processActionVariables(&pbji.PipelineBuildJob.Job.Action, nil, pbji.PipelineBuildJob, pbji.Secrets); err != nil {
		log.Warning("run> Cannot process action %s parameters: %s", pbji.PipelineBuildJob.Job.Action.Name, err)
		return sdk.Result{Status: sdk.StatusFail.String()}
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

	// Setup user ssh keys
	keysDirectory = workingDirectory(w.basedir, pbji)
	if err := os.MkdirAll(keysDirectory, 0755); err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	// DEPRECATED - BEGIN
	if err := w.setupSSHKey(pbji.Secrets, keysDirectory); err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}
	// DEPRECATED - END

	// The right way to go is :
	if err := vcs.SetupSSHKey(pbji.Secrets, keysDirectory, nil); err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	logsecrets = pbji.Secrets

	// add cds.worker on parameters available
	pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{Name: "cds.worker", Value: pbji.PipelineBuildJob.Job.WorkerName, Type: sdk.StringParameter})

	res := w.startAction(ctx, &pbji.PipelineBuildJob.Job.Action, pbji.PipelineBuildJob, -1, "")
	logsecrets = nil

	if err := teardownBuildDirectory(wd); err != nil {
		log.Error("Cannot remove build directory: %s", err)
	}
	return res
}
