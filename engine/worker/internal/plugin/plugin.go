package plugin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

// NewClient create a plugin client for the given plugin
func NewClient(ctx context.Context, wk workerruntime.Runtime, pluginType string, pluginName string) (Client, error) {
	// Create socket
	pluginSocket, currentPlugin, err := createGRPCPluginSocket(ctx, pluginType, pluginName, wk)
	if err != nil {
		return nil, errors.Errorf("unable to start GRPCPlugin %s: %v", pluginName, err)
	}

	// Create plugin client
	c := &client{
		w:          wk,
		socket:     pluginSocket,
		done:       make(chan struct{}),
		pluginType: pluginType,
		grpcPlugin: currentPlugin,
		pluginName: pluginName,
	}

	switch pluginType {
	case TypeAction:
		// Create grpc client
		grpcClient, err := actionplugin.Client(context.Background(), pluginSocket.Socket)
		if err != nil {
			return nil, errors.Errorf("unable to call GRPCPlugin %s: %v", pluginName, err)
		}
		qPort := actionplugin.WorkerHTTPPortQuery{Port: wk.HTTPPort()}
		if _, err := grpcClient.WorkerHTTPPort(ctx, &qPort); err != nil {
			return nil, errors.Errorf("unable to setup plugin %s with worker port: %v", pluginName, err)
		}
		c.grpcClient = grpcClient
	case TypeIntegration:
		// Create grpc client
		grpcClient, err := integrationplugin.Client(context.Background(), pluginSocket.Socket)
		if err != nil {
			return nil, errors.Errorf("unable to call GRPCPlugin %s: %v", pluginName, err)
		}
		qPort := integrationplugin.WorkerHTTPPortQuery{Port: wk.HTTPPort()}
		if _, err := grpcClient.WorkerHTTPPort(ctx, &qPort); err != nil {
			return nil, errors.Errorf("unable to setup plugin %s with worker port: %v", pluginName, err)
		}
		c.grpcClient = grpcClient
	}

	logCtx, stopLogs := context.WithCancel(ctx)
	c.stopLog = stopLogs

	// Start plugin logger
	sdk.NewGoRoutines(ctx).Run(ctx, "runGRPCPlugin", func(ctx context.Context) {
		c.enablePluginLogger(logCtx)
	})

	// Test plugin
	if err := c.Manifest(ctx); err != nil {
		c.Close(ctx)
		return nil, errors.Errorf("unable to retrieve retrieve plugin %s manifest: %v", pluginName, err)
	}

	return c, nil
}

func (c *client) Manifest(ctx context.Context) error {
	var name, version string
	switch c.pluginType {
	case TypeAction:
		m, err := c.grpcClient.(actionplugin.ActionPluginClient).Manifest(ctx, &empty.Empty{})
		if err != nil {
			return err
		}
		name, version = m.Name, m.Version
	case TypeIntegration:
		m, err := c.grpcClient.(integrationplugin.IntegrationPluginClient).Manifest(ctx, &empty.Empty{})
		if err != nil {
			return err
		}
		name, version = m.Name, m.Version
	default:
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unknown plugin of type: %s", c.pluginType)
	}
	c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Plugin %s v%s is ready", name, version))
	return nil
}

func (c *client) Run(ctx context.Context, opts map[string]string) *Result {
	// FIXME - check input only for action v2
	// inputs := c.getInputs(ctx, opts)

	if c.pluginType == TypeAction {
		return c.runActionPlugin(ctx, actionplugin.ActionQuery{Options: opts})
	}
	return c.runIntegrationPlugin(ctx, integrationplugin.RunQuery{Options: opts})
}

func (c *client) getInputs(ctx context.Context, opts map[string]string) map[string]string {
	inputs := make(map[string]string)

	// Get default value
	for k, v := range c.grpcPlugin.Inputs {
		inputs[k] = v.Default
	}

	// Override with user value
	for k := range inputs {
		if v, has := opts[k]; has {
			inputs[k] = v
		}
	}
	log.Debug(ctx, "Plugin inputs: %v", inputs)
	return inputs
}

func (c *client) runIntegrationPlugin(ctx context.Context, query integrationplugin.RunQuery) *Result {
	if c.pluginType != TypeIntegration {
		return &Result{Status: sdk.StatusFail, Details: "wrong plugin type"}
	}

	res, err := c.grpcClient.(integrationplugin.IntegrationPluginClient).Run(ctx, &query)
	if err != nil {
		res = &integrationplugin.RunResult{
			Status:  sdk.StatusFail,
			Details: fmt.Sprintf("error while running plugin: %v", err),
		}
	}
	result := &Result{Status: res.Status, Details: res.Details, Outputs: res.Outputs}

	if !strings.EqualFold(result.Status, sdk.StatusSuccess) {
		result.Status = sdk.StatusFail
	} else {
		result.Status = sdk.StatusSuccess
	}

	c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Details: %s", result.Details))
	c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Status: %s", result.Status))
	return result
}

func (c *client) runActionPlugin(ctx context.Context, query actionplugin.ActionQuery) *Result {
	if c.pluginType != TypeAction {
		return &Result{Status: sdk.StatusFail, Details: "wrong plugin type"}
	}

	if workerruntime.RunJobID(ctx) == "" {
		jobID, err := workerruntime.JobID(ctx)
		if err != nil {
			return &Result{Status: sdk.StatusFail, Details: fmt.Sprintf("Unable to retrieve job ID... Aborting (%v)", err)}
		}
		query.JobID = jobID
	}

	res, err := c.grpcClient.(actionplugin.ActionPluginClient).Run(ctx, &query)
	if err != nil {
		res = &actionplugin.ActionResult{
			Status:  sdk.StatusFail,
			Details: fmt.Sprintf("error while running plugin %s: %v", c.pluginName, err),
		}
	}
	result := &Result{Status: res.Status, Details: res.Details}
	if !strings.EqualFold(result.Status, sdk.StatusSuccess) {
		result.Status = sdk.StatusFail
	} else {
		result.Status = sdk.StatusSuccess
	}

	c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Details: %s", result.Details))
	c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Status: %s", result.Status))
	return result
}

func (c *client) Close(ctx context.Context) {
	switch c.pluginType {
	case TypeAction:
		if _, err := c.grpcClient.(actionplugin.ActionPluginClient).Stop(ctx, new(empty.Empty)); err != nil {
			// Transport is closing is a "normal" error, as we requested plugin to stop
			if !strings.Contains(err.Error(), "transport is closing") {
				log.Error(ctx, "Error on plugin.Stop: %s", err)
			}
		}
	case TypeIntegration:
		if _, err := c.grpcClient.(integrationplugin.IntegrationPluginClient).Stop(ctx, new(empty.Empty)); err != nil {
			// Transport is closing is a "normal" error, as we requested plugin to stop
			if !strings.Contains(err.Error(), "transport is closing") {
				log.Error(ctx, "Error on plugin.Stop: %s", err)
			}
		}
	}
	c.stopLog()
	<-c.done
}

func (c *client) enablePluginLogger(ctx context.Context) {
	reader := bufio.NewReader(c.socket.StdPipe)
	var accumulator string
	var shouldExit bool
	defer func() {
		if accumulator != "" {
			c.w.SendLog(ctx, workerruntime.LevelInfo, accumulator)
		}
		close(c.done)
	}()

	for {
		if ctx.Err() != nil {
			shouldExit = true
		}

		if reader.Buffered() == 0 && shouldExit {
			return
		}
		b, err := reader.ReadByte()
		if err == io.EOF {
			if shouldExit {
				return
			}
			continue
		}

		content := string(b)
		switch content {
		case "":
			continue
		case "\n":
			accumulator += content
			c.w.SendLog(ctx, workerruntime.LevelInfo, accumulator)
			accumulator = ""
			continue
		default:
			accumulator += content
			continue
		}
	}
}
