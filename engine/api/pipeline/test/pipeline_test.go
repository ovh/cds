package test

import (
	"testing"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestInsertPipeline(t *testing.T) {
	db := test.SetupPG(t)
	pk := assets.RandomString(t, 8)

	p := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	if err := project.Insert(db, &p); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
	}

	tests := []struct {
		name    string
		p       *sdk.Pipeline
		wantErr bool
	}{
		{
			name:    "InsertPipeline should fail with sdk.ErrInvalidName",
			p:       &sdk.Pipeline{},
			wantErr: true,
		},
		{
			name: "InsertPipeline should fail with sdk.ErrInvalidType",
			p: &sdk.Pipeline{
				Name: "Name",
			},
			wantErr: true,
		},
		{
			name: "InsertPipeline should fail with sdk.ErrInvalidProject",
			p: &sdk.Pipeline{
				Name: "Name",
				Type: sdk.DeploymentPipeline,
			},
			wantErr: true,
		},
		{
			name: "InsertPipeline should not fail",
			p: &sdk.Pipeline{
				Name:      "Name",
				Type:      sdk.DeploymentPipeline,
				ProjectID: p.ID,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if err := pipeline.InsertPipeline(db, tt.p); (err != nil) != tt.wantErr {
			t.Errorf("%q. InsertPipeline() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestInsertPipelineWithParemeters(t *testing.T) {
	db := test.SetupPG(t)
	pk := assets.RandomString(t, 8)

	p := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	if err := project.Insert(db, &p); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
	}

	pip := &sdk.Pipeline{
		Name:      "Name",
		Type:      sdk.DeploymentPipeline,
		ProjectID: p.ID,
		Parameter: []sdk.Parameter{
			{
				Name:  "P1",
				Value: "V1",
				Type:  sdk.StringParameter,
			},
			{
				Name:  "P2",
				Value: "V2",
				Type:  sdk.StringParameter,
			},
		},
	}

	test.NoError(t, pipeline.InsertPipeline(db, pip))

	pip1, err := pipeline.LoadPipeline(db, p.Key, "Name", true)
	test.NoError(t, err)

	assert.Equal(t, len(pip.Parameter), len(pip1.Parameter))
}
