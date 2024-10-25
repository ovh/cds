package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/cds/contrib/integrations/arsenal"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/interpolate"
)

/* Inside contrib/grpcplugins/action
$ make build plugin-arsenal-delete-alternative
$ make publish plugin-arsenal-delete-alternative
*/

type arsenalDeploymentPlugin struct {
	actionplugin.Common
}

func (e *arsenalDeploymentPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "OVH Arsenal Delete Alternative Plugin",
		Author:      "Alexandre Jin",
		Description: "OVH Arsenal plugin to delete an alternative from a deployment",
		Version:     sdk.VERSION,
	}, nil
}

func (p *arsenalDeploymentPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	return sdk.ErrNotImplemented
}

func (e *arsenalDeploymentPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	// Read and check inputs
	var (
		arsenalHost     = getStringOption(q, "cds.integration.deployment.host")
		deploymentToken = getStringOption(q, "cds.integration.deployment.deployment.token", "cds.integration.deployment.token")
		alternative     = getStringOption(q, "cds.integration.deployment.alternative.config")
		alternativeName = getStringOption(q, "alternative_name")
	)
	if arsenalHost == "" {
		return fail("missing arsenal host")
	}
	if deploymentToken == "" {
		return fail("missing arsenal deployment token")
	}
	if alternativeName == "" {
		if alternative == "" {
			return fail("missing arsenal alternative config")
		}
		// Resolve alternative.
		var err error
		alternativeName, err = resolveAlternativeName(alternative, q.GetOptions())
		if err != nil {
			return fail("failed to resolve alternative config: %v\n", err)
		}
	}

	// Delete alternative.
	arsenalClient := arsenal.NewClient(arsenalHost, deploymentToken)
	fmt.Printf("Deleting alternative %q\n", alternativeName)
	if err := arsenalClient.DeleteAlternative(alternativeName); err != nil {
		return fail("failed to delete alternative: %v", err)
	}

	fmt.Printf("Alternative %q deleted\n", alternativeName)
	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func resolveAlternativeName(alternative string, options map[string]string) (string, error) {
	var altConfig *arsenal.Alternative
	altTmpl, err := template.New("alternative").Delims("[[", "]]").Funcs(interpolate.InterpolateHelperFuncs).Parse(alternative)
	if err != nil {
		return "", err
	}
	var altBuf bytes.Buffer
	if err = altTmpl.Execute(&altBuf, options); err != nil {
		return "", fmt.Errorf("failed to interpolate alternative config: %w", err)
	}
	if altBuf.Len() == 0 {
		return "", fmt.Errorf("no alternative resolved from arsenal alternative config")
	}
	if err = json.Unmarshal(altBuf.Bytes(), &altConfig); err != nil {
		fmt.Println("Resolved alternative:", altBuf.String())
		return "", fmt.Errorf("failed to unmarshal alternative config: %w", err)
	}
	return altConfig.Name, nil
}

func getStringOption(q *actionplugin.ActionQuery, keys ...string) string {
	for _, k := range keys {
		if v, exists := q.GetOptions()[k]; exists {
			return v
		}
	}
	return ""
}

func fail(format string, args ...interface{}) (*actionplugin.ActionResult, error) {
	return failErr(fmt.Errorf(format, args...))
}

func failErr(err error) (*actionplugin.ActionResult, error) {
	fmt.Println("Error:", err)
	return &actionplugin.ActionResult{
		Details: err.Error(),
		Status:  sdk.StatusFail,
	}, nil
}

func main() {
	e := arsenalDeploymentPlugin{}
	if err := actionplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
}
