package hatchery

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Interface describe an interface for each hatchery mode (mesos, local)
type Interface interface {
	Init() error
	KillWorker(worker sdk.Worker) error
	SpawnWorker(model *sdk.Model, req []sdk.Requirement) error
	CanSpawn(model *sdk.Model, req []sdk.Requirement) bool
	WorkerStarted(model *sdk.Model) int
	Hatchery() *sdk.Hatchery
	ID() int64
}

var (
	// Client is a CDS Client
	Client sdk.HTTPClient
)

// Born creates hatchery
func Born(h Interface, api, token string, provision int, requestSecondsTimeout int, insecureSkipVerifyTLS bool) {
	Client = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout:  time.Duration(requestSecondsTimeout) * time.Second,
			MaxTries:        5,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerifyTLS},
		},
	}

	sdk.SetHTTPClient(Client)
	// No user / password, only token used for auth hatchery
	sdk.Options(api, "", "", token)

	if err := h.Init(); err != nil {
		log.Critical("Init error: %s\n", err)
		os.Exit(10)
	}

	go hearbeat(h, token)

	for {
		time.Sleep(2 * time.Second)
		if h.Hatchery() == nil || h.Hatchery().ID == 0 {
			log.Debug("Born> continue")
			continue
		}

		if err := hatcheryRoutine(h, provision); err != nil {
			log.Warning("Born> Error: %s\n", err)
		}
	}
}

func hatcheryRoutine(h Interface, provision int) error {
	wms, err := sdk.GetWorkerModelStatus()
	if err != nil {
		log.Debug("hatcheryRoutine> err while GetWorkerModelStatus:%e\n", err)
		return err
	}

	if len(wms) == 0 {
		log.Warning("hatcheryRoutine> No model from GetWorkerModelStatus")
	}

	var sumProvisionning int

	for _, ms := range wms {
		// Provisionning
		ms.WantedCount += int64(provision)
		sumProvisionning += int(ms.WantedCount)

		if ms.CurrentCount == ms.WantedCount {
			// ok, do nothing
			continue
		}
		m, err := sdk.GetWorkerModel(ms.ModelName)
		if err != nil {
			return fmt.Errorf("cannot get model named '%s' (%s)", ms.ModelName, err)
		}

		if !h.CanSpawn(m, ms.Requirements) {
			continue
		}

		log.Debug("hatcheryRoutine> CurrentCount=%d WantedCount=%d BuildingCount=%d Requirements=%v", ms.CurrentCount, ms.WantedCount, ms.BuildingCount, ms.Requirements)

		if ms.CurrentCount < ms.WantedCount {
			diff := ms.WantedCount - ms.CurrentCount
			// Check the number of worker started by hatchery
			if ms.WantedCount < int64(h.WorkerStarted(m))-ms.BuildingCount {
				// Ok so they are starting...
				log.Notice("%d wanted, but %d (%d building) %s workers started already...\n", ms.WantedCount, h.WorkerStarted(m), ms.BuildingCount, ms.ModelName)
				continue
			}
			log.Notice("I got to spawn %d %s worker ! (%d/%d)\n", diff, ms.ModelName, ms.CurrentCount, ms.WantedCount)

			for i := 0; i < int(diff); i++ {
				if errSpawn := h.SpawnWorker(m, ms.Requirements); errSpawn != nil {
					log.Warning("Cannot spawn %s: %s\n", ms.ModelName, errSpawn)
					continue
				}
			}
			continue
		}

		if ms.CurrentCount > ms.WantedCount {
			diff := ms.CurrentCount - ms.WantedCount
			if int(diff) < provision { // Chill...
				continue
			}
			log.Notice("I got to kill %d %s worker !\n", diff, ms.ModelName)

			if err := killWorker(h, m); err != nil {
				log.Warning("hatcheryRoutine> Unable to kill worker %s", ms.ModelName)
				return err
			}
			continue
		}
	}

	if sumProvisionning == 0 {
		log.Warning("hatcheryRoutine> Nothing to provision")
	}

	return nil
}

// Register calls CDS API to register current hatchery
func Register(h *sdk.Hatchery, token string) error {

	log.Notice("Register Hatchery %s\n", h.Name)

	h.UID = token
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}

	data, code, err := sdk.Request("POST", "/hatchery", data)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("Register> HTTP %d", code)
	}

	if err = json.Unmarshal(data, h); err != nil {
		return err
	}

	sdk.Authorization(h.UID)
	return nil
}

// CheckRequirement checks binary requirement in path
func CheckRequirement(r sdk.Requirement) (bool, error) {
	switch r.Type {
	case sdk.BinaryRequirement:
		_, err := exec.LookPath(r.Value)
		if err != nil {
			// Return nil because the error contains 'Exit status X', that's what we wanted
			return false, nil
		}
		return true, nil
	default:
		return false, nil
	}
}

func hearbeat(m Interface, token string) {
	for {
		time.Sleep(5 * time.Second)
		if m.Hatchery().ID == 0 {
			log.Notice("Disconnected from CDS engine, trying to register...\n")
			if err := Register(m.Hatchery(), token); err != nil {
				log.Notice("Cannot register: %s\n", err)
				continue
			}
			log.Notice("Registered back: ID %d", m.Hatchery().Model.ID)
		}

		_, _, err := sdk.Request("PUT", fmt.Sprintf("/hatchery/%d", m.Hatchery().ID), nil)
		if err != nil {
			log.Notice("heartbeat> cannot refresh beat: %s\n", err)
			m.Hatchery().ID = 0
			continue
		}
	}
}
