package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// HatcheryMode describe an interface for each hatchery mode (mesos, local)
type HatcheryMode interface {
	ParseConfig()
	Init() error
	Refresh() error
	KillWorker(worker sdk.Worker) error
	SpawnWorker(model *sdk.Model, req []sdk.Requirement) error
	CanSpawn(model *sdk.Model, req []sdk.Requirement) bool
	WorkerStarted(model *sdk.Model) int
	SetWorkerModelID(int64)
	Hatchery() *hatchery.Hatchery
	ID() int64
	Mode() string
}

// Definition of different hatchery mode
const (
	LocalMode  = "local"
	DockerMode = "docker"
	SwarmMode  = "swarm"
	MesosMode  = "mesos"
	CloudMode  = "openstack"
)

var (
	uk           string
	hatcheryMode string
	maxWorker    int
	client       *HTTPClient
	api          string
)

var cmd = &cobra.Command{
	Use:   "hatchery",
	Short: "CDS Hatchery",
	Run:   func(cmd *cobra.Command, args []string) {},
}

func init() {
	flags := cmd.Flags()
	viper.SetEnvPrefix("hatchery")
	viper.AutomaticEnv()

	flags.String("mode", "", "Hatchery mode : local, docker, mesos")
	viper.BindPFlag("mode", flags.Lookup("mode"))

	flags.String("docker-add-host", "", "Start worker with a custom host-to-IP mapping (host:ip)")
	viper.BindPFlag("docker-add-host", flags.Lookup("docker-add-host"))

	flags.Int("provision", 0, "Allowed worker model provisioning")
	viper.BindPFlag("provision", flags.Lookup("provision"))
}

func main() {
	log.SetLevel(log.NoticeLevel)

	cmd.Execute()
	h := parseConfig()

	if err := h.Init(); err != nil {
		log.Critical("Init error: %s\n", err)
		os.Exit(10)
	}

	go hearbeat(h)

	for {
		time.Sleep(2 * time.Second)
		if err := hatcheryRoutine(h); err != nil {
			log.Warning("Error: %s\n", err)
		}
	}
}

func hatcheryRoutine(h HatcheryMode) error {
	wms, err := sdk.GetWorkerModelStatus()
	if err != nil {
		return err
	}

	provision := int64(viper.GetInt("provision"))

	for _, ms := range wms {
		// Provisionning
		ms.WantedCount += provision

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
				if err := h.SpawnWorker(m, ms.Requirements); err != nil {
					log.Warning("Cannot spawn %s: %s\n", ms.ModelName, err)
					continue
				}
			}
			continue
		}

		if ms.CurrentCount > ms.WantedCount {
			diff := ms.CurrentCount - ms.WantedCount
			if int(diff) < viper.GetInt("provision") { // Chill...
				continue
			}
			log.Notice("I got to kill %d %s worker !\n", diff, ms.ModelName)
			err = killWorker(h, m)
			if err != nil {
				return err
			}
			continue
		}

	}

	return nil

}

func killWorker(h HatcheryMode, model *sdk.Model) error {

	workers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	// Get list of worker for this model
	for i := range workers {
		if workers[i].Model != model.ID {
			continue
		}

		// Check if worker was spawned by this hatchery
		if workers[i].HatcheryID == 0 || workers[i].HatcheryID != h.ID() {
			continue
		}

		// If worker is not currently executing an action
		if workers[i].Status == sdk.StatusWaiting {
			// then disable him
			if err = sdk.DisableWorker(workers[i].ID); err != nil {
				return err
			}
			log.Notice("KillWorker> Disabled %s\n", workers[i].Name)
			return h.KillWorker(workers[i])
		}
	}

	return nil
}

func parseConfig() HatcheryMode {
	hatcheryMode = os.Getenv("HATCHERY_MODE")
	if hatcheryMode == "" {
		sdk.Exit("HATCHERY_MODE not provided, aborting.\n")
	}

	maxWorker = 90
	maxS := os.Getenv("CDS_MAX_WORKER")
	if maxS != "" {
		maxI, err := strconv.Atoi(maxS)
		if err != nil {
			sdk.Exit("CDS_MAX_WORKER is not a valid integer, aborting.\n")
		}
		maxWorker = maxI
	}

	if api = os.Getenv("CDS_API"); api == "" {
		sdk.Exit("CDS_API not provided, aborting.\n")
	}

	if os.Getenv("CDS_USER") == "" {
		sdk.Exit("CDS_USER not provided, aborting.\n")
	}

	if os.Getenv("CDS_PASSWORD") == "" && os.Getenv("CDS_TOKEN") == "" {
		sdk.Exit("CDS_PASSWORD or CDS_TOKEN not provided, aborting\n")
	}

	client = NewHTTPClient(os.Getenv("CDS_API"), os.Getenv("CDS_USER"), os.Getenv("CDS_PASSWORD"), os.Getenv("CDS_TOKEN"))
	sdk.SetHTTPClient(client)
	sdk.Options(os.Getenv("CDS_API"), os.Getenv("CDS_USER"), os.Getenv("CDS_PASSWORD"), os.Getenv("CDS_TOKEN"))

	var h HatcheryMode
	switch hatcheryMode {
	case LocalMode:
		h = &HatcheryLocal{}
	case DockerMode:
		h = &HatcheryDocker{}
	case MesosMode:
		h = &HatcheryMesos{}
	case CloudMode:
		h = &HatcheryCloud{}
	case SwarmMode:
		h = &HatcherySwarm{}
	default:
		sdk.Exit("Unknown hatchery mode, aborting\n")
	}

	h.ParseConfig()
	return h
}
