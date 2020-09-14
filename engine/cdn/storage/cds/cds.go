package cds

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

type CDS struct {
	client cdsclient.Interface
	storage.AbstractUnit
	encryption.ConvergentEncryption
	config storage.CDSStorageConfiguration
}

var _ storage.StorageUnit = new(CDS)

func init() {
	storage.RegisterDriver("cds", new(CDS))
}

func (c *CDS) GetClient() cdsclient.Interface {
	return c.client
}

func (c *CDS) Init(cfg interface{}) error {
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
	return nil
}

func (c *CDS) ItemExists(i index.Item) (bool, error) {
	return true, nil
}

func (c *CDS) NewWriter(i storage.ItemUnit) (io.WriteCloser, error) {
	return nil, nil
}

func (c *CDS) NewReader(i storage.ItemUnit) (io.ReadCloser, error) {
	switch i.Item.Type {
	case sdk.CDNTypeItemStepLog:
		bs, err := c.client.WorkflowNodeRunJobStepLog(i.Item.APIRef.ProjectKey, i.Item.APIRef.WorkflowName, i.Item.APIRef.NodeRunID, i.Item.APIRef.NodeRunJobID, i.Item.APIRef.StepOrder)
		if err != nil {
			return nil, err
		}
		rc := ioutil.NopCloser(bytes.NewReader([]byte(bs.StepLogs.Val)))
		return rc, nil
	case sdk.CDNTypeItemServiceLog:
		log, err := c.ServiceLogs(i.Item.APIRef.ProjectKey, i.Item.APIRef.WorkflowName, i.Item.APIRef.NodeRunID, i.Item.APIRef.NodeRunJobID, i.Item.APIRef.RequirementServiceName)
		if err != nil {
			return nil, err
		}
		return ioutil.NopCloser(bytes.NewReader([]byte(log.Val))), nil
	}
	return nil, sdk.WithStack(fmt.Errorf("unable to find data for ref: %+v", i.Item.APIRef))
}

func (c *CDS) ServiceLogs(pKey string, wkfName string, nodeRunID int64, jobID int64, serviceName string) (*sdk.ServiceLog, error) {
	return c.client.WorkflowNodeRunJobServiceLog(pKey, wkfName, nodeRunID, jobID, serviceName)
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
