package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
)

func processJobParameter(params *[]sdk.Parameter, secrets []sdk.Variable) {
	parameters := *params

	for i := range parameters {
		keepReplacing := true
		for keepReplacing {
			t := parameters[i].Value

			for _, p := range parameters {
				parameters[i].Value = strings.Replace(parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			for _, p := range secrets {
				parameters[i].Value = strings.Replace(parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			// If parameters wasn't updated, consider it done
			if parameters[i].Value == t {
				keepReplacing = false
			}
		}
	}

	params = &parameters
	return
}

// ProcessActionVariables replaces all placeholders inside action recursively using
// - parent parameters
// - action build arguments
// - Secrets from project, application and environment
//
// This function should be called ONLY from worker
func (w *currentWorker) processActionVariables(a *sdk.Action, parent *sdk.Action, jobParameters []sdk.Parameter, secrets []sdk.Variable) error {
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

			for _, p := range jobParameters {
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
		if err := w.processActionVariables(&a.Actions[i], a, jobParameters, secrets); err != nil {
			return nil
		}
	}

	return nil
}

func (w *currentWorker) startAction(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, stepOrder int, stepName string) sdk.Result {
	log.Debug("startAction> Begin %p", ctx)
	defer func() {
		log.Debug("startAction> End %p (%v)", ctx, ctx.Err())
	}()
	// Process action build arguments
	for _, abp := range *params {
		// Process build variable for root action
		for j := range a.Parameters {
			if abp.Name == a.Parameters[j].Name {
				a.Parameters[j].Value = abp.Value
			}
		}
	}

	if a.Name != sdk.ScriptAction {
		// ExpandEnv over all action parameters, avoid expending "CDS_*" env variables
		var getFilteredEnv = func(s string) string {
			if strings.HasPrefix(s, "CDS_") {
				return s
			}
			return os.Getenv(s)
		}
		for i := range a.Parameters {
			a.Parameters[i].Value = os.Expand(a.Parameters[i].Value, getFilteredEnv)
		}
	}

	return w.runJob(ctx, a, buildID, params, stepOrder, stepName)
}

func (w *currentWorker) replaceVariablesPlaceholder(a *sdk.Action, params []sdk.Parameter) {
	for i := range a.Parameters {
		for _, v := range w.currentJob.buildVariables {
			a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value, "{{."+v.Name+"}}", v.Value, -1)
		}
		for _, v := range params {
			a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value, "{{."+v.Name+"}}", v.Value, -1)
		}
	}
}

func (w *currentWorker) runJob(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, stepOrder int, stepName string) sdk.Result {
	log.Info("runJob> start run %d stepOrder:%d", buildID, stepOrder)
	defer func() { log.Info("runJob> end run %d stepOrder:%d", buildID, stepOrder) }()
	// Replace variable placeholder that may have been added by last step
	w.replaceVariablesPlaceholder(a, *params)
	// Set the params
	w.currentJob.params = *params
	// Unset the params at the end
	defer func() {
		w.currentJob.params = nil
	}()

	//If the action is disabled; skip it
	if !a.Enabled {
		return sdk.Result{
			Status:  sdk.StatusDisabled.String(),
			BuildID: buildID,
		}
	}

	//If the action if a edge of the action tree; run it
	switch a.Type {
	case sdk.BuiltinAction:
		return w.runBuiltin(ctx, a, buildID, params, stepOrder)
	case sdk.PluginAction:
		//Define a loggin function
		sendLog := getLogger(w, buildID, stepOrder)
		//Run the plugin
		return w.runPlugin(ctx, a, buildID, params, stepOrder, sendLog)
	}

	// There is is no children actions (action is empty) to do, success !
	if len(a.Actions) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess.String(),
			BuildID: buildID,
		}
	}

	//Run children actions
	r, nDisabled := w.runSteps(ctx, a.Actions, a, buildID, params, stepOrder, stepName, 0)
	//If all steps are disabled, set action status to disabled
	if nDisabled >= len(a.Actions) {
		r.Status = sdk.StatusDisabled.String()
	}

	return r
}

func (w *currentWorker) runSteps(ctx context.Context, steps []sdk.Action, a *sdk.Action, buildID int64, params *[]sdk.Parameter, stepOrder int, stepName string, stepBaseCount int) (sdk.Result, int) {
	log.Info("runSteps> start run %d stepOrder:%d len(steps):%d context=%p", buildID, stepOrder, len(steps), ctx)
	defer func() {
		log.Info("runSteps> end run %d stepOrder:%d len(steps):%d context=%p (%s)", buildID, stepOrder, len(steps), ctx, ctx.Err())
	}()
	var criticalStepFailed bool
	var nbDisabledChildren int

	// Nothing to do, success !
	if len(steps) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess.String(),
			BuildID: buildID,
		}, 0
	}

	r := sdk.Result{
		Status:  sdk.StatusFail.String(),
		BuildID: buildID,
	}

	for i, child := range steps {
		if stepOrder == -1 {
			w.currentJob.currentStep = stepBaseCount + i
		} else {
			w.currentJob.currentStep = stepOrder
		}
		childName := fmt.Sprintf("%s/%s-%d", stepName, child.Name, i+1)
		if !child.Enabled || w.manualExit {
			// Update step status and continue
			if err := w.updateStepStatus(buildID, w.currentJob.currentStep, sdk.StatusDisabled.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s", w.currentJob.currentStep, sdk.StatusDisabled.String(), buildID, err)
			}

			if w.manualExit {
				w.sendLog(buildID, fmt.Sprintf("End of Step %s [Disabled - user worker exit]\n", childName), w.currentJob.currentStep, true)
			} else {
				w.sendLog(buildID, fmt.Sprintf("End of Step %s [Disabled]\n", childName), w.currentJob.currentStep, true)
			}
			nbDisabledChildren++
			continue
		}

		if !criticalStepFailed || child.AlwaysExecuted {
			// Update step status
			if err := w.updateStepStatus(buildID, w.currentJob.currentStep, sdk.StatusBuilding.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s\n", w.currentJob.currentStep, sdk.StatusDisabled.String(), buildID, err)
			}
			w.sendLog(buildID, fmt.Sprintf("Starting step %s\n", childName), w.currentJob.currentStep, false)

			r = w.startAction(ctx, &child, buildID, params, w.currentJob.currentStep, childName)
			if r.Status != sdk.StatusSuccess.String() && !child.Optional {
				criticalStepFailed = true
			}

			if r.Reason != "" {
				w.sendLog(buildID, fmt.Sprintf("End of step %s [%s] with reason: %s", childName, r.Status, r.Reason), w.currentJob.currentStep, true)
			} else {
				w.sendLog(buildID, fmt.Sprintf("End of step %s [%s]", childName, r.Status), w.currentJob.currentStep, true)
			}

			// Update step status
			if err := w.updateStepStatus(buildID, w.currentJob.currentStep, r.Status); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s", w.currentJob.currentStep, sdk.StatusDisabled.String(), buildID, err)
			}
		} else if criticalStepFailed && !child.AlwaysExecuted { // Update status of steps which are never built
			// Update step status
			if err := w.updateStepStatus(buildID, w.currentJob.currentStep, sdk.StatusNeverBuilt.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s", w.currentJob.currentStep, sdk.StatusNeverBuilt.String(), buildID, err)
			}
		}
	}

	if criticalStepFailed {
		r.Status = sdk.StatusFail.String()
	} else {
		r.Status = sdk.StatusSuccess.String()
	}

	return r, nbDisabledChildren
}

func (w *currentWorker) updateStepStatus(buildID int64, stepOrder int, status string) error {
	step := sdk.StepStatus{
		StepOrder: stepOrder,
		Status:    status,
		Start:     time.Now(),
		Done:      time.Now(),
	}
	body, errM := json.Marshal(step)
	if errM != nil {
		return errM
	}

	var path string
	if w.currentJob.wJob != nil {
		path = fmt.Sprintf("/queue/workflows/%d/step", buildID)
	} else {
		path = fmt.Sprintf("/build/%d/step", buildID)
	}

	for try := 1; try <= 10; try++ {
		log.Info("updateStepStatus> Sending step status %s buildID:%d stepOrder:%d", status, buildID, stepOrder)
		_, code, lasterr := sdk.Request("POST", path, body)
		if lasterr == nil && code < 300 {
			log.Info("updateStepStatus> Sending step status %s buildID:%d stepOrder:%d OK", status, buildID, stepOrder)
			return nil
		}
		log.Warning("updateStepStatus> Cannot send step %d result: HTTP %d err: %s - try: %d - new try in 5s", stepOrder, code, lasterr, try)
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("updateStepStatus> Could not send built result 10 times on step %d, giving up. job: %d", stepOrder, buildID)
}

// creates a working directory in $HOME/PROJECT/APP/PIP/BN
func setupBuildDirectory(wd string) error {
	if err := os.MkdirAll(wd, 0755); err != nil {
		return err
	}

	if err := os.Chdir(wd); err != nil {
		return err
	}

	return os.Setenv("HOME", wd)
}

// remove the buildDirectory created by setupBuildDirectory
func teardownBuildDirectory(wd string) error {
	return os.RemoveAll(wd)
}

func generateWorkingDirectory() (string, error) {
	size := 16
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	return string(token), nil
}

func workingDirectory(basedir, jobPath string) string {
	gen, _ := generateWorkingDirectory()
	return path.Join(basedir, jobPath, gen)
}

func (w *currentWorker) processJob(ctx context.Context, jobInfo *worker.WorkflowNodeJobRunInfo) sdk.Result {
	t0 := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 6*time.Hour)

	log.Debug("processJob> Begin %p", ctx)
	defer log.Debug("processJob> End %p", ctx)
	defer func() { log.Info("processJob> Process Job Done (%s)", sdk.Round(time.Since(t0), time.Second).String()) }()
	defer cancel()
	defer w.drainLogsAndCloseLogger(ctx)

	// Setup working directory
	pbJobPath := path.Join(fmt.Sprintf("%d", jobInfo.Number),
		fmt.Sprintf("%d", jobInfo.SubNumber),
		fmt.Sprintf("%d", jobInfo.NodeJobRun.ID),
		fmt.Sprintf("%d", jobInfo.NodeJobRun.Job.PipelineActionID))

	wd := workingDirectory(w.basedir, pbJobPath)

	if err := setupBuildDirectory(wd); err != nil {
		log.Debug("processJob> setupBuildDirectory error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup working directory: %s", err),
		}
	}

	//Add working directory as job parameter
	jobInfo.NodeJobRun.Parameters = append(jobInfo.NodeJobRun.Parameters, sdk.Parameter{
		Name:  "cds.workspace",
		Type:  sdk.StringParameter,
		Value: wd,
	})

	// add cds.worker on parameters available
	jobInfo.NodeJobRun.Parameters = append(jobInfo.NodeJobRun.Parameters, sdk.Parameter{
		Name:  "cds.worker",
		Type:  sdk.StringParameter,
		Value: jobInfo.NodeJobRun.Job.WorkerName,
	})

	// REPLACE ALL VARIABLE EVEN SECRETS HERE
	processJobParameter(&jobInfo.NodeJobRun.Parameters, jobInfo.Secrets)
	if err := w.processActionVariables(&jobInfo.NodeJobRun.Job.Action, nil, jobInfo.NodeJobRun.Parameters, jobInfo.Secrets); err != nil {
		log.Warning("processJob> Cannot process action %s parameters: %s", jobInfo.NodeJobRun.Job.Action.Name, err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot process action %s parameters", jobInfo.NodeJobRun.Job.Action.Name),
		}
	}

	// Add secrets as string or password in ActionBuild.Args
	// So they can be used by plugins
	for _, s := range jobInfo.Secrets {
		p := sdk.Parameter{
			Type:  s.Type,
			Name:  s.Name,
			Value: s.Value,
		}
		jobInfo.NodeJobRun.Parameters = append(jobInfo.NodeJobRun.Parameters, p)
	}

	// Setup user ssh keys
	keysDirectory = workingDirectory(w.basedir, pbJobPath)
	log.Debug("processJob> Setup user ssh keys - mkdir %s", keysDirectory)
	if err := os.MkdirAll(keysDirectory, 0755); err != nil {
		log.Debug("processJob> call os.MkdirAll error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup workingDirectory (%s)", err),
		}
	}

	// DEPRECATED - BEGIN
	if err := w.setupSSHKey(jobInfo.Secrets, keysDirectory); err != nil {
		log.Debug("processJob> call w.setupSSHKey error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}
	// DEPRECATED - END

	// The right way to go is :
	if err := vcs.SetupSSHKey(jobInfo.Secrets, keysDirectory, nil); err != nil {
		log.Debug("processJob> call vcs.SetupSSHKey error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup vcs ssh key (%s)", err),
		}
	}

	logsecrets = jobInfo.Secrets
	res := w.startAction(ctx, &jobInfo.NodeJobRun.Job.Action, jobInfo.NodeJobRun.ID, &jobInfo.NodeJobRun.Parameters, -1, "")
	logsecrets = nil

	if err := teardownBuildDirectory(wd); err != nil {
		log.Error("Cannot remove build directory: %s", err)
	}
	return res
}

func (w *currentWorker) run(ctx context.Context, pbji *worker.PipelineBuildJobInfo) sdk.Result {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Hour)
	defer cancel()

	log.Debug("run> Begin %p", ctx)
	defer func() {
		log.Debug("run> End %p (%v)", ctx, ctx.Err())
	}()
	t0 := time.Now()
	defer func() {
		log.Info("run> Run Pipeline Build Job Done (%s)", sdk.Round(time.Since(t0), time.Second).String())
	}()

	defer w.drainLogsAndCloseLogger(ctx)

	// Setup working directory
	pbJobPath := path.Join(fmt.Sprintf("%d", pbji.PipelineID),
		fmt.Sprintf("%d", pbji.PipelineBuildJob.Job.PipelineActionID),
		fmt.Sprintf("%d", pbji.BuildNumber))
	wd := workingDirectory(w.basedir, pbJobPath)

	if err := setupBuildDirectory(wd); err != nil {
		log.Debug("run> setupBuildDirectory error %s", err)
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

	// add cds.worker on parameters available
	pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{
		Name:  "cds.worker",
		Type:  sdk.StringParameter,
		Value: pbji.PipelineBuildJob.Job.WorkerName,
	})

	// REPLACE ALL VARIABLE EVEN SECRETS HERE
	processJobParameter(&pbji.PipelineBuildJob.Parameters, pbji.Secrets)

	if err := w.processActionVariables(&pbji.PipelineBuildJob.Job.Action, nil, pbji.PipelineBuildJob.Parameters, pbji.Secrets); err != nil {
		log.Warning("run> Cannot process action %s parameters: %s", pbji.PipelineBuildJob.Job.Action.Name, err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot process action %s parameters", pbji.PipelineBuildJob.Job.Action.Name),
		}
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
	keysDirectory = workingDirectory(w.basedir, pbJobPath)
	if err := os.MkdirAll(keysDirectory, 0755); err != nil {
		log.Debug("run> error on MkdirAll %s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	// DEPRECATED - BEGIN
	if err := w.setupSSHKey(pbji.Secrets, keysDirectory); err != nil {
		log.Debug("run> error on w.setupSSHKey %s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}
	// DEPRECATED - END

	// The right way to go is :
	if err := vcs.SetupSSHKey(pbji.Secrets, keysDirectory, nil); err != nil {
		log.Debug("run> error vcs.SetupSSHKey %s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	logsecrets = pbji.Secrets

	res := w.startAction(ctx, &pbji.PipelineBuildJob.Job.Action, pbji.PipelineBuildJob.ID, &pbji.PipelineBuildJob.Parameters, -1, "")
	logsecrets = nil

	if err := teardownBuildDirectory(wd); err != nil {
		log.Error("Cannot remove build directory: %s", err)
	}
	return res
}
