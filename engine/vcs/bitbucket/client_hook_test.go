package bitbucket

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
		URL:    "http://localhost:8080",
	}

	err := client.CreateHook(context.Background(), "CDS/cds-event-function", &h)
	test.NoError(t, err)
}

func TestDeleteHook(t *testing.T) {
	client := getAuthorizedClient(t)

	h := sdk.VCSHook{
		Method: "POST",
		URL:    "http://localhost:8080",
	}

	err := client.CreateHook(context.Background(), "CDS/cds-event-function", &h)
	test.NoError(t, err)

	err = client.DeleteHook(context.Background(), "CDS/cds-event-function", h)
	test.NoError(t, err)
}

func TestUpdateHook(t *testing.T) {
	client := getAuthorizedClient(t)

	h := sdk.VCSHook{
		Method: "POST",
		URL:    "http://localhost:8080",
	}

	err := client.CreateHook(context.Background(), "CDS/cds-event-function", &h)
	test.NoError(t, err)

	h = sdk.VCSHook{
		Method: "GET",
		URL:    "http://localhost:8080",
	}

	err = client.UpdateHook(context.Background(), "CDS/cds-event-function", h.URL, h)
	test.NoError(t, err)

	err = client.DeleteHook(context.Background(), "CDS/cds-event-function", h)
	test.NoError(t, err)
}
