package main

import (
	"container/list"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

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

func startAction(a *sdk.Action, pipBuildJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {

	// Process action build arguments
	for _, abp := range pipBuildJob.Parameters {

		// Process build variable for root action
		for j := range a.Parameters {
			if abp.Name == a.Parameters[j].Name {
				a.Parameters[j].Value = abp.Value
			}
		}
	}

	return runAction(a, pipBuildJob, stepOrder)
}

func replaceBuildVariablesPlaceholder(a *sdk.Action) {
	for i := range a.Parameters {
		for _, v := range buildVariables {
			a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value,
				"{{.cds.build."+v.Name+"}}", v.Value, -1)

		}
	}
}

func runAction(a *sdk.Action, pipBuildJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	r := sdk.Result{
		Status:  sdk.StatusFail,
		BuildID: pipBuildJob.ID,
	}

	// Replace build variable placeholder that may have been added by last step
	replaceBuildVariablesPlaceholder(a)

	if a.Type == sdk.BuiltinAction {
		return runBuiltin(a, pipBuildJob, stepOrder)
	}
	if a.Type == sdk.PluginAction {
		return runPlugin(a, pipBuildJob, stepOrder)
	}

	// Nothing to do, success !
	if len(a.Actions) == 0 {
		r.Status = sdk.StatusSuccess
		return r
	}

	var nbDisabledChildren int

	finalActions := []sdk.Action{}
	noFinalActions := []sdk.Action{}
	for _, child := range a.Actions {
		if child.Final {
			finalActions = append(finalActions, child)
		} else {
			noFinalActions = append(noFinalActions, child)
		}
	}

	var doNotRunChildrenAnymore bool
	for i, child := range noFinalActions {
		currentStep = i
		if !child.Enabled {
			childName := fmt.Sprintf("%s/%s-%d", a.Name, child.Name, i+1)

			// Update step status and continue
			if err := updateStepStatus(pipBuildJob.ID, currentStep, sdk.StatusDisabled.String()); err != nil {
				log.Printf("Cannot update step (%d) status (%s) for build %d: %s\n", currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
			}

			sendLog(pipBuildJob.ID, fmt.Sprintf("%s: Step %s is disabled\n", name, childName), pipBuildJob.PipelineBuildID, currentStep, true)
			nbDisabledChildren++
			continue
		}

		if !doNotRunChildrenAnymore {
			childName := fmt.Sprintf("%s/%s-%d", a.Name, child.Name, i+1)
			log.Printf("Running %s\n", childName)

			// Update step status
			if err := updateStepStatus(pipBuildJob.ID, currentStep, sdk.StatusBuilding.String()); err != nil {
				log.Printf("Cannot update step (%d) status (%s) for build %d: %s\n", currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
			}

			sendLog(pipBuildJob.ID, fmt.Sprintf("Starting step %s", child.Name), pipBuildJob.PipelineBuildID, currentStep, false)

			r = startAction(&child, pipBuildJob, currentStep)
			if r.Status != sdk.StatusSuccess {
				// Update step status
				if err := updateStepStatus(pipBuildJob.ID, currentStep, r.Status.String()); err != nil {
					log.Printf("Cannot update step (%d) status (%s) for build %d: %s\n", currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
				}

				log.Printf("Stopping %s at step %s", a.Name, childName)
				doNotRunChildrenAnymore = true
			}

			sendLog(pipBuildJob.ID, fmt.Sprintf("End of step %s", child.Name), pipBuildJob.PipelineBuildID, currentStep, true)

			// Update step status
			if err := updateStepStatus(pipBuildJob.ID, currentStep, sdk.StatusSuccess.String()); err != nil {
				log.Printf("Cannot update step (%d) status (%s) for build %d: %s\n", currentStep, sdk.StatusDisabled.String(), pipBuildJob.ID, err)
			}
		}
	}

	//If all steps are disabled, set action status to disabled
	if nbDisabledChildren >= (len(a.Actions) - len(finalActions)) {
		r.Status = sdk.StatusDisabled
	}

	for i, child := range finalActions {
		currentStep = len(noFinalActions) + i
		childName := fmt.Sprintf("%s/%s-%d", a.Name, child.Name, i+1)
		log.Printf("Running final action : %s\n", childName)

		sendLog(pipBuildJob.ID, fmt.Sprintf("Starting step %s", child.Name), pipBuildJob.PipelineBuildID, currentStep, false)

		finalActionResult := startAction(&child, pipBuildJob, currentStep)

		sendLog(pipBuildJob.ID, fmt.Sprintf("End of step %s", child.Name), pipBuildJob.PipelineBuildID, currentStep, true)

		//If action is success or disabled we consider final action status
		if r.Status == sdk.StatusSuccess || r.Status == sdk.StatusDisabled {
			r = finalActionResult
		}
		if finalActionResult.Status != sdk.StatusSuccess {
			log.Printf("Stoping %s at final step %s", a.Name, childName)
			return r
		}
	}

	return r
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

var logsecrets []sdk.Variable

func sendLog(pipJobID int64, value string, pipelineBuildID int64, stepOrder int, final bool) error {
	for i := range logsecrets {
		if len(logsecrets[i].Value) >= 6 {
			value = strings.Replace(value, logsecrets[i].Value, "**"+logsecrets[i].Name+"**", -1)
		}
	}

	l := sdk.NewLog(pipJobID, value, pipelineBuildID, stepOrder)
	if final {
		l.Done = time.Now()
	}
	logChan <- *l
	return nil
}

func logger(inputChan chan sdk.Log) {
	llist := list.New()

	for {
		select {
		case l, ok := <-inputChan:
			if ok {
				llist.PushBack(l)
			}
			break
		case <-time.After(1 * time.Second):

			var logs []*sdk.Log

			var currentStepLog *sdk.Log
			// While list is not empty
			for llist.Len() > 0 {
				// get older log line
				l := llist.Front().Value.(sdk.Log)
				llist.Remove(llist.Front())

				// then count how many lines are exactly the same
				count := 1
				for llist.Len() > 0 {
					n := llist.Front().Value.(sdk.Log)
					if string(n.Value) != string(l.Value) {
						break
					}
					count++
					llist.Remove(llist.Front())
				}

				// and if count > 1, then add it at the beginning of the log
				if count > 1 {
					l.Value = fmt.Sprintf("[x%d] %s %s", count, l.Value)
				}
				// and append to the logs batch
				l.Value = strings.Trim(strings.Replace(l.Value, "\n", " ", -1), " \t\n") + "\n"

				// First log
				if currentStepLog == nil {
					currentStepLog = &l
				} else {
					// Same step : concat value
					if l.StepOrder == currentStepLog.StepOrder {
						currentStepLog.Value += l.Value
						currentStepLog.LastModified = l.LastModified
						currentStepLog.Done = l.Done
					} else {
						// new Step
						logs = append(logs, currentStepLog)
						currentStepLog = &l
					}
				}

				fmt.Printf("Get %s\n", currentStepLog.Value)

			}

			// insert last step
			if currentStepLog != nil {
				logs = append(logs, currentStepLog)
			}

			if len(logs) == 0 {
				continue
			}

			for _, l := range logs {
				// Buffer log list is empty, sending batch to API
				data, err := json.Marshal(l)
				if err != nil {
					fmt.Printf("Error: cannot marshal logs: %s\n", err)
					continue
				}

				path := fmt.Sprintf("/build/%d/log", l.PipelineBuildJobID)
				_, _, err = sdk.Request("POST", path, data)
				if err != nil {
					fmt.Printf("error: cannot send logs: %s\n", err)
					continue
				}
			}
		}
	}
}

// creates a working directory in $HOME/PROJECT/APP/PIP/BN
func setupBuildDirectory(wd string) error {

	err := os.MkdirAll(wd, 0755)
	if err != nil {
		return err
	}

	err = os.Chdir(wd)
	if err != nil {
		return err
	}

	err = os.Setenv("HOME", wd)
	if err != nil {
		return err
	}

	return nil
}

// remove the buildDirectory created by setupBuildDirectory
func teardownBuildDirectory(wd string) error {

	err := os.RemoveAll(wd)
	if err != nil {
		return err
	}

	return nil
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
	err := processActionVariables(&pbji.PipelineBuildJob.Job.Action, nil, pbji.PipelineBuildJob, pbji.Secrets)
	if err != nil {
		log.Printf("takeActionBuildHandler> Cannot process action %s parameters: %s\n", pbji.PipelineBuildJob.Job.Action.Name, err)
		return sdk.Result{Status: sdk.StatusFail}
	}

	// Add secrets as string in ActionBuild.Args
	// So they can be used by plugins
	for _, s := range pbji.Secrets {
		p := sdk.Parameter{
			Type:  sdk.StringParameter,
			Name:  s.Name,
			Value: s.Value,
		}
		pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, p)
	}

	// If action is not done within 1 hour, KILL IT WITH FIRE
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
	err = setupBuildDirectory(wd)
	if err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup working directory: %s", err),
		}
	}

	// Setup user ssh keys
	err = setupSSHKey(pbji.Secrets, path.Join(wd, ".ssh"))
	if err != nil {
		time.Sleep(5 * time.Second)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}

	logsecrets = pbji.Secrets
	res := startAction(&pbji.PipelineBuildJob.Job.Action, pbji.PipelineBuildJob, 0)
	close(doneChan)
	logsecrets = nil

	err = teardownBuildDirectory(wd)
	if err != nil {
		fmt.Printf("Cannot remove build directory: %s\n", err)
	}

	fmt.Printf("Run> Done.\n")
	return res
}
