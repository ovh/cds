package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func register(h *hatchery.Hatchery) error {
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}

	data, code, err := client.CDSRequest("POST", "/hatchery", data)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, h)
	if err != nil {
		return err
	}

	sdk.Authorization(h.UID)
	return nil
}

func checkCapabilities(req []sdk.Requirement) ([]sdk.Requirement, error) {
	var capa []sdk.Requirement
	var tmp map[string]sdk.Requirement

	tmp = make(map[string]sdk.Requirement)
	for _, r := range req {
		ok, err := checkRequirement(r)
		if err != nil {
			return nil, err
		}

		if ok {
			tmp[r.Name] = r
		}
	}

	for _, r := range tmp {
		capa = append(capa, r)
	}

	return capa, nil
}

func checkRequirement(r sdk.Requirement) (bool, error) {
	switch r.Type {
	case sdk.BinaryRequirement:
		_, err := exec.LookPath(r.Value)
		if err != nil {
			// Return nil because the error contains 'Exit status X', that's what we wanted
			return false, nil
		}
		return true, nil
	default:
		log.Warning("checkRequirement> Unknown type of requirement: %s\n", r.Type)
		return false, nil
	}
}

func hearbeat(m HatcheryMode) {
	for {
		time.Sleep(5 * time.Second)
		if m.Hatchery().ID == 0 {
			log.Notice("Disconnected from CDS engine, trying to register...\n")
			if err := register(m.Hatchery()); err != nil {
				log.Notice("Cannot register: %s\n", err)
				continue
			}
			log.Notice("Registered back: ID %d", m.Hatchery().Model.ID)
			m.SetWorkerModelID(m.Hatchery().Model.ID)
		}

		_, _, err := sdk.Request("PUT", fmt.Sprintf("/hatchery/%d", m.Hatchery().ID), nil)
		if err != nil {
			log.Notice("heartbeat> cannot refresh beat: %s\n", err)
			m.Hatchery().ID = 0
			continue
		}
		log.Info("heartbeat> done")
	}
}
