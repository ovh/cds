package cds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

type CDS struct {
	client cdsclient.Interface
	storage.AbstractUnit
	config storage.CDSStorageConfiguration
}

func init() {
	storage.RegisterDriver("cds", new(CDS))
}

func (c *CDS) GetClient() cdsclient.Interface {
	return c.client
}

func (c *CDS) Init(ctx context.Context, cfg interface{}) error {
	config, is := cfg.(*storage.CDSStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	c.config = *config

	c.client = cdsclient.New(cdsclient.Config{
		Host:                              config.Host,
		InsecureSkipVerifyTLS:             config.InsecureSkipVerifyTLS,
		BuitinConsumerAuthenticationToken: config.Token,
	})

	return nil
}

func (c *CDS) ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	return true, nil
}

func (c *CDS) NewWriter(_ context.Context, _ sdk.CDNItemUnit) (io.WriteCloser, error) {
	return nil, nil
}

func (c *CDS) NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	switch i.Item.Type {
	case sdk.CDNTypeItemStepLog:
		bs, err := c.client.WorkflowNodeRunJobStepLog(ctx, i.Item.APIRef.ProjectKey, i.Item.APIRef.WorkflowName, i.Item.APIRef.NodeRunID, i.Item.APIRef.NodeRunJobID, i.Item.APIRef.StepOrder)
		if err != nil {
			return nil, err
		}
		rc := ioutil.NopCloser(bytes.NewReader([]byte(bs.StepLogs.Val)))
		return rc, nil
	case sdk.CDNTypeItemServiceLog:
		log, err := c.ServiceLogs(ctx, i.Item.APIRef.ProjectKey, i.Item.APIRef.WorkflowName, i.Item.APIRef.NodeRunID, i.Item.APIRef.NodeRunJobID, i.Item.APIRef.RequirementServiceName)
		if err != nil {
			return nil, err
		}
		rc := ioutil.NopCloser(bytes.NewReader([]byte(log.Val)))
		return rc, nil
	}
	return nil, sdk.WithStack(fmt.Errorf("unable to find data for ref: %+v", i.Item.APIRef))
}

func (c *CDS) ServiceLogs(ctx context.Context, pKey string, wkfName string, nodeRunID int64, jobID int64, serviceName string) (*sdk.ServiceLog, error) {
	return c.client.WorkflowNodeRunJobServiceLog(ctx, pKey, wkfName, nodeRunID, jobID, serviceName)
}

func (c *CDS) ListProjects() ([]sdk.Project, error) {
	return c.client.ProjectList(false, false)
}

func (c *CDS) ListNodeRunIdentifiers(pKey string) ([]sdk.WorkflowNodeRunIdentifiers, error) {
	return c.client.WorkflowRunsAndNodesIDs(pKey)
}

func (c *CDS) FeatureEnabled(name string, params map[string]string) (sdk.FeatureEnabledResponse, error) {
	return c.client.FeatureEnabled(name, params)
}

func (c *CDS) GetWorkflowNodeRun(pKey string, nodeRunIdentifier sdk.WorkflowNodeRunIdentifiers) (*sdk.WorkflowNodeRun, error) {
	return c.client.WorkflowNodeRun(pKey, nodeRunIdentifier.WorkflowName, nodeRunIdentifier.RunNumber, nodeRunIdentifier.NodeRunID)
}

func (c *CDS) Status(_ context.Context) []sdk.MonitoringStatusLine {
	if _, err := c.client.Version(); err != nil {
		return []sdk.MonitoringStatusLine{{Component: "backend/" + c.Name(), Value: "cds KO" + err.Error(), Status: sdk.MonitoringStatusAlert}}
	}
	return []sdk.MonitoringStatusLine{{
		Component: "backend/" + c.Name(),
		Value:     "connect OK",
		Status:    sdk.MonitoringStatusOK,
	}}
}

func (c *CDS) Remove(ctx context.Context, i sdk.CDNItemUnit) error {
	return nil
}

func (c *CDS) Read(_ sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return sdk.WithStack(err)
}

func (c *CDS) Write(_ sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	return nil
}
