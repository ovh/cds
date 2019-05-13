package sdk_test

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowRunTag(t *testing.T) {
	a := sdk.Action{
		Enabled: true,
		Requirements: sdk.RequirementList{
			{Type: sdk.ModelRequirement, Value: "model1"},
			{Type: sdk.ServiceRequirement, Value: "service1"},
		},
		Actions: []sdk.Action{{
			Enabled: true,
			Requirements: sdk.RequirementList{
				{Type: sdk.HostnameRequirement, Value: "hostname1"},
				{Type: sdk.ServiceRequirement, Value: "service2"},
			},
		}, {
			Enabled: false,
			Requirements: sdk.RequirementList{
				{Type: sdk.ServiceRequirement, Value: "service3"},
			},
		}},
	}

	rs := a.FlattenRequirements()
	assert.Equal(t, 4, len(rs))
	assert.Equal(t, "model1", rs[0].Value)
	assert.Equal(t, "service1", rs[1].Value)
	assert.Equal(t, "hostname1", rs[2].Value)
	assert.Equal(t, "service2", rs[3].Value)
}
