package mesos

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func Test_marathonConfig(t *testing.T) {
	m := &HatcheryMesos{
		marathonHost:         "marathonHost",
		marathonID:           "marathonID",
		marathonVHOST:        "marathonVHOST",
		marathonUser:         "marathonUser",
		marathonPassword:     "marathonPassword",
		marathonLabelsString: "marathonLabelsString",
		marathonLabels: map[string]string{
			"blabla": "blabla",
		},
		workerTTL: 100,
	}

	r, err := m.marathonConfig(&sdk.Model{ID: 1, Name: "model", Image: "my-image:latest"}, 1, 64)
	test.NoError(t, err)
	assert.NotNil(t, r)

	b, err := ioutil.ReadAll(r)
	test.NoError(t, err)

	t.Logf("%s", b)

	config := map[string]interface{}{}
	expected := map[string]interface{}{}
	json.Unmarshal(b, &config)
	json.Unmarshal([]byte(`	{
		    "container": {
		        "docker": {
		            "forcePullImage": true,
		            "image": "my-image:latest",
		            "network": "BRIDGE",
							  "portMapping": []
						},
		        "type": "DOCKER"
		    },
				"cmd": "rm -f worker && curl ${CDS_API}/download/worker/$(uname -m) -o worker &&  chmod +x worker && exec ./worker",
				"cpus": 0.5,
		    "env": {
		        "CDS_API": "",
		        "CDS_KEY": "",
		        "CDS_NAME": "model-silly-einstein",
		        "CDS_MODEL": "1",
		        "CDS_HATCHERY": "1",
		        "CDS_SINGLE_USE": "1",
				"CDS_TTL" : "10"
		    },
		    "id": "marathonID/model-silly-einstein",
		    "instances": 1,
			"ports": [],
			"mem": 70,
			"labels": {"blabla":"blabla","hatchery":"1"}
		}
`), &expected)

	assert.True(t, reflect.DeepEqual(expected["container"], config["container"]))
	assert.True(t, reflect.DeepEqual(expected["labels"], config["labels"]))

}
