package cds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
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

func (c *CDS) Init(cfg interface{}) error {
	config, is := cfg.(*storage.CDSStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	c.config = *config
	c.ConvergentEncryption = encryption.New(config.Encryption)
	cdsClient, _, err := cdsclient.NewServiceClient(cdsclient.ServiceConfig{
		Host:                  config.Host,
		InsecureSkipVerifyTLS: config.InsecureSkipVerifyTLS,
		Token:                 config.Token,
	})
	if err != nil {
		return sdk.WithStack(err)
	}
	c.client = cdsClient
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
	btsData := make([]byte, 0)
	for _, l := range bs.Logs {
		btsData = append(btsData, []byte(l.Val)...)
	}
	rc := ioutil.NopCloser(bytes.NewReader(btsData))
	return rc, nil
}

func (c *CDS) Sync(ctx context.Context) {
	log.Info(ctx, "cdn: Start CDS sync")
	tick := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "cdn:sync: %v", ctx.Err())
			}
			return
		case <-tick.C:
			projects, err := c.client.ProjectList(false, true)
			if err != nil {
				log.Error(ctx, "cdn:cds:sync: unable to list projects")
				continue
			}
			for _, p := range projects {
				// Check feature enable
				resp, err := c.client.FeatureEnabled("cdn-job-logs", map[string]string{"project_key": p.Key})
				if err != nil {
					log.Error(ctx, "unable to check feature %s for %s", "cdn-job-logs", "project_key")
					continue
				}
				if !resp.Enabled {
					continue
				}

				wkflws, err := c.client.WorkflowList(p.Key)
				if err != nil {
					log.Error(ctx, "unable to list workflow: %v", err)
					continue
				}
				for _, w := range wkflws {
					runs, err := c.client.WorkflowRunList(p.Key, w.Name, 0, 1000)
					if err != nil {
						log.Error(ctx, "unable to list workflow run: %v", err)
						continue
					}

					for _, r := range runs {
						for _, nrs := range r.WorkflowNodeRuns {
							if len(nrs) > 0 {
								nr := nrs[0]
								for _, s := range nr.Stages {
									for _, j := range s.RunJobs {

									}
								}
							}
						}
					}
				}
			}
		}
	}
}
