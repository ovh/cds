package bitbucket

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestListStatuses(t *testing.T) {
	client := getAuthorizedClient(t)
	statuses, err := client.ListStatuses(context.Background(), "CDS/images", "9c4df9d61d85beb096715ace90acefb697f1e4d8")
	test.NoError(t, err)
	t.Logf("%+v", statuses)
}
