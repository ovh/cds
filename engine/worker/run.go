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

func (w *currentWorker) startAction(ctx context.Context, a *sdk.Action, buildID int64, params []sdk.Parameter, stepOrder int, stepName string) sdk.Result {
	// Process action build arguments
	for _, abp := range params {
		// Process build variable for root action
		for j := range a.Parameters {
			if abp.Name == a.Parameters[j].Name {
				a.Parameters[j].Value = abp.Value
			}
		}
	}
	return w.runJob(ctx, a, buildID, params, stepOrder, stepName)
}

func (w *currentWorker) replaceBuildVariablesPlaceholder(a *sdk.Action) {
	for i := range a.Parameters {
		for _, v := range w.currentJob.buildVariables {
			a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value,
				"{{.cds.build."+v.Name+"}}", v.Value, -1)
		}
	}
}

func (w *currentWorker) runJob(ctx context.Context, a *sdk.Action, buildID int64, params []sdk.Parameter, stepOrder int, stepName string) sdk.Result {
	log.Debug("runJob> start run %d stepOrder:%d", buildID, stepOrder)
	defer log.Debug("runJob> end run %d stepOrder:%d", buildID, stepOrder)
	// Replace build variable placeholder that may have been added by last step
	w.replaceBuildVariablesPlaceholder(a)
	// Set the params
	w.currentJob.params = params
	// Unset the params at the end
	defer func() {
		w.currentJob.params = nil
	}()

	if a.Type == sdk.BuiltinAction {
		return w.runBuiltin(ctx, a, buildID, params, stepOrder)
	}
	if a.Type == sdk.PluginAction {
		//Define a loggin function
		sendLog := func(s string) {
			if !strings.HasSuffix(s, "\n") {
				s += "\n"
			}
			w.sendLog(buildID, s, stepOrder, false)
		}
		return w.runPlugin(ctx, a, buildID, params, stepOrder, sendLog)
	}

	if !a.Enabled {
		return sdk.Result{
			Status:  sdk.StatusDisabled.String(),
			BuildID: buildID,
		}
	}

	// Nothing to do, success !
	if len(a.Actions) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess.String(),
			BuildID: buildID,
		}
	}

	r, nDisabled := w.runSteps(ctx, a.Actions, a, buildID, params, stepOrder, stepName, 0)
	//If all steps are disabled, set action status to disabled
	if nDisabled >= len(a.Actions) {
		r.Status = sdk.StatusDisabled.String()
	}

	return r
}

func (w *currentWorker) runSteps(ctx context.Context, steps []sdk.Action, a *sdk.Action, buildID int64, params []sdk.Parameter, stepOrder int, stepName string, stepBaseCount int) (sdk.Result, int) {
	log.Debug("runSteps> start run %d stepOrder:%d len(steps):%d", buildID, stepOrder, len(steps))
	defer log.Debug("runSteps> end run %d stepOrder:%d len(steps):%d", buildID, stepOrder, len(steps))
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
		if !child.Enabled {
			// Update step status and continue
			if err := w.updateStepStatus(buildID, w.currentJob.currentStep, sdk.StatusDisabled.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s", w.currentJob.currentStep, sdk.StatusDisabled.String(), buildID, err)
			}

			w.sendLog(buildID, fmt.Sprintf("End of Step %s [Disabled]\n", childName), w.currentJob.currentStep, true)
			nbDisabledChildren++
			continue
		}

		if !criticalStepFailed || child.AlwaysExecuted {
			log.Debug("Running %s", childName)
			// Update step status
			if err := w.updateStepStatus(buildID, w.currentJob.currentStep, sdk.StatusBuilding.String()); err != nil {
				log.Warning("Cannot update step (%d) status (%s) for build %d: %s\n", w.currentJob.currentStep, sdk.StatusDisabled.String(), buildID, err)
			}
			w.sendLog(buildID, fmt.Sprintf("Starting step %s\n", childName), w.currentJob.currentStep, false)

			r = w.startAction(ctx, &child, buildID, params, w.currentJob.currentStep, childName)
			if r.Status != sdk.StatusSuccess.String() && !child.Optional {
				criticalStepFailed = true
			}

			w.sendLog(buildID, fmt.Sprintf("End of step %s [%s]", childName, r.Status), w.currentJob.currentStep, true)

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

func (w *currentWorker) updateStepStatus(pbJobID int64, stepOrder int, status string) error {
	step := sdk.StepStatus{
		StepOrder: stepOrder,
		Status:    status,
	}
	body, errM := json.Marshal(step)
	if errM != nil {
		return errM
	}

	var path string
	if w.currentJob.wJob != nil {
		path = fmt.Sprintf("/queue/workflows/%d/step", pbJobID)
	} else {
		path = fmt.Sprintf("/build/%d/step", pbJobID)
	}

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

func workingDirectory(basedir, jobPath string) string {
	gen, _ := generateWorkingDirectory()
	return path.Join(basedir, jobPath, gen)
}

func (w *currentWorker) processJob(ctx context.Context, jobInfo *worker.WorkflowNodeJobRunInfo) sdk.Result {
	t0 := time.Now()
	defer func() { log.Info("processJob> Process Job Done (%s)", sdk.Round(time.Since(t0), time.Second).String()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	defer cancel()

	defer w.drainLogsAndCloseLogger(ctx)

	// Setup working directory
	pbJobPath := path.Join(fmt.Sprintf("%d", jobInfo.Number),
		fmt.Sprintf("%d", jobInfo.SubNumber),
		fmt.Sprintf("%s", jobInfo.NodeJobRun.Job.Action.Name))

	log.Debug("processJob> init workingDirectory basedir:%s pbJobPath:%s", w.basedir, pbJobPath)
	wd := workingDirectory(w.basedir, pbJobPath)

	log.Debug("processJob> setupBuildDirectory:%s", wd)
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
	log.Debug("processJob> call processJobParameter")
	processJobParameter(&jobInfo.NodeJobRun.Parameters, jobInfo.Secrets)

	log.Debug("processJob> call w.processActionVariables")
	if err := w.processActionVariables(&jobInfo.NodeJobRun.Job.Action, nil, jobInfo.NodeJobRun.Parameters, jobInfo.Secrets); err != nil {
		log.Warning("run> Cannot process action %s parameters: %s", jobInfo.NodeJobRun.Job.Action.Name, err)
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
	log.Debug("processJob> Setup user ssh keys")
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
	log.Debug("processJob> call w.setupSSHKey")
	if err := w.setupSSHKey(jobInfo.Secrets, keysDirectory); err != nil {
		log.Debug("processJob> call w.setupSSHKey error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}
	// DEPRECATED - END

	// The right way to go is :
	log.Debug("processJob> call vcs.SetupSSHKey")
	if err := vcs.SetupSSHKey(jobInfo.Secrets, keysDirectory, nil); err != nil {
		log.Debug("processJob> call vcs.SetupSSHKey error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup vcs ssh key (%s)", err),
		}
	}

	logsecrets = jobInfo.Secrets
	res := w.startAction(ctx, &jobInfo.NodeJobRun.Job.Action, jobInfo.NodeJobRun.ID, jobInfo.NodeJobRun.Parameters, -1, "")
	logsecrets = nil

	log.Debug("processJob> call teardownBuildDirectory wd:%s", wd)
	if err := teardownBuildDirectory(wd); err != nil {
		log.Error("Cannot remove build directory: %s", err)
	}

	log.Debug("Result: %v", res)

	return res
}

func (w *currentWorker) run(ctx context.Context, pbji *worker.PipelineBuildJobInfo) sdk.Result {
	t0 := time.Now()
	defer func() {
		log.Info("run> Run Pipeline Build Job Done (%s)", sdk.Round(time.Since(t0), time.Second).String())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	defer cancel()

	defer w.drainLogsAndCloseLogger(ctx)

	// Setup working directory
	pbJobPath := path.Join(fmt.Sprintf("%d", pbji.PipelineID),
		fmt.Sprintf("%d", pbji.PipelineBuildJob.Job.PipelineActionID),
		fmt.Sprintf("%d", pbji.BuildNumber))
	wd := workingDirectory(w.basedir, pbJobPath)

	log.Debug("run> setupBuildDirectory %s", setupBuildDirectory)
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
	log.Debug("run> processJobParameter")
	processJobParameter(&pbji.PipelineBuildJob.Parameters, pbji.Secrets)

	log.Debug("run> processActionVariables")
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
	log.Debug("run> workingDirectory w.basedir:%s pbJobPath:%s", w.basedir, pbJobPath)
	keysDirectory = workingDirectory(w.basedir, pbJobPath)
	if err := os.MkdirAll(keysDirectory, 0755); err != nil {
		log.Debug("run> error on MkdirAll %s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	// DEPRECATED - BEGIN
	log.Debug("run> w.setupSSHKey keysDirectory:%s", keysDirectory)
	if err := w.setupSSHKey(pbji.Secrets, keysDirectory); err != nil {
		log.Debug("run> error on w.setupSSHKey %s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}
	// DEPRECATED - END

	// The right way to go is :
	log.Debug("run> vcs.SetupSSHKey %s", keysDirectory)
	if err := vcs.SetupSSHKey(pbji.Secrets, keysDirectory, nil); err != nil {
		log.Debug("run> error vcs.SetupSSHKey %s", err)
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	logsecrets = pbji.Secrets

	log.Debug("run> run startAction")
	res := w.startAction(ctx, &pbji.PipelineBuildJob.Job.Action, pbji.PipelineBuildJob.ID, pbji.PipelineBuildJob.Parameters, -1, "")
	logsecrets = nil

	if err := teardownBuildDirectory(wd); err != nil {
		log.Error("Cannot remove build directory: %s", err)
	}
	return res
}
