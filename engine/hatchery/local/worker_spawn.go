package local

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

type localWorkerRunner struct{}

func (localWorkerRunner) NewCmd(ctx context.Context, command string, args ...string) *exec.Cmd {
	var cmd = exec.CommandContext(ctx, command, args...)
	cmd.Env = []string{}
	return cmd
}

type localWorkerLogger struct {
	name string
}

func (l localWorkerLogger) Logf(fmt string, values ...interface{}) {
	fmt = strings.TrimSuffix(fmt, "\n")
	log.Info(context.Background(), "hatchery> local> worker> %s> "+fmt, l.name)
}

func (l localWorkerLogger) Errorf(fmt string, values ...interface{}) {
	fmt = strings.TrimSuffix(fmt, "\n")
	log.Error(context.Background(), "hatchery> local> worker> %s> "+fmt, l.name)
}

func (l localWorkerLogger) Fatalf(fmt string, values ...interface{}) {
	fmt = strings.TrimSuffix(fmt, "\n")
	log.Fatalf("hatchery> local> worker> %s> "+fmt, l.name)
}

const workerCmdTmpl = "{{.WorkerBinary}} --api={{.API}} --token={{.Token}} --log-level=debug --basedir={{.BaseDir}} --name={{.Name}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --graylog-extra-key={{.GraylogExtraKey}} --graylog-extra-value={{.GraylogExtraValue}} --graylog-host={{.GraylogHost}} --graylog-port={{.GraylogPort}} --booked-workflow-job-id={{.WorkflowJobID}}"

// SpawnWorker starts a new worker process
func (h *HatcheryLocal) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	log.Debug("HatcheryLocal.SpawnWorker> %s want to spawn a worker named %s (jobID = %d)", spawnArgs.HatcheryName, spawnArgs.WorkerName, spawnArgs.JobID)

	// Generate a random string 16 chars length
	bs := make([]byte, 16)
	if _, err := rand.Read(bs); err != nil {
		return err
	}
	rndstr := hex.EncodeToString(bs)[0:16]
	basedir := path.Join(h.Config.Basedir, rndstr)
	// Create the directory
	if err := os.MkdirAll(basedir, os.FileMode(0755)); err != nil {
		return err
	}

	log.Info(ctx, "HatcheryLocal.SpawnWorker> basedir: %s", basedir)

	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Token:             spawnArgs.WorkerToken,
		BaseDir:           basedir,
		HTTPInsecure:      h.Config.API.HTTP.Insecure,
		Name:              spawnArgs.WorkerName,
		Model:             spawnArgs.ModelName(),
		HatcheryName:      h.Name(),
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
		WorkerBinary:      path.Join(h.BasedirDedicated, h.getWorkerBinaryName()),
	}

	udataParam.WorkflowJobID = spawnArgs.JobID

	tmpl, errt := template.New("cmd").Parse(workerCmdTmpl)
	if errt != nil {
		return errt
	}
	var buffer bytes.Buffer
	if errTmpl := tmpl.Execute(&buffer, udataParam); errTmpl != nil {
		return errTmpl
	}

	cmdSplitted := strings.Split(buffer.String(), " -")
	for i := range cmdSplitted[1:] {
		cmdSplitted[i+1] = "-" + strings.Trim(cmdSplitted[i+1], " ")
	}

	// Prefix the command with the directory where the worker binary has been downloaded
	log.Debug("Command exec: %v", cmdSplitted)
	var cmd *exec.Cmd
	if spawnArgs.RegisterOnly {
		cmdSplitted[0] = "register"
		cmd = h.LocalWorkerRunner.NewCmd(ctx, cmdSplitted[0], cmdSplitted...)
	} else {
		cmd = h.LocalWorkerRunner.NewCmd(ctx, cmdSplitted[0], cmdSplitted[1:]...)
	}

	cmd.Dir = udataParam.BaseDir

	// Clearenv
	env := os.Environ()
	for _, e := range env {
		if !strings.HasPrefix(e, "CDS") && !strings.HasPrefix(e, "HATCHERY") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	// Wait in a goroutine so that when process exits, Wait() update cmd.ProcessState
	go func() {
		log.Debug("hatchery> local> starting worker: %s", spawnArgs.WorkerName)
		if err := h.startCmd(spawnArgs.WorkerName, cmd, localWorkerLogger{spawnArgs.WorkerName}); err != nil {
			log.Error(ctx, "hatchery> local> %v", err)
		}
	}()

	return nil
}

func (h *HatcheryLocal) startCmd(name string, cmd *exec.Cmd, logger log.Logger) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failure due to internal error: unable to capture stdout: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failure due to internal error: unable to capture stderr: %v", err)
	}

	stdoutreader := bufio.NewReader(stdout)
	stderrreader := bufio.NewReader(stderr)

	outchan := make(chan bool)
	go func() {
		for {
			line, err := stdoutreader.ReadString('\n')
			if line != "" {
				logger.Logf(line)
			}
			if err != nil {
				stdout.Close()
				close(outchan)
				return
			}
		}
	}()

	errchan := make(chan bool)
	go func() {
		for {
			line, err := stderrreader.ReadString('\n')
			if line != "" {
				logger.Logf(line)
			}
			if err != nil {
				stderr.Close()
				close(errchan)
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start command: %v", err)
	}

	h.Lock()
	h.workers[name] = workerCmd{cmd: cmd, created: time.Now()}
	h.Unlock()

	<-outchan
	<-errchan
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command failure: %v", err)
	}

	return nil
}
