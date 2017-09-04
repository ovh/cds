package main

import (
	"context"

	"github.com/ovh/cds/engine/api"
)

type Configuration struct {
	Log struct {
		Level string
	}
	API      api.Configuration
	Hatchery []struct{}
}

type ServiceServeOptions struct {
	SetHeaderFunc func() map[string]string
	Middlewares   []api.Middleware
}

type Service interface {
	Init(cfg interface{}) error
	Serve(ctx context.Context, opts *ServiceServeOptions) error
	CheckConfiguration(cfg interface{}) error
}
