package cds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/telemetry"
	"go.opencensus.io/stats"
)

type CDS struct {
	client cdsclient.Interface
	storage.AbstractUnit
	encryption.ConvergentEncryption
	config storage.CDSStorageConfiguration
}

var (
	_                         storage.StorageUnit = new(CDS)
	metricsReaders                                = stats.Int64("cdn/storage/cds/readers", "nb readers", stats.UnitDimensionless)
	metricsReadersStepLogs                        = stats.Int64("cdn/storage/cds/readers/steps", "nb readers for steps logs", stats.UnitDimensionless)
	metricsReadersServiceLogs                     = stats.Int64("cdn/storage/cds/readers/services", "nb readers for service slogs", stats.UnitDimensionless)
	metricsWriters                                = stats.Int64("cdn/storage/cds/writers", "nb writers", stats.UnitDimensionless)
)

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
	c.ConvergentEncryption = encryption.New(config.Encryption)

	c.client = cdsclient.New(cdsclient.Config{
		Host:                              config.Host,
		InsecureSkipVerifyTLS:             config.InsecureSkipVerifyTLS,
		BuitinConsumerAuthenticationToken: config.Token,
	})

	if err := telemetry.InitMetricsInt64(ctx, metricsReaders, metricsWriters, metricsReadersStepLogs, metricsReadersServiceLogs); err != nil {
		return err
	}

	return nil
}

func (c *CDS) ItemExists(i index.Item) (bool, error) {
	return true, nil
}

func (c *CDS) NewWriter(ctx context.Context, i storage.ItemUnit) (io.WriteCloser, error) {
	return nil, nil
}

func (c *CDS) NewReader(ctx context.Context, i storage.ItemUnit) (io.ReadCloser, error) {
	telemetry.Record(ctx, metricsReaders, 1)
	switch i.Item.Type {
	case index.TypeItemStepLog:
		bs, err := c.client.WorkflowNodeRunJobStep(i.Item.ApiRef.ProjectKey, i.Item.ApiRef.WorkflowName, 0, i.Item.ApiRef.NodeRunID, i.Item.ApiRef.NodeRunJobID, int(i.Item.ApiRef.StepOrder))
		if err != nil {
			return nil, err
		}
		rc := ioutil.NopCloser(bytes.NewReader([]byte(bs.StepLogs.Val)))
		telemetry.Record(ctx, metricsReadersStepLogs, 1)
		return rc, nil
	case index.TypeItemServiceLog:
		logs, err := c.ServiceLogs(i.Item.ApiRef.ProjectKey, i.Item.ApiRef.WorkflowName, i.Item.ApiRef.NodeRunID, i.Item.ApiRef.NodeRunJobID)
		if err != nil {
			return nil, err
		}
		for _, l := range logs {
			if l.ServiceRequirementName != i.Item.ApiRef.RequirementServiceName {
				continue
			}
			rc := ioutil.NopCloser(bytes.NewReader([]byte(l.Val)))
			telemetry.Record(ctx, metricsReadersServiceLogs, 1)
			return rc, nil
		}
	default:
		return nil, sdk.WithStack(fmt.Errorf("unable to read type %s", i.Item.Type))
	}
	return nil, sdk.WithStack(fmt.Errorf("unable to find data for ref: %+v", i.Item.ApiRef))
}

func (c *CDS) ServiceLogs(pKey string, wkfName string, nodeRunID int64, jobID int64) ([]sdk.ServiceLog, error) {
	return c.client.WorkflowNodeRunJobServiceLog(pKey, wkfName, 0, nodeRunID, jobID)
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

func (c *CDS) Status(ctx context.Context) []sdk.MonitoringStatusLine {
	if _, err := c.client.Version(); err != nil {
		return []sdk.MonitoringStatusLine{{Component: "backend/cds", Value: "cds KO" + err.Error(), Status: sdk.MonitoringStatusAlert}}
	}
	return []sdk.MonitoringStatusLine{{
		Component: "backend/cds",
		Value:     "connect OK",
		Status:    sdk.MonitoringStatusOK,
	}}
}
