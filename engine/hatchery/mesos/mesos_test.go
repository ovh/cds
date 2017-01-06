package mesos

import (
	"testing"

	"io/ioutil"

	"encoding/json"

	"reflect"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
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

	r, err := m.marathonConfig(&sdk.Model{ID: 1, Name: "model"}, 1, 64)
	test.NoError(t, err)
	assert.NotNil(t, r)

	b, err := ioutil.ReadAll(r)

	t.Logf("%s", b)

	config := map[string]interface{}{}
	expected := map[string]interface{}{}
	json.Unmarshal(b, &config)
	json.Unmarshal([]byte(`	{
		    "container": {
		        "docker": {
		            "forcePullImage": false,
		            "image": "",
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
