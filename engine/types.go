package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/structs"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/hatchery/docker"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/marathon"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/engine/hatchery/vsphere"
)

type Configuration struct {
	Log struct {
		Level string `default:"warning" comment:"Log Level: debug, info, warning, notice, critical"`
	} `comment:"#####################\n# CDS Logs Settings #\n#####################"`
	API      api.Configuration
	Hatchery struct {
		Docker    docker.HatcheryConfiguration
		Local     local.HatcheryConfiguration
		Marathon  marathon.HatcheryConfiguration
		Openstack openstack.HatcheryConfiguration
		Swarm     swarm.HatcheryConfiguration
		VSphere   vsphere.HatcheryConfiguration
	}
}

type ServiceServeOptions struct {
	SetHeaderFunc func() map[string]string
	Middlewares   []api.Middleware
}

type Service interface {
	ApplyConfiguration(cfg interface{}) error
	Serve(ctx context.Context) error
	CheckConfiguration(cfg interface{}) error
}

func AsEnvVariables(o interface{}, prefix string) map[string]string {
	r := map[string]string{}
	prefix = strings.ToUpper(prefix)
	fields := structs.Fields(o)
	for _, f := range fields {
		if structs.IsStruct(f.Value()) {
			rf := AsEnvVariables(f.Value(), prefix+"_"+f.Name())
			for k, v := range rf {
				r[k] = v
			}
		} else {
			r[prefix+"_"+strings.ToUpper(f.Name())] = fmt.Sprintf("%v", f.Value())
		}
	}
	return r
}
