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
func NewClient(ctx context.Context, wk workerruntime.Runtime, pluginType string, pluginName string, inputManagement string, env map[string]string) (Client, error) {
	// Create socket
	pluginSocket, currentPlugin, err := createGRPCPluginSocket(ctx, pluginType, pluginName, wk, env)
	if err != nil {
		return nil, errors.Errorf("unable to start GRPCPlugin %s: %v", pluginName, err)
	}

	// Create plugin client
	c := &client{
		w:               wk,
		socket:          pluginSocket,
		done:            make(chan struct{}),
		pluginType:      pluginType,
		grpcPlugin:      currentPlugin,
		pluginName:      pluginName,
		inputManagement: inputManagement,
	}
	if currentPlugin.Post.Plugin != "" {
		c.postAction = &currentPlugin.Post
	}

	switch pluginType {
	case TypeAction, TypeStream:
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
	case TypeAction, TypeStream:
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
	log.Debug(ctx, "# Plugin %s v%s is ready", name, version)
	return nil
}

func (c *client) GetPostAction() *sdk.PluginPost {
	return c.postAction
}

func (c *client) Run(ctx context.Context, opts map[string]string) *Result {
	var inputs map[string]string
	if c.inputManagement == InputManagementStrict {
		var err error
		inputs = c.getInputs(ctx, opts)
		if err != nil {
			return &Result{
				Status:  sdk.StatusFail,
				Details: fmt.Sprintf("unable to interpolate secrets: %v", err),
			}
		}
	} else {
		inputs = opts
	}

	switch c.pluginType {
	case TypeStream:
		return c.runStreamActionPlugin(ctx, actionplugin.ActionQuery{Options: inputs})
	case TypeAction:
		return c.runActionPlugin(ctx, actionplugin.ActionQuery{Options: inputs})
	default:
		return c.runIntegrationPlugin(ctx, integrationplugin.RunQuery{Options: inputs})
	}

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
		c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Details: %s", result.Details))
	} else {
		result.Status = sdk.StatusSuccess
	}
	c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Status: %s", result.Status))
	return result
}

func (c *client) runStreamActionPlugin(ctx context.Context, query actionplugin.ActionQuery) *Result {
	stream, err := c.grpcClient.(actionplugin.ActionPluginClient).Stream(ctx, &query)
	if err != nil {
		return &Result{Status: sdk.StatusFail, Details: fmt.Sprintf("error while running plugin %s: %v", c.pluginName, err)}
	}

	res := &Result{
		Status: sdk.StatusBuilding,
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			if res.Status == sdk.StatusBuilding {
				res.Details = "unexpected end of plugin: connection closed"
				res.Status = sdk.StatusFail
			}
			break
		} else if err == nil {
			if resp.GetLogs() != "" {
				c.w.SendLog(ctx, workerruntime.LevelInfo, resp.Logs)
			}
			if resp.GetStatus() != "" {
				res.Status = resp.GetStatus()
			}
			if resp.GetDetails() != "" {
				res.Details = resp.Details
			}
		}
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = err.Error()
			break
		}
	}

	if !strings.EqualFold(res.Status, sdk.StatusSuccess) {
		res.Status = sdk.StatusFail
	}
	return res
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
		c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Details: %s", result.Details))
	} else {
		result.Status = sdk.StatusSuccess
	}
	c.w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Status: %s", result.Status))
	return result
}

func (c *client) Close(ctx context.Context) {
	switch c.pluginType {
	case TypeAction, TypeStream:
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
