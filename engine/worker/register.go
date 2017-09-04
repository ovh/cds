package main

import (
	"fmt"
	"runtime"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Workers need to register to main api so they can run actions
func (w *currentWorker) register(form worker.RegistrationForm) error {
	log.Info("Registering %s on %s", form.Name, w.apiEndpoint)
	sdk.InitEndpoint(w.apiEndpoint)
	sdk.Authorization("")

	requirements, errR := w.client.Requirements()
	if errR != nil {
		log.Warning("register> unable to get requirements : %s", errR)
		return errR
	}

	log.Debug("Checking %d requirements", len(requirements))
	form.BinaryCapabilities = LoopPath(w, requirements)
	form.Version = sdk.VERSION
	form.OS = runtime.GOOS
	form.Arch = runtime.GOOS

	worker, uptodate, err := w.client.WorkerRegister(form)
	if err != nil {
		sdk.Exit("register> Got HTTP %d, exiting\n", err)
		return err
	}

	w.id = worker.ID
	w.groupID = worker.GroupID
	if worker.Model != nil {
		w.model = *worker.Model
	}
	sdk.Authorization(worker.ID)
	w.initGRPCConn()
	log.Info("%s Registered on %s", form.Name, w.apiEndpoint)

	if !uptodate {
		log.Warning("-=-=-=-=- Please update your worker binary -=-=-=-=-")
	}

	return nil
}

func (w *currentWorker) unregister() error {
	log.Info("Unregistering worker")
	w.alive = false
	w.id = ""
	_, code, err := sdk.Request("POST", "/worker/unregister", nil)
	if err != nil {
		return err
	}
	if code > 300 {
		return fmt.Errorf("HTTP %d", code)
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
