package main

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestXxx(t *testing.T) {
	payload := `{
		"type": "V2WorkflowRunResultStaticFilesDetail",
		"data": {
		  "name": "hello",
		  "artifactory_url": "fsamin-default-static/test-static-files/",
		  "public_url": "https://rtstatic.ovhcloud.tools/fsamin/default/test-static-files"
		}
	  }`

	var detail sdk.V2WorkflowRunResultDetail

	err := sdk.JSONUnmarshal([]byte(payload), &detail)
	require.NoError(t, err)

}
