package hooks

import (
	"encoding/json"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildHookMessage(t *testing.T) {
	msg := `
    {"type":"patchset-created","change":{"project":"CDS/gerrit","branch":"master","topic":"patchset","id":"Ied67a65d33f13d77c2d98540823a9eb0c887f35a","subject":"fix","owner":{"username":"steven.guiheux"},"url":"https://mygerrit/c/CDS/gerrit/+/11","commitMessage":"fix\n\nChange-Id: Ied67a65d33f13d77c2d98540823a9eb0c887f35a\n","createdOn":1549015585,"status":"NEW"},"eventCreatedOn":1549015585,"patchSet":{"number":1,"revision":"bb488dea35f140fcac3ffd04d2d01f0f29c75100","parents":["70849c92d899f30f092ad74cd59a651e03a07902"],"ref":"refs/changes/11/11/1","uploader":{"username":"steven.guiheux"},"author":{"name":"steven guiheux","email":"steven.guiheux@corp.ovh.com"},"createdOn":1549015585,"kind":"REWORK","sizeInsertions":1}}
  `

	var gerritEvent GerritEvent
	assert.NoError(t, json.Unmarshal([]byte(msg), &gerritEvent))

	s := Service{}
	te := &sdk.TaskExecution{
		GerritEvent: &sdk.GerritEventExecution{
			Message: []byte(msg),
		},
		UUID: "123",
	}
	hookEvent, err := s.doGerritExecution(te)
	assert.NoError(t, err)

	assert.Equal(t, hookEvent.Payload["git.author"], "steven.guiheux")
	assert.Equal(t, hookEvent.Payload["git.author.email"], "steven.guiheux@corp.ovh.com")
	assert.Equal(t, hookEvent.Payload["git.hash"], "bb488dea35f140fcac3ffd04d2d01f0f29c75100")
	assert.Equal(t, hookEvent.Payload["git.hash.before"], "70849c92d899f30f092ad74cd59a651e03a07902")
	assert.Equal(t, hookEvent.Payload["git.message"], "fix\n\nChange-Id: Ied67a65d33f13d77c2d98540823a9eb0c887f35a\n")
	assert.Equal(t, hookEvent.Payload["gerrit.change.id"], "Ied67a65d33f13d77c2d98540823a9eb0c887f35a")
	assert.Equal(t, hookEvent.Payload["gerrit.change.ref"], "refs/changes/11/11/1")
	assert.Equal(t, hookEvent.Payload["gerrit.change.status"], "NEW")
	assert.Equal(t, hookEvent.Payload["gerrit.type"], "patchset-created")
	assert.Equal(t, hookEvent.Payload["gerrit.change.branch"], "master")
	assert.Equal(t, hookEvent.Payload["git.branch"], "")
}
