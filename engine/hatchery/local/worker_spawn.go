package local

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

type localWorkerRunner struct{}

func (localWorkerRunner) NewCmd(ctx context.Context, command string, args ...string) *exec.Cmd {
	var cmd = exec.CommandContext(ctx, command, args...)
	cmd.Env = []string{}
	return cmd
}

// SpawnWorker starts a new worker process
func (h *HatcheryLocal) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	log.Debug(ctx, "HatcheryLocal.SpawnWorker> %s want to spawn a worker named %s (jobID = %d)", spawnArgs.HatcheryName, spawnArgs.WorkerName, spawnArgs.JobID)

	if spawnArgs.JobID == "0" && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

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

	workerBinary := path.Join(h.BasedirDedicated, h.getWorkerBinaryName())
	workerConfig := h.GenerateWorkerConfig(ctx, h, spawnArgs)
	workerConfig.Basedir = basedir

	// Prefix the command with the directory where the worker binary has been downloaded
	log.Debug(ctx, "Command exec: %v", workerBinary)
	var cmd *exec.Cmd
	if spawnArgs.RegisterOnly {
		cmd = h.LocalWorkerRunner.NewCmd(context.Background(), workerBinary, "register", "--config", workerConfig.EncodeBase64())
	} else {
		cmd = h.LocalWorkerRunner.NewCmd(context.Background(), workerBinary, "--config", workerConfig.EncodeBase64())
	}
	cmd.Dir = basedir

	// Clearenv
	env := os.Environ()
	for _, e := range env {
		if !strings.HasPrefix(e, "CDS") && !strings.HasPrefix(e, "HATCHERY") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	// Wait in a goroutine so that when process exits, Wait() update cmd.ProcessState
	go func() {
		log.Debug(ctx, "hatchery> local> starting worker: %s", spawnArgs.WorkerName)
		if err := h.startCmd(spawnArgs.WorkerName, cmd, localWorkerLogger{spawnArgs.WorkerName}); err != nil {
			log.Error(ctx, "hatchery> local> %v", err)
		}
	}()

	return nil
}
