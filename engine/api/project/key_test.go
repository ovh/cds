package project_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestEncryptWithBuiltinKey(t *testing.T) {
	db, cache := test.SetupPG(t)
	key := sdk.RandomString(10)
	u, _ := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	content := "This is my content"
	encryptedContent, err := project.EncryptWithBuiltinKey(db, proj.ID, "test", content)
	test.NoError(t, err)
	t.Logf("%s => %s", content, encryptedContent)

	decryptedContent, err := project.DecryptWithBuiltinKey(db, proj.ID, encryptedContent)
	test.NoError(t, err)
	t.Logf("%s => %s", encryptedContent, decryptedContent)

	assert.Equal(t, content, decryptedContent)

	encryptedContent2, err := project.EncryptWithBuiltinKey(db, proj.ID, "test", content)
	test.NoError(t, err)
	t.Logf("%s => %s", content, encryptedContent2)
	assert.Equal(t, encryptedContent, encryptedContent2)

}
