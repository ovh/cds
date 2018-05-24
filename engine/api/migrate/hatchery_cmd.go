package migrate

import (
	"encoding/base64"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type deprecatedOpenstackModelData struct {
	Image    string `json:"os,omitempty"`
	Flavor   string `json:"flavor,omitempty"`
	UserData string `json:"user_data,omitempty"`
}

type deprecatedVSphereModelData struct {
	OS       string `json:"os"`
	UserData string `json:"user_data"` //Commands to execute when create vm model
}

// HatcheryCmdMigration useful to change worker model configuration
func HatcheryCmdMigration(store cache.Store, DBFunc func() *gorp.DbMap) {
	db := DBFunc()

	log.Info("HatcheryCmdMigration> Begin")

	wms, err := worker.LoadWorkerModels(db)
	if err != nil {
		log.Warning("HatcheryCmdMigration> Cannot load worker models : %v", err)
		return
	}

	for _, wmTmp := range wms {
		if wmTmp.ModelDocker.Cmd != "" || wmTmp.ModelVirtualMachine.Cmd != "" {
			continue
		}
		tx, errTx := db.Begin()
		if errTx != nil {
			log.Warning("HatcheryCmdMigration> cannot create a transaction : %v", errTx)
			continue
		}

		wm, errL := worker.LoadAndLockWorkerModelByID(tx, wmTmp.ID)
		if errL != nil {
			log.Warning("HatcheryCmdMigration> cannot load and lock a worker model : %v", errL)
			tx.Rollback()
			continue
		}

		switch wm.Type {
		case sdk.Docker:
			if wm.ModelDocker.Image != "" && wm.ModelDocker.Cmd != "" {
				tx.Rollback()
				continue
			}
			wm.ModelDocker = sdk.ModelDocker{
				Image: wm.Image,
				Shell: "sh -c",
				Cmd:   "curl {{.API}}/download/worker/linux/$(uname -m) -o worker --retry 10 --retry-max-time 120 -C - && chmod +x worker && exec ./worker",
			}
		case sdk.Openstack:
			var osdata deprecatedOpenstackModelData
			if wm.ModelVirtualMachine.Image != "" && wm.ModelVirtualMachine.Cmd != "" {
				tx.Rollback()
				continue
			}
			if wm.Image == "" {
				log.Warning("HatcheryCmdMigration> worker model image field is empty for %s", wm.Name)
				tx.Rollback()
				continue
			}

			if err := json.Unmarshal([]byte(wm.Image), &osdata); err != nil {
				log.Warning("HatcheryCmdMigration> cannot unmarshal image field is empty for %s : %v", wm.Name, err)
				tx.Rollback()
				continue
			}

			preCmd := `#!/bin/bash
set +e
export CDS_FROM_WORKER_IMAGE={{.FromWorkerImage}}
export CDS_SINGLE_USE=1
export CDS_FORCE_EXIT=1
export CDS_API={{.API}}
export CDS_TOKEN={{.Token}}
export CDS_NAME={{.Name}}
export CDS_MODEL={{.Model}}
export CDS_HATCHERY={{.Hatchery}}
export CDS_HATCHERY_NAME={{.HatcheryName}}
export CDS_BOOKED_PB_JOB_ID={{.PipelineBuildJobID}}
export CDS_BOOKED_WORKFLOW_JOB_ID={{.WorkflowJobID}}
export CDS_TTL={{.TTL}}
export CDS_GRAYLOG_HOST={{.GraylogHost}}
export CDS_GRAYLOG_PORT={{.GraylogPort}}
export CDS_GRAYLOG_EXTRA_KEY={{.GraylogExtraKey}}
export CDS_GRAYLOG_EXTRA_VALUE={{.GraylogExtraValue}}
#export CDS_GRPC_API={{.GrpcAPI}}
#export CDS_GRPC_INSECURE={{.GrpcInsecure}}
export CDS_INSECURE={{.HTTPInsecure}}
`
			userdata, errD := base64.StdEncoding.DecodeString(osdata.UserData)
			if errD != nil {
				log.Warning("HatcheryCmdMigration> cannot decode base64 image field for %s : %v", wm.Name, errD)
				tx.Rollback()
				continue
			}
			preCmd += string(userdata)

			preCmd += `
			curl -L "{{.API}}/download/worker/linux/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C -
			chmod +x worker
			`
			wm.ModelVirtualMachine = sdk.ModelVirtualMachine{
				Flavor:  osdata.Flavor,
				Image:   osdata.Image,
				PreCmd:  preCmd,
				Cmd:     "./worker",
				PostCmd: "sudo shutdown -h now",
			}

		case sdk.VSphere:
			var vspheredata deprecatedVSphereModelData
			if wm.ModelVirtualMachine.Image != "" && wm.ModelVirtualMachine.Cmd != "" {
				tx.Rollback()
				continue
			}
			if wm.Image == "" {
				log.Warning("HatcheryCmdMigration> worker model image field is empty for %s", wm.Name)
				tx.Rollback()
				continue
			}

			if err := json.Unmarshal([]byte(wm.Image), &vspheredata); err != nil {
				log.Warning("HatcheryCmdMigration> cannot unmarshal image field is empty for %s : %v", wm.Name, err)
				tx.Rollback()
				continue
			}

			preCmd := `#!/bin/bash
set +e
export CDS_FROM_WORKER_IMAGE={{.FromWorkerImage}}
export CDS_SINGLE_USE=1
export CDS_FORCE_EXIT=1
export CDS_API={{.API}}
export CDS_TOKEN={{.Token}}
export CDS_NAME={{.Name}}
export CDS_MODEL={{.Model}}
export CDS_HATCHERY={{.Hatchery}}
export CDS_HATCHERY_NAME={{.HatcheryName}}
export CDS_BOOKED_PB_JOB_ID={{.PipelineBuildJobID}}
export CDS_BOOKED_WORKFLOW_JOB_ID={{.WorkflowJobID}}
export CDS_TTL={{.TTL}}
export CDS_INSECURE={{.HTTPInsecure}}
export CDS_GRAYLOG_HOST={{.GraylogHost}}
export CDS_GRAYLOG_PORT={{.GraylogPort}}
export CDS_GRAYLOG_EXTRA_KEY={{.GraylogExtraKey}}
export CDS_GRAYLOG_EXTRA_VALUE={{.GraylogExtraValue}}
#export CDS_GRPC_API={{.GrpcAPI}}
#export CDS_GRPC_INSECURE={{.GrpcInsecure}}

curl -L "{{.API}}/download/worker/linux/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C -
chmod +x worker
`
			wm.ModelVirtualMachine = sdk.ModelVirtualMachine{
				Image:   vspheredata.OS,
				PreCmd:  preCmd + vspheredata.UserData,
				Cmd:     "PATH=$PATH ./worker",
				PostCmd: "shutdown -h now",
			}
		case sdk.HostProcess:
			if wm.ModelVirtualMachine.Image != "" && wm.ModelVirtualMachine.Cmd != "" {
				tx.Rollback()
				continue
			}
			wm.ModelVirtualMachine = sdk.ModelVirtualMachine{
				Image: wm.Name,
				Cmd:   "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --graylog-extra-key={{.GraylogExtraKey}} --graylog-extra-value={{.GraylogExtraValue}} --graylog-host={{.GraylogHost}} --graylog-port={{.GraylogPort}} --single-use --force-exit",
			}
		}

		if err := worker.UpdateWorkerModelWithoutRegistration(tx, *wm); err != nil {
			log.Warning("HatcheryCmdMigration> cannot update worker model %s : %v", wm.Name, err)
			tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Warning("HatcheryCmdMigration> cannot commit tx for worker model %s : %v", wm.Name, err)
			tx.Rollback()
		}
	}

	log.Info("HatcheryCmdMigration> Done")
}

// HatcheryCmdMigrationForDockerEnvs useful to change worker model configuration TO DELETE BEFORE RELEASE
func HatcheryCmdMigrationForDockerEnvs(store cache.Store, DBFunc func() *gorp.DbMap) {
	db := DBFunc()

	log.Info("HatcheryCmdMigrationForDockerEnvs> Begin")

	wms, err := worker.LoadWorkerModels(db)
	if err != nil {
		log.Warning("HatcheryCmdMigrationForDockerEnvs> Cannot load worker models : %v", err)
		return
	}

	for _, wmTmp := range wms {
		if wmTmp.Type != sdk.Docker || len(wmTmp.ModelDocker.Envs) != 0 {
			continue
		}

		tx, errTx := db.Begin()
		if errTx != nil {
			log.Warning("HatcheryCmdMigrationForDockerEnvs> cannot create a transaction : %v", errTx)
			continue
		}

		wm, errL := worker.LoadAndLockWorkerModelByID(tx, wmTmp.ID)
		if errL != nil {
			log.Warning("HatcheryCmdMigrationForDockerEnvs> cannot load and lock a worker model : %v", errL)
			tx.Rollback()
			continue
		}

		defaultEnvs := map[string]string{
			"CDS_SINGLE_USE":          "1",
			"CDS_TTL":                 "{{.TTL}}",
			"CDS_GRAYLOG_HOST":        "{{.GraylogHost}}",
			"CDS_GRAYLOG_PORT":        "{{.GraylogPort}}",
			"CDS_GRAYLOG_EXTRA_KEY":   "{{.GraylogExtraKey}}",
			"CDS_GRAYLOG_EXTRA_VALUE": "{{.GraylogExtraValue}}",
		}

		wm.ModelDocker.Envs = defaultEnvs
		wm.ModelDocker.Cmd = "curl {{.API}}/download/worker/linux/$(uname -m) -o worker --retry 10 --retry-max-time 120 -C - && chmod +x worker && exec ./worker"

		if err := worker.UpdateWorkerModelWithoutRegistration(tx, *wm); err != nil {
			log.Warning("HatcheryCmdMigrationForDockerEnvs> cannot update worker model %s : %v", wm.Name, err)
			tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Warning("HatcheryCmdMigrationForDockerEnvs> cannot commit tx for worker model %s : %v", wm.Name, err)
			tx.Rollback()
		}
	}

	log.Info("HatcheryCmdMigrationForDockerEnvs> Done")
}
