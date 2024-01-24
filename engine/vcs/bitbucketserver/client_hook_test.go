package bitbucketserver

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestCreateHook(t *testing.T) {
	client := getAuthorizedClient(t)

	h := sdk.VCSHook{
		Method: "POST",
		URL:    "http://localhost:8090",
	}

	err := client.CreateHook(context.Background(), "CDS/tests", &h)
	test.NoError(t, err)
}

func TestDeleteHook(t *testing.T) {
	client := getAuthorizedClient(t)

	h := sdk.VCSHook{
		Method: "POST",
		URL:    "http://localhost:8080",
	}

	err := client.CreateHook(context.Background(), "CDS/tests", &h)
	test.NoError(t, err)

	err = client.DeleteHook(context.Background(), "CDS/tests", h)
	test.NoError(t, err)
}
