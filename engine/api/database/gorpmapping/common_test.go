package gorpmapping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
)

func TestIDsToQueryString(t *testing.T) {
	assert.Equal(t, "1,2,3", gorpmapping.IDsToQueryString([]int64{1, 2, 3}))
}
