package bitbucketserver

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestListStatuses(t *testing.T) {
	client := getAuthorizedClient(t)
	statuses, err := client.ListStatuses(context.Background(), "CDS/tests", "0b6d50472e9b2c03d72a422ea11bf3faa570d0bd")
	test.NoError(t, err)
	t.Logf("%+v", statuses)
}
