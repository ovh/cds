package gorpmapping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

type TestEncryptedData struct {
	gorpmapping.SignedEntity
	ID                   int64  `db:"id"`
	Data                 string `db:"data"`
	SensitiveData        string `db:"sensitive_data" gorpmapping:"encrypted,Data"`
	AnotherSensitiveData string `db:"another_sensitive_data" gorpmapping:"encrypted,ID,Data"`
}

func (e TestEncryptedData) Canonical() gorpmapping.CanonicalForms {
	return gorpmapping.CanonicalForms{
		"{{.ID}} {{.Data}}",
	}
}

func TestEncryption(t *testing.T) {
	gorpmapping.Register(gorpmapping.New(TestEncryptedData{}, "test_encrypted_data", true, "id"))

	db, _, end := test.SetupPG(t)
	defer end()

	var d = TestEncryptedData{
		Data:                 "data",
		SensitiveData:        "sensitive-data",
		AnotherSensitiveData: "another-sensitive-data",
	}

	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), db, &d)) //
	assert.Equal(t, sdk.PasswordPlaceholder, d.SensitiveData)
	assert.Equal(t, sdk.PasswordPlaceholder, d.AnotherSensitiveData)

	// UpdateAndSign should not save place holders
	require.NoError(t, gorpmapping.UpdateAndSign(context.TODO(), db, &d))

	var d1 TestEncryptedData
	query := gorpmapping.NewQuery("select * from test_encrypted_data where id = $1").Args(d.ID)
	_, err := gorpmapping.Get(context.TODO(), db, query, &d1, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)

	isValid, err := gorpmapping.CheckSignature(d1, d1.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d.ID, d1.ID)
	require.Equal(t, d.Data, d1.Data)
	require.Equal(t, "sensitive-data", d1.SensitiveData)
	require.Equal(t, "another-sensitive-data", d1.AnotherSensitiveData)

	// Test updates
	d.SensitiveData = "sensitive--data"
	d.AnotherSensitiveData = "another-sensitive-data"

	require.NoError(t, gorpmapping.UpdateAndSign(context.TODO(), db, &d))
	assert.Equal(t, sdk.PasswordPlaceholder, d.SensitiveData)
	assert.Equal(t, sdk.PasswordPlaceholder, d.AnotherSensitiveData)

	query = gorpmapping.NewQuery("select * from test_encrypted_data where id = $1").Args(d.ID)

	var d2 TestEncryptedData
	_, err = gorpmapping.Get(context.TODO(), db, query, &d2)
	require.NoError(t, err)

	isValid, err = gorpmapping.CheckSignature(d2, d2.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d.ID, d2.ID)
	require.Equal(t, d.Data, d2.Data)
	require.NotEqual(t, "sensitive--data", d2.SensitiveData)
	require.Equal(t, sdk.PasswordPlaceholder, d2.SensitiveData)
	require.NotEqual(t, "another-sensitive-data", d2.AnotherSensitiveData)
	require.Equal(t, sdk.PasswordPlaceholder, d2.AnotherSensitiveData)

	_, err = gorpmapping.Get(context.TODO(), db, query, &d2, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)

	isValid, err = gorpmapping.CheckSignature(d2, d2.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d.ID, d2.ID)
	require.Equal(t, d.Data, d2.Data)
	require.Equal(t, "sensitive--data", d2.SensitiveData)
	require.Equal(t, "another-sensitive-data", d2.AnotherSensitiveData)
}

func TestEncryption_Multiple(t *testing.T) {
	gorpmapping.Register(gorpmapping.New(TestEncryptedData{}, "test_encrypted_data", true, "id"))

	db, _, end := test.SetupPG(t)
	defer end()

	var d = TestEncryptedData{
		Data:                 "data",
		SensitiveData:        "sensitive-data",
		AnotherSensitiveData: "another-sensitive-data",
	}
	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), db, &d))

	var dd = TestEncryptedData{
		Data:                 "data-2",
		SensitiveData:        "sensitive-data-2",
		AnotherSensitiveData: "another-sensitive-data-2",
	}
	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), db, &dd))

	// Test that GetAll replaces encrypted values with placeholders
	query := gorpmapping.NewQuery("select * from test_encrypted_data where id IN ($1, $2) order by id").Args(d.ID, dd.ID)
	var dslice []TestEncryptedData
	err := gorpmapping.GetAll(context.TODO(), db, query, &dslice)
	require.NoError(t, err)
	require.Len(t, dslice, 2)

	d2 := dslice[0]
	require.Equal(t, sdk.PasswordPlaceholder, d2.SensitiveData)
	require.Equal(t, sdk.PasswordPlaceholder, d2.AnotherSensitiveData)

	d2 = dslice[1]
	require.Equal(t, sdk.PasswordPlaceholder, d2.SensitiveData)
	require.Equal(t, sdk.PasswordPlaceholder, d2.AnotherSensitiveData)

	// Test that GetAll replaces encrypted values with clearValue if WithDecryption options is used
	query = gorpmapping.NewQuery("select * from test_encrypted_data where id IN ($1, $2) order by id").Args(d.ID, dd.ID)
	dslice = []TestEncryptedData{}
	err = gorpmapping.GetAll(context.TODO(), db, query, &dslice, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.Len(t, dslice, 2)

	d2 = dslice[0]
	isValid, err := gorpmapping.CheckSignature(d2, d2.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d.ID, d2.ID)
	require.Equal(t, d.Data, d2.Data)
	require.Equal(t, "sensitive-data", d2.SensitiveData)
	require.Equal(t, "another-sensitive-data", d2.AnotherSensitiveData)

	d2 = dslice[1]
	isValid, err = gorpmapping.CheckSignature(d2, d2.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, dd.ID, d2.ID)
	require.Equal(t, dd.Data, d2.Data)
	require.Equal(t, "sensitive-data-2", d2.SensitiveData)
	require.Equal(t, "another-sensitive-data-2", d2.AnotherSensitiveData)
}
