package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelMatchesRequirementValue guards the model requirement matching used both by the hatchery
// to decide if it can spawn a worker for a job, and by the API to decide if a queued job is stuck on
// a disabled model. Both must agree, otherwise the API would fail jobs a hatchery could still run.
func TestModelMatchesRequirementValue(t *testing.T) {
	groupModel := Model{Name: "myModel", Group: &Group{Name: "myGroup"}}
	sharedModel := Model{Name: "myModel", Group: &Group{Name: SharedInfraGroupName}}

	tests := []struct {
		name     string
		model    Model
		value    string
		expected bool
	}{
		{"group model by full path", groupModel, "myGroup/myModel", true},
		{"group model by bare name, kept for backward compatibility with old runs", groupModel, "myModel", true},
		{"group model with extra options", groupModel, "myGroup/myModel --port=8888:9999", true},
		{"group model of another group", groupModel, "otherGroup/myModel", false},
		{"group model of another name", groupModel, "myGroup/otherModel", false},
		{"shared.infra model by bare name", sharedModel, "myModel", true},
		{"shared.infra model by full path", sharedModel, SharedInfraGroupName + "/myModel", true},
		{"model without group loaded, bare name still matches", Model{Name: "myModel"}, "myModel", true},
		{"model without group loaded, path cannot match", Model{Name: "myModel"}, "myGroup/myModel", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.model.MatchesRequirementValue(tt.value))
		})
	}
}

// TestModelIsValidEOL checks that an end of life date is only accepted on a deprecated model: the
// date exists to schedule the automatic disabling of a deprecated model, it has no meaning alone.
func TestModelIsValidEOL(t *testing.T) {
	eol := time.Now()

	t.Run("eol on a deprecated model is valid", func(t *testing.T) {
		m := Model{Name: "myModel", GroupID: 1, IsDeprecated: true, EOL: &eol}
		require.NoError(t, m.IsValid())
	})

	t.Run("eol without deprecated is rejected", func(t *testing.T) {
		m := Model{Name: "myModel", GroupID: 1, EOL: &eol}
		err := m.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "end of life date can only be set on a deprecated worker model")
	})

	t.Run("deprecated without eol is valid", func(t *testing.T) {
		m := Model{Name: "myModel", GroupID: 1, IsDeprecated: true}
		require.NoError(t, m.IsValid())
	})
}
