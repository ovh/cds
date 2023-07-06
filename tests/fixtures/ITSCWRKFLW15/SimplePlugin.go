package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
)

type simpleIntegrationPlugin struct {
	integrationplugin.Common
}

func (actPlugin *simpleIntegrationPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "Hello deployment plugin",
		Author:      "Steven GUIHEUX <foo.bar@foobar.com>",
		Description: `This plugin do nothing.`,
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *simpleIntegrationPlugin) Run(ctx context.Context, q *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	fmt.Println("Hello, I'm the simple plugin")
	var deploymentToken = q.GetOptions()["cds.integration.deployment.deployment.token"]
	var maxRetryStr = q.GetOptions()["cds.integration.deployment.retry.max"]
	var delayRetryStr = q.GetOptions()["cds.integration.deployment.retry.delay"]
	maxRetry, err := strconv.Atoi(maxRetryStr)
	if err != nil {
		fmt.Printf("Error parsing cds.integration.deployment.retry.max: %v. Default value (10) will be used\n", err)
		maxRetry = 10
	}
	delayRetry, err := strconv.Atoi(delayRetryStr)
	if err != nil {
		fmt.Printf("Error parsing cds.integration.deployment.retry.max: %v. Default value (5) will be used\n", err)
		delayRetry = 5
	}
	fmt.Printf("Deployment.token %s\n", reverse(deploymentToken))
	fmt.Printf("Retry.max %d\n", maxRetry)
	fmt.Printf("Retry.delay %d\n", delayRetry)
	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := simpleIntegrationPlugin{}
	if err := integrationplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
