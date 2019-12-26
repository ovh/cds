package workflow

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func Test_purgeAudits(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()

	err := purgeAudits(context.TODO(), db)
	test.NoError(t, err)
}
