package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// HatcheryDocker spawns instances of worker model with type 'Docker'
// by directly using available docker daemon
type HatcheryDocker struct {
	sync.Mutex
	workers map[string]*exec.Cmd

	hatch *hatchery.Hatchery
}

// ParseConfig for docker mode
func (hd *HatcheryDocker) ParseConfig() {
}

// ID must returns hatchery id
func (hd *HatcheryDocker) ID() int64 {
	if hd.hatch == nil {
		return 0
	}
	return hd.hatch.ID
}

// SetWorkerModelID set the workerModelIDon each heartbeat
func (hd *HatcheryDocker) SetWorkerModelID(id int64) {}

// Mode must returns hatchery mode
func (hd *HatcheryDocker) Mode() string {
	if hd == nil {
		return ""
	}
	return DockerMode
}

//Hatchery returns hatchery instance
func (hd *HatcheryDocker) Hatchery() *hatchery.Hatchery {
	return hd.hatch
}

// CanSpawn return wether or not hatchery can spawn model
// requirement are not supported
func (hd *HatcheryDocker) CanSpawn(model *sdk.Model, req []sdk.Requirement) bool {
	if model.Type != sdk.Docker {
		return false
	}
	if len(req) > 0 {
		return false
	}
	return true
}

// Init starts cleaning routine
// and check hatchery can run in docker mode with given configuration
func (hd *HatcheryDocker) Init() error {
	hd.workers = make(map[string]*exec.Cmd)

	ok, err := checkRequirement(sdk.Requirement{Type: sdk.BinaryRequirement, Value: "docker"})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Docker not found on this host")
	}

	// Register without declaring model
	name, err := os.Hostname()
	if err != nil {
		log.Warning("Cannot retrieve hostname: %s\n", err)
		name = "cds-hatchery"
	}
	name += "-docker"
	hd.hatch = &hatchery.Hatchery{
		Name: name,
	}

	if err := register(hd.hatch); err != nil {
		log.Warning("Cannot register hatchery: %s\n", err)
	}

	go hd.workerIndexCleanupRoutine()
	go hd.killAwolWorkerRoutine()
	return nil
}

// Refresh fetch worker model status from API
// and spawn/delete workers if needed
func (hd *HatcheryDocker) Refresh() error {

	wms, err := sdk.GetWorkerModelStatus()
	if err != nil {
		return err
	}

	for _, ms := range wms {
		/* TODO: Add model type in model status ffs
			// if model is not of docker type, ignore it
		if ms.Type != sdk.Docker {
			continue
		}
		*/

		if ms.CurrentCount == ms.WantedCount {
			// ok, do nothing
			continue
		}

		m, err := sdk.GetWorkerModel(ms.ModelName)
		if err != nil {
			return fmt.Errorf("cannot get model named '%s' (%s)", ms.ModelName, err)
		}
		// if model is not of docker type, ignore it
		if m.Type != sdk.Docker {
			continue
		}

		if ms.CurrentCount < ms.WantedCount {
			diff := ms.WantedCount - ms.CurrentCount
			log.Notice("I got to spawn %d %s worker !\n", diff, ms.ModelName)
			for i := 0; i < int(diff); i++ {
				err = hd.SpawnWorker(m, ms.Requirements)
				if err != nil {
					return err
				}
			}
			continue
		}

		if ms.CurrentCount > ms.WantedCount {
			diff := ms.CurrentCount - ms.WantedCount
			log.Notice("I got to kill %d %s worker !\n", diff, ms.ModelName)
			err = killWorker(hd, m)
			if err != nil {
				return err
			}
			continue
		}

	}

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

// WorkerStarted returns the number of instances of given model started but
// not necessarily register on CDS yet
func (hd *HatcheryDocker) WorkerStarted(model *sdk.Model) int {
	var x int
	for name := range hd.workers {
		if strings.Contains(name, model.Name) {
			x++
		}
	}

	return x
}

// SpawnWorker starts a new worker in a docker container locally
func (hd *HatcheryDocker) SpawnWorker(wm *sdk.Model, req []sdk.Requirement) error {
	var err error
	uk, err = sdk.GenerateWorkerKey(sdk.FirstUseExpire)
	if err != nil {
		return fmt.Errorf("cannot generate worker key: %s", err)
	}

	if wm.Type != sdk.Docker {
		return fmt.Errorf("cannot handle %s worker model", wm.Type)
	}

	if len(hd.workers) >= maxWorker {
		return fmt.Errorf("Max capacity reached (%d)", maxWorker)
	}

	name, err := randSeq(16)
	if err != nil {
		return fmt.Errorf("cannot create worker name: %s", err)
	}
	name = wm.Name + "-" + name

	var args []string
	args = append(args, "run", "--rm", "-a", "STDOUT", "-a", "STDERR")
	//args = append(args, "run")
	args = append(args, fmt.Sprintf("--name=%s", name))
	args = append(args, "-e", "CDS_SINGLE_USE=1")
	args = append(args, "-e", fmt.Sprintf("CDS_API=%s", sdk.Host))
	args = append(args, "-e", fmt.Sprintf("CDS_NAME=%s", name))
	args = append(args, "-e", fmt.Sprintf("CDS_KEY=%s", uk))
	args = append(args, "-e", fmt.Sprintf("CDS_MODEL=%d", wm.ID))
	args = append(args, "-e", fmt.Sprintf("CDS_HATCHERY=%d", hd.hatch.ID))
	args = append(args, fmt.Sprintf("--add-host=%s", viper.GetString("docker-add-host")))
	args = append(args, wm.Image)
	args = append(args, "sh", "-c", fmt.Sprintf("rm -f worker && echo lol && curl %s/download/worker/`uname -m` -o worker && echo chmod worker && chmod +x worker && echo starting worker && ./worker", sdk.Host))

	cmd := exec.Command("docker", args...)
	//log.Debug("Running %s\n", cmd.Args)

	err = cmd.Start()
	if err != nil {
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
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	ex := hex.EncodeToString(b)
	sized := []byte(ex)[0:n]
	return string(sized), nil
}
