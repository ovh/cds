package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Workers need to register to main api so they can run actions
func register(cdsURI string, name string, uk string) error {
	log.Notice("Registering [%s] at [%s]\n", name, cdsURI)

	sdk.InitEndpoint(cdsURI)
	path := "/worker"

	requirements, err := sdk.GetRequirements()
	if err != nil {
		log.Warning("register> unable to get requirements")
	}

	binaryCapabilities := LoopPath(requirements)

	in := worker.RegistrationForm{
		Name:               name,
		UserKey:            uk,
		Model:              model,
		Hatchery:           hatchery,
		BinaryCapabilities: binaryCapabilities,
		Version:            VERSION,
	}

	body, err := json.MarshalIndent(in, " ", " ")
	if err != nil {
		log.Notice("register: Cannot marshal body: %s\n", err)
		return err
	}

	data, code, err := sdk.Request("POST", path, body)
	if err != nil {
		log.Notice("Cannot register worker: %s\n", err)
		return err
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
	log.Notice("Registered: %s\n", data)

	if !w.Uptodate {
		log.Warning("-=-=-=-=- Please update your worker binary -=-=-=-=-")
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
