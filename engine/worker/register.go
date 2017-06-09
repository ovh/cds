package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/spf13/viper"
)

// Workers need to register to main api so they can run actions
func (w *currentWorker) register(form worker.RegistrationForm) error {
	log.Info("Registering %s on %s", form.Name, w.apiEndpoint)
	sdk.InitEndpoint(w.apiEndpoint)
	sdk.Authorization("")

	requirements, errR := sdk.GetRequirements()
	if errR != nil {
		log.Warning("register> unable to get requirements : %s", errR)
	}

	log.Debug("Checking %d requirements", len(requirements))
	form.BinaryCapabilities = LoopPath(w, requirements)
	form.Version = VERSION
	form.OS = runtime.GOOS
	form.Arch = runtime.GOOS

	WorkerID, Uptodate, err := w.client.WorkerRegister(form)
	if err != nil {
		sdk.Exit("register> Got HTTP %d, exiting\n", err)
		return err
	}

	w.id = WorkerID
	sdk.Authorization(WorkerID)
	w.initGRPCConn()
	log.Info("%s Registered on %s", form.Name, w.apiEndpoint)

	if !Uptodate {
		log.Warning("-=-=-=-=- Please update your worker binary -=-=-=-=-")
	}

	return nil
}

func (w *currentWorker) unregister() error {
	w.alive = false
	_, code, err := sdk.Request("POST", "/worker/unregister", nil)
	if err != nil {
		return err
	}
	if code > 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	if viper.GetBool("single_use") {
		if viper.GetBool("force_exit") {
			log.Info("unregister> worker will exit (force exit after register)")
			os.Exit(0)
		}
		if w.hatchery.id > 0 {
			log.Info("unregister> waiting 30min to be killed by hatchery, if not killed, worker will exit")
			time.Sleep(30 * time.Minute)
		}
		log.Info("unregister> worker will exit")
		time.Sleep(3 * time.Second)
		os.Exit(0)
	}
	return nil
}

// LoopPath return the list of evailable command in path
func LoopPath(w *currentWorker, reqs []sdk.Requirement) []string {
	binaries := []string{}
	for _, req := range reqs {
		if req.Type == sdk.BinaryRequirement {
			if b, _ := checkBinaryRequirement(w, req); b {
				binaries = append(binaries, req.Value)
			}
		}
	}
	return binaries
}
