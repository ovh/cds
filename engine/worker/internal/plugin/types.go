package plugin

import (
	"context"
	"io"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

const (
	TypeAction      = "action"
	TypeIntegration = "integration"
	TypeStream      = "stream"

	InputManagementStrict  = "strict"
	InputManagementDefault = "default"
)

type clientSocket struct {
	Socket  string
	StdPipe io.Reader
	Client  interface{}
}

type Client interface {
	Close(ctx context.Context)
	Run(ctx context.Context, opts map[string]string) *Result
	GetPostAction() *sdk.PluginPost
}

type client struct {
	socket          *clientSocket
	grpcClient      interface{}
	done            chan struct{}
	stopLog         context.CancelFunc
	w               workerruntime.Runtime
	pluginType      string
	pluginName      string
	grpcPlugin      *sdk.GRPCPlugin
	inputManagement string
	postAction      *sdk.PluginPost
}

type Result struct {
	Status  string
	Details string
	Outputs map[string]string
}

type Factory interface {
	NewClient(ctx context.Context, wk workerruntime.Runtime, pluginType string, pluginName string, inputManagement string, env map[string]string) (Client, error)
}

type PluginFactory struct{}

func (pf *PluginFactory) NewClient(ctx context.Context, wk workerruntime.Runtime, pluginType string, pluginName string, inputManagement string, env map[string]string) (Client, error) {
	return NewClient(ctx, wk, pluginType, pluginName, inputManagement, env)
}
