package docker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/spf13/viper"
)

var hatcheryDocker *HatcheryDocker

// HatcheryDocker spawns instances of worker model with type 'Docker'
// by directly using available docker daemon
type HatcheryDocker struct {
	sync.Mutex
	workers map[string]*exec.Cmd
	hatch   *sdk.Hatchery
	addhost string
}

// ID must returns hatchery id
func (hd *HatcheryDocker) ID() int64 {
	if hd.hatch == nil {
		return 0
	}
	return hd.hatch.ID
}

//Hatchery returns hatchery instance
func (hd *HatcheryDocker) Hatchery() *sdk.Hatchery {
	return hd.hatch
}

// ModelType returns type of hatchery
func (*HatcheryDocker) ModelType() string {
	return sdk.Docker
}

// CanSpawn return wether or not hatchery can spawn model
// requirement are not supported
func (hd *HatcheryDocker) CanSpawn(model *sdk.Model, job *sdk.PipelineBuildJob) bool {
	for _, r := range job.Job.Action.Requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			return false
		}
	}

	return true
}

// Init starts cleaning routine
// and check hatchery can run in docker mode with given configuration
func (hd *HatcheryDocker) Init() error {
	hd.workers = make(map[string]*exec.Cmd)

	ok, err := hatchery.CheckRequirement(sdk.Requirement{Type: sdk.BinaryRequirement, Value: "docker"})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Docker not found on this host")
	}

	hd.hatch = &sdk.Hatchery{
		Name: hatchery.GenerateName("docker", viper.GetString("name")),
		UID:  viper.GetString("token"),
	}

	if err := hatchery.Register(hd.hatch, viper.GetString("token")); err != nil {
		log.Warning("Cannot register hatchery: %s\n", err)
	}

	go hd.workerIndexCleanupRoutine()
	go hd.killAwolWorkerRoutine()
	return nil
}

func (hd *HatcheryDocker) workerIndexCleanupRoutine() {
	for {
		time.Sleep(1 * time.Second)
		hd.Lock()

		for name, cmd := range hd.workers {
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				log.Info("HatcheryDocker.IndexCleanup: removing exited %s\n", name)
				delete(hd.workers, name)
				break
			}
		}
		hd.Unlock()
	}
}

func (hd *HatcheryDocker) killAwolWorkerRoutine() {
	for {
		time.Sleep(5 * time.Second)
		hd.killAwolWorker()
	}
}

func (hd *HatcheryDocker) killAwolWorker() {
	apiworkers, err := sdk.GetWorkers()
	if err != nil {
		log.Warning("Cannot get workers: %s", err)
		return
	}

	hd.Lock()
	defer hd.Unlock()
	log.Info("Hatchery has %d processes in index\n", len(hd.workers))

	for name, cmd := range hd.workers {
		for _, n := range apiworkers {
			// If worker is disabled, kill it
			if n.Name == name && n.Status == sdk.StatusDisabled {
				log.Info("Worker %s is disabled. Kill it with fire !\n", name)

				// if process not killed, kill it
				if cmd.ProcessState == nil || (cmd.ProcessState != nil && !cmd.ProcessState.Exited()) {
					err = cmd.Process.Kill()
					if err != nil {
						log.Warning("HatcheryDocker.killAwolWorker: cannot kill %s: %s\n", name, err)
					}
				}

				// Remove container
				go func() {
					cmd := exec.Command("docker", "rm", "-f", name)
					err = cmd.Run()
					if err != nil {
						log.Warning("HatcheryDocker.killAwolWorker: cannot rm container %s: %s\n", name, err)
					}
				}()

				delete(hd.workers, name)
				log.Notice("HatcheryDocker.killAwolWorker> Killed disabled worker %s\n", name)
				return
			}
		}
	}
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (hd *HatcheryDocker) WorkersStarted() int {
	return len(hd.workers)
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (hd *HatcheryDocker) WorkersStartedByModel(model *sdk.Model) int {
	var x int
	for name := range hd.workers {
		if strings.Contains(name, model.Name) {
			x++
		}
	}
	return x
}

// SpawnWorker starts a new worker in a docker container locally
func (hd *HatcheryDocker) SpawnWorker(wm *sdk.Model, job *sdk.PipelineBuildJob) error {
	if wm.Type != sdk.Docker {
		return fmt.Errorf("cannot handle %s worker model", wm.Type)
	}

	if len(hd.workers) >= viper.GetInt("max-worker") {
		return fmt.Errorf("Max capacity reached (%d)", viper.GetInt("max-worker"))
	}

	name, errs := randSeq(16)
	if errs != nil {
		return fmt.Errorf("cannot create worker name: %s", errs)
	}
	name = wm.Name + "-" + name

	var args []string
	args = append(args, "run", "--rm", "-a", "STDOUT", "-a", "STDERR")
	args = append(args, fmt.Sprintf("--name=%s", name))
	args = append(args, "-e", "CDS_SINGLE_USE=1")
	args = append(args, "-e", fmt.Sprintf("CDS_API=%s", sdk.Host))
	args = append(args, "-e", fmt.Sprintf("CDS_NAME=%s", name))
	args = append(args, "-e", fmt.Sprintf("CDS_KEY=%s", viper.GetString("token")))
	args = append(args, "-e", fmt.Sprintf("CDS_MODEL=%d", wm.ID))
	args = append(args, "-e", fmt.Sprintf("CDS_HATCHERY=%d", hd.hatch.ID))

	if job != nil {
		args = append(args, "-e", fmt.Sprintf("CDS_BOOKED_JOB_ID=%d", job.ID))
	}

	if hd.addhost != "" {
		args = append(args, fmt.Sprintf("--add-host=%s", hd.addhost))
	}
	args = append(args, wm.Image)
	args = append(args, "sh", "-c", fmt.Sprintf("rm -f worker && echo 'Download worker' && curl %s/download/worker/`uname -m` -o worker && echo 'chmod worker' && chmod +x worker && echo 'starting worker' && ./worker", sdk.Host))

	cmd := exec.Command("docker", args...)
	log.Debug("Running %s\n", cmd.Args)

	if err := cmd.Start(); err != nil {
		return err
	}
	hd.Lock()
	hd.workers[name] = cmd
	hd.Unlock()

	// Wait in a goroutine so that when process exits, Wait() update cmd.ProcessState
	// ProcessState is then checked in nextAvailableLocalID
	go func() {
		cmd.Wait()
	}()

	// Do not spam docker daemon
	time.Sleep(2 * time.Second)
	return nil
}

// KillWorker stops a worker locally
func (hd *HatcheryDocker) KillWorker(worker sdk.Worker) error {
	hd.Lock()
	defer hd.Unlock()

	for name, cmd := range hd.workers {
		if worker.Name == name {
			log.Info("HatcheryDocker.KillWorker> %s\n", name)
			err := cmd.Process.Kill()
			if err != nil {
				return err
			}

			// Remove container
			cmd := exec.Command("docker", "rm", "-f", name)
			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("HatcheryDocker.KillWorker: cannot rm container %s: %s\n", name, err)
			}

			delete(hd.workers, worker.Name)
			return nil
		}
	}

	return nil
}

func randSeq(n int) (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	ex := hex.EncodeToString(b)
	sized := []byte(ex)[0:n]
	return string(sized), nil
}
