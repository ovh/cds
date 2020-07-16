package sdk_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
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

func TestFlattenRequirementRecursively(t *testing.T) {
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
				{Type: sdk.BinaryRequirement, Value: "git1"},
			},
			Actions: []sdk.Action{
				{
					Enabled: true,
					Requirements: sdk.RequirementList{
						{Type: sdk.BinaryRequirement, Value: "git2"},
					},
					Actions: []sdk.Action{
						{
							Enabled: true,
							Requirements: sdk.RequirementList{
								{Type: sdk.HostnameRequirement, Value: "hostname1"},
								{Type: sdk.ServiceRequirement, Value: "service2"},
								{Type: sdk.BinaryRequirement, Value: "git4"},
							},
						},
					},
				},
			},
		}, {
			Enabled: false,
			Requirements: sdk.RequirementList{
				{Type: sdk.ServiceRequirement, Value: "service3"},
				{Type: sdk.BinaryRequirement, Value: "git3"},
			},
		}},
	}

	rs := a.FlattenRequirementsRecursively()
	assert.Equal(t, 7, len(rs))
	assert.Equal(t, "model1", rs[0].Value)
	assert.Equal(t, "service1", rs[1].Value)
	assert.Equal(t, "hostname1", rs[2].Value)
	assert.Equal(t, "service2", rs[3].Value)
	assert.Equal(t, "git1", rs[4].Value)
	assert.Equal(t, "git2", rs[5].Value)
	assert.Equal(t, "git4", rs[6].Value)
}
