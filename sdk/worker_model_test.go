package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
