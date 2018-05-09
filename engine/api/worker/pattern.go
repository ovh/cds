package worker

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const patternColumns = `
	worker_model_pattern.id,
	worker_model_pattern.name,
	worker_model_pattern.type`

// InsertWorkerModelPattern insert a new worker model in database
func InsertWorkerModelPattern(db gorp.SqlExecutor, modelPattern *sdk.ModelPattern) error {
	dbmodelPattern := WorkerModelPattern(*modelPattern)
	if err := db.Insert(&dbmodelPattern); err != nil {
		return err
	}
	*modelPattern = sdk.ModelPattern(dbmodelPattern)
	return nil
}

// UpdateWorkerModelPattern insert a new worker model in database
func UpdateWorkerModelPattern(db gorp.SqlExecutor, modelPattern *sdk.ModelPattern) error {
	dbmodelPattern := WorkerModelPattern(*modelPattern)
	if _, err := db.Update(&dbmodelPattern); err != nil {
		return err
	}
	return nil
}

// DeleteWorkerModelPatter removes from database worker model pattern
func DeleteWorkerModelPattern(db gorp.SqlExecutor, ID int64) error {
	wmp := WorkerModelPattern(sdk.ModelPattern{ID: ID})
	count, err := db.Delete(&wmp)
	if err != nil {
		return err
	}
	if count == 0 {
		return sdk.ErrNotFound
	}
	return nil
}

// LoadWorkerModelPatterns retrieves model patterns from database
func LoadWorkerModelPatterns(db gorp.SqlExecutor) ([]sdk.ModelPattern, error) {
	var wmPatterns []WorkerModelPattern
	query := fmt.Sprintf(`SELECT %s from WORKER_MODEL_PATTERN ORDER BY name`, patternColumns)
	if _, err := db.Select(&wmPatterns, query); err != nil {
		return nil, sdk.WrapError(err, "LoadWorkerModelPatterns> ")
	}

	workerModelPatterns := make([]sdk.ModelPattern, len(wmPatterns))
	for i := range wmPatterns {
		if err := wmPatterns[i].PostGet(db); err != nil {
			return nil, err
		}
		workerModelPatterns[i] = sdk.ModelPattern(wmPatterns[i])
	}
	return workerModelPatterns, nil
}

// LoadWorkerModelPatternByName retrieves model patterns from database given its name and type
func LoadWorkerModelPatternByName(db gorp.SqlExecutor, patternType, name string) (*sdk.ModelPattern, error) {
	var wmp WorkerModelPattern
	query := fmt.Sprintf(`SELECT %s FROM worker_model_pattern WHERE name = $1 AND type = $2`, patternColumns)
	if err := db.SelectOne(&wmp, query, name, patternType); err != nil {
		return nil, sdk.WrapError(err, "LoadWorkerModelPatternByName> ")
	}

	if err := wmp.PostGet(db); err != nil {
		return nil, err
	}
	workerModelPattern := sdk.ModelPattern(wmp)

	return &workerModelPattern, nil
}

type patternCase struct {
	patternType  string
	patternModel sdk.ModelPattern
}

func insertFirstPatterns(db gorp.SqlExecutor) {
	preCmdOs := `#!/bin/bash
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

# Basic build binaries
cd $HOME
apt-get -y --force-yes update >> /tmp/user_data 2>&1
apt-get -y --force-yes install curl git >> /tmp/user_data 2>&1
apt-get -y --force-yes install binutils >> /tmp/user_data 2>&1

# Docker installation (FOR DEBIAN)
if [[ "x{{.FromWorkerImage}}" = "xtrue" ]]; then
  echo "$(date) - CDS_FROM_WORKER_IMAGE == true - no install docker required "
else
	# Install docker
	apt-get install -y --force-yes apt-transport-https ca-certificates >> /tmp/user_data 2>&1
	apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
	mkdir -p /etc/apt/sources.list.d
	sh -c "echo deb https://apt.dockerproject.org/repo debian-jessie main > /etc/apt/sources.list.d/docker.list"
	apt-get -y --force-yes update >> /tmp/user_data 2>&1
	apt-cache policy docker-engine >> /tmp/user_data 2>&1
	apt-get install -y --force-yes docker-engine >> /tmp/user_data 2>&1
	service docker start >> /tmp/user_data 2>&1

	# Non-root access
	groupadd docker >> /tmp/user_data 2>&1
	gpasswd -a ${USER} docker >> /tmp/user_data 2>&1
	service docker restart >> /tmp/user_data 2>&1
fi;

curl -L "{{.API}}/download/worker/linux/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C - >> /tmp/user_data 2>&1
chmod +x worker
`
	patternCases := [...]patternCase{
		{
			patternType: sdk.Docker,
			patternModel: sdk.ModelPattern{
				Type: sdk.Docker,
				Name: "basic_unix",
				Model: sdk.ModelCmds{
					Shell: "sh -c",
					Cmd:   "rm -f worker && curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --single-use --force-exit",
				},
			},
		},
		{
			patternType: sdk.Openstack,
			patternModel: sdk.ModelPattern{
				Type: sdk.Openstack,
				Name: "basic_debian",
				Model: sdk.ModelCmds{
					PreCmd:  preCmdOs,
					Cmd:     "./worker",
					PostCmd: "sudo shutdown -h now",
				},
			},
		},
		{
			patternType: sdk.VSphere,
			patternModel: sdk.ModelPattern{
				Type: sdk.VSphere,
				Name: "basic_debian",
				Model: sdk.ModelCmds{
					PreCmd:  preCmdOs,
					Cmd:     "PATH=$PATH ./worker",
					PostCmd: "sudo shutdown -h now",
				},
			},
		},
		{
			patternType: sdk.HostProcess,
			patternModel: sdk.ModelPattern{
				Type: sdk.HostProcess,
				Name: "basic_unix",
				Model: sdk.ModelCmds{
					Cmd: "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --single-use --force-exit",
				},
			},
		},
	}

	for _, pattern := range patternCases {
		numPattern, err := db.SelectInt("SELECT COUNT(1) FROM worker_model_pattern WHERE type = $1", pattern.patternType)
		if err != nil {
			log.Warning("insertFirstPatterns> cannot load worker_model_pattern for type %s", pattern.patternType, err)
			continue
		}

		if numPattern > 0 {
			continue
		}

		if err := InsertWorkerModelPattern(db, &pattern.patternModel); err != nil {
			log.Warning("insertFirstPatterns> cannot insert basic model %s : %v", pattern.patternType, err)
		}
	}
}
