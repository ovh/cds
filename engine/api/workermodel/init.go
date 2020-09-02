package workermodel

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// Initialize worker model package.
func Initialize(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) error {
	db := DBFunc()
	return insertFirstPatterns(db)
}

func insertFirstPatterns(db gorp.SqlExecutor) error {
	preCmdOs := `#!/bin/bash
set +e
export CDS_FROM_WORKER_IMAGE={{.FromWorkerImage}}
export CDS_SINGLE_USE=1
export CDS_API={{.API}}
export CDS_TOKEN={{.Token}}
export CDS_NAME={{.Name}}
export CDS_MODEL={{.Model}}
export CDS_HATCHERY_NAME={{.HatcheryName}}
export CDS_BOOKED_WORKFLOW_JOB_ID={{.WorkflowJobID}}
export CDS_TTL={{.TTL}}
export CDS_INSECURE={{.HTTPInsecure}}
export CDS_GRAYLOG_HOST={{.GraylogHost}}
export CDS_GRAYLOG_PORT={{.GraylogPort}}
export CDS_GRAYLOG_EXTRA_KEY={{.GraylogExtraKey}}
export CDS_GRAYLOG_EXTRA_VALUE={{.GraylogExtraValue}}

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

curl -L "{{.API}}/download/worker/linux/$(uname -m)" -o worker --retry 10 --retry-max-time 120 >> /tmp/user_data 2>&1
chmod +x worker
`
	patterns := []sdk.ModelPattern{
		{
			Type: sdk.Docker,
			Name: "basic_unix",
			Model: sdk.ModelCmds{
				Shell: "sh -c",
				Cmd:   "curl {{.API}}/download/worker/linux/$(uname -m) -o worker --retry 10 --retry-max-time 120 && chmod +x worker && exec ./worker",
			},
		},
		{
			Type: sdk.Openstack,
			Name: "basic_debian",
			Model: sdk.ModelCmds{
				PreCmd:  preCmdOs,
				Cmd:     "./worker",
				PostCmd: "sudo shutdown -h now",
			},
		},
		{
			Type: sdk.VSphere,
			Name: "basic_debian",
			Model: sdk.ModelCmds{
				PreCmd:  preCmdOs,
				Cmd:     "PATH=$PATH ./worker",
				PostCmd: "sudo shutdown -h now",
			},
		},
		{
			Type: sdk.HostProcess,
			Name: "basic_unix",
			Model: sdk.ModelCmds{
				Cmd: "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --graylog-extra-key={{.GraylogExtraKey}} --graylog-extra-value={{.GraylogExtraValue}} --graylog-host={{.GraylogHost}} --graylog-port={{.GraylogPort}} --booked-workflow-job-id={{.WorkflowJobID}} --single-use --force-exit",
			},
		},
	}

	for _, pattern := range patterns {
		numPattern, err := db.SelectInt("SELECT COUNT(1) FROM worker_model_pattern WHERE type = $1", pattern.Type)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return sdk.WrapError(err, "cannot load worker_model_pattern for type %s", pattern.Type)
		}
		if numPattern > 0 {
			continue
		}
		if err := InsertPattern(db, &pattern); err != nil {
			return sdk.WrapError(err, "cannot insert basic model %s", pattern.Type)
		}
	}

	return nil
}
