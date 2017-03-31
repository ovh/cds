package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/viper"
)

// Workers need to register to main api so they can run actions
func register(cdsURI string, name string, uk string) error {
	log.Notice("Registering [%s] at [%s]\n", name, cdsURI)

	sdk.InitEndpoint(cdsURI)
	sdk.Authorization(WorkerID)

	requirements, errR := sdk.GetRequirements()
	if errR != nil {
		log.Warning("register> unable to get requirements : %s", errR)
	}

	log.Debug("Checking %d requirements", len(requirements))
	binaryCapabilities := LoopPath(requirements)

	in := worker.RegistrationForm{
		Name:               name,
		UserKey:            uk,
		Model:              model,
		Hatchery:           hatchery,
		BinaryCapabilities: binaryCapabilities,
		Version:            VERSION,
	}

	body, errM := json.Marshal(in)
	if errM != nil {
		log.Notice("register: Cannot marshal body: %s", errM)
		return errM
	}

	data, code, errR := sdk.Request("POST", "/worker", body)
	if errR != nil {
		log.Notice("Cannot register worker: %s", errR)
		return errR
	}

	if code == http.StatusUnauthorized {
		// Nothing to do here, better exit
		time.Sleep(10 * time.Second)
		sdk.Exit("register> Got HTTP %d, exiting\n", code)
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	var w sdk.Worker
	json.Unmarshal(data, &w)
	WorkerID = w.ID
	sdk.Authorization(w.ID)
	initGRPCConn()
	log.Notice("Registered: %s\n", data)

	if !w.Uptodate {
		log.Warning("-=-=-=-=- Please update your worker binary -=-=-=-=-")
	}

	return nil
}

func unregister() error {
	alive = false
	_, code, err := sdk.Request("POST", "/worker/unregister", nil)
	if err != nil {
		return err
	}
	if code > 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	if viper.GetBool("single_use") {
		if hatchery > 0 {
			log.Notice("unregister> waiting 30min to be killed by hatchery, if not killed, worker will exit")
			time.Sleep(30 * time.Minute)
		}
		log.Notice("unregister> worker will exit")
		time.Sleep(3 * time.Second)
		os.Exit(0)
	}
	return nil
}

// LoopPath return the list of evailable command in path
func LoopPath(reqs []sdk.Requirement) []string {
	binaries := []string{}
	for _, req := range reqs {
		if req.Type == sdk.BinaryRequirement {
			if b, _ := checkBinaryRequirement(req); b {
				binaries = append(binaries, req.Value)
			}
		}
	}
	return binaries
}
