package cds

import (
	"bytes"
	"fmt"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"io"
	"io/ioutil"
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
	bs, err := c.client.WorkflowNodeRunJobStep(i.Item.ApiRef.ProjectKey, i.Item.ApiRef.WorkflowName, 0, i.Item.ApiRef.NodeRunID, i.Item.ApiRef.NodeRunJobID, int(i.Item.ApiRef.StepOrder))
	if err != nil {
		return nil, err
	}
	rc := ioutil.NopCloser(bytes.NewReader([]byte(bs.StepLogs.Val)))
	return rc, nil
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
