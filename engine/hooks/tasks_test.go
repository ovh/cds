package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

func Test_doWebHookExecution(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: nil,
			RequestURL:  "uid=42413e87905b813a375c7043ce9d4047b7e265ae3730b60180cad02ae81cc62385e5b05b9e7c758b15bb3872498a5e88963f3deac308f636baf345ed9cf1b259&project=IRTM&name=rtm-packaging&branch=master&hash=123456789&message=monmessage&author=sguiheux",
		},
	}
	hs, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	assert.Equal(t, 1, len(hs))
	assert.Equal(t, "master", hs[0].Payload["branch"])
	assert.Equal(t, "sguiheux", hs[0].Payload["author"])
	assert.Equal(t, "monmessage", hs[0].Payload["message"])
	assert.Equal(t, "123456789", hs[0].Payload["hash"])
}
