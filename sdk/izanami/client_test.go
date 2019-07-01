package izanami

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeature(t *testing.T) {
	c, err := New("http://localhost:8080/api", "yourclientid", "yoursecret")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Creating a feature")
	feat := FeatureModel{
		ID:       "test:feature",
		Enabled:  false,
		Strategy: NoStrategy,
	}
	assert.NoError(t, c.Feature().Create(feat))

	t.Log("Updating a feature")
	feat.Enabled = true
	assert.NoError(t, c.Feature().Update(feat))

	check, err := c.Feature().CheckWithoutContext(feat.ID)
	assert.NoError(t, err)
	assert.Equal(t, check.Active, true)

	t.Log("Getting a feature")
	f, errF := c.Feature().Get(feat.ID)
	assert.NoError(t, errF)

	assert.Equal(t, true, f.Enabled)

	assert.NoError(t, c.Feature().Delete(f.ID))

	features, errS := c.Feature().ListAll()
	assert.NoError(t, errS)

	assert.Equal(t, 0, len(features))
}
