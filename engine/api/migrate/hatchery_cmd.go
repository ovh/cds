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

// HatcheryCmdMigration useful to set default tags to git.branch git.author
func HatcheryCmdMigration(store cache.Store, DBFunc func() *gorp.DbMap) {
	db := DBFunc()

	log.Info("HatcheryCmdMigration> Begin")

	wms, err := worker.LoadWorkerModels(db)
	if err != nil {
		log.Warning("HatcheryCmdMigration> Cannot load worker models : %v", err)
		return
	}

	for _, wm := range wms {
		switch wm.Type {
		case sdk.Docker:
			wm.ModelDocker = sdk.ModelDocker{
				Image: wm.Image,
				Cmd:   "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --single-use --force-exit",
			}
		case sdk.Openstack:
			var osdata deprecatedOpenstackModelData

			if wm.Image == "" {
				log.Warning("HatcheryCmdMigration> worker model image field is empty for %s", wm.Name)
				continue
			}

			if err := json.Unmarshal([]byte(wm.Image), &osdata); err != nil {
				log.Warning("HatcheryCmdMigration> cannot unmarshal image field is empty for %s : %v", wm.Name, err)
				continue
			}

			preCmd := `#!/bin/bash
set +e
export CDS_FROM_WORKER_IMAGE={{.FromWorkerImage}}
export CDS_SINGLE_USE=1
export CDS_FORCE_EXIT=1
export CDS_API={{.API}}
export CDS_TOKEN={{.Key}}
export CDS_NAME={{.Name}}
export CDS_MODEL={{.Model}}
export CDS_HATCHERY={{.Hatchery}}
export CDS_HATCHERY_NAME={{.HatcheryName}}
export CDS_BOOKED_PB_JOB_ID={{.PipelineBuildJobID}}
export CDS_BOOKED_WORKFLOW_JOB_ID={{.WorkflowJobID}}
export CDS_TTL={{.TTL}}
export CDS_INSECURE={{.HTTPInsecure}}

`
			userdata, errD := base64.StdEncoding.DecodeString(osdata.Image)
			if errD != nil {
				log.Warning("HatcheryCmdMigration> cannot decode base64 image field for %s : %v", wm.Name, errD)
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
				PreCmd:  preCmd + string(userdata),
				Cmd:     "./worker",
				PostCmd: "sudo shutdown -h now",
			}

		case sdk.VSphere:
			var vspheredata deprecatedVSphereModelData

			if wm.Image == "" {
				log.Warning("HatcheryCmdMigration> worker model image field is empty for %s", wm.Name)
				continue
			}

			if err := json.Unmarshal([]byte(wm.Image), &vspheredata); err != nil {
				log.Warning("HatcheryCmdMigration> cannot unmarshal image field is empty for %s : %v", wm.Name, err)
				continue
			}

			preCmd := `#!/bin/bash
set +e
export CDS_FROM_WORKER_IMAGE={{.FromWorkerImage}}
export CDS_SINGLE_USE=1
export CDS_FORCE_EXIT=1
export CDS_API={{.API}}
export CDS_TOKEN={{.Key}}
export CDS_NAME={{.Name}}
export CDS_MODEL={{.Model}}
export CDS_HATCHERY={{.Hatchery}}
export CDS_HATCHERY_NAME={{.HatcheryName}}
export CDS_BOOKED_PB_JOB_ID={{.PipelineBuildJobID}}
export CDS_BOOKED_WORKFLOW_JOB_ID={{.WorkflowJobID}}
export CDS_TTL={{.TTL}}
export CDS_INSECURE={{.HTTPInsecure}}

curl -L "{{.API}}/download/worker/linux/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C -
chmod +x worker
`
			wm.ModelVirtualMachine = sdk.ModelVirtualMachine{
				Image:   vspheredata.OS,
				PreCmd:  preCmd + vspheredata.UserData,
				Cmd:     "PATH=$PATH ./worker",
				PostCmd: "shutdown -h now",
			}
		case sdk.Host:
			wm.ModelVirtualMachine = sdk.ModelVirtualMachine{
				Image: wm.Name,
				Cmd:   "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --single-use --force-exit",
			}
		}

		if err := worker.UpdateWorkerModel(db, wm); err != nil {
			log.Warning("HatcheryCmdMigration> cannot update worker model %s : %v", wm.Name, err)
		}

	}

	log.Info("HatcheryCmdMigration> Done")
}
