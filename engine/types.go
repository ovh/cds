package main

import (
	"context"

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
		Level string `default:"warning"`
	}
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
