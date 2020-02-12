package workflow

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func Test_PurgeAudits(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()

	err := PurgeAudits(context.TODO(), db)
	test.NoError(t, err)
}
