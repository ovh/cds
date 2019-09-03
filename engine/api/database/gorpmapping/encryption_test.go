package gorpmapping_test

import (
	"context"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/test"
)

type TestEncryptedData struct {
	gorpmapping.SignedEntity
	ID                   int64  `db:"id"`
	Data                 string `db:"data"`
	SensitiveData        string `db:"sensitive_data" gorpmapping:"encrypted"`
	AnotherSensitiveData string `db:"another_sensitive_data" gorpmapping:"encrypted"`
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

	require.NoError(t, gorpmapping.InsertAndSign(db, &d)) //
	assert.Equal(t, "", d.SensitiveData)
	assert.Equal(t, "", d.AnotherSensitiveData)

	d.SensitiveData = "sensitive--data"
	d.AnotherSensitiveData = "another-sensitive-data"

	require.NoError(t, gorpmapping.UpdateAndSign(db, &d))
	assert.Equal(t, "", d.SensitiveData)
	assert.Equal(t, "", d.AnotherSensitiveData)

	query := gorpmapping.NewQuery("select * from test_encrypted_data where id = $1").Args(d.ID)

	var d2 TestEncryptedData
	_, err := gorpmapping.Get(context.TODO(), db, query, &d2)
	require.NoError(t, err)

	isValid, err := gorpmapping.CheckSignature(d2, d2.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d.ID, d2.ID)
	require.Equal(t, d.Data, d2.Data)
	require.NotEqual(t, "sensitive--data", d2.SensitiveData)
	require.Equal(t, "", d2.SensitiveData)
	require.NotEqual(t, "another-sensitive-data", d2.AnotherSensitiveData)
	require.Equal(t, "", d2.AnotherSensitiveData)

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
	require.NoError(t, gorpmapping.InsertAndSign(db, &d))

	var dd = TestEncryptedData{
		Data:                 "data-2",
		SensitiveData:        "sensitive-data-2",
		AnotherSensitiveData: "another-sensitive-data-2",
	}
	require.NoError(t, gorpmapping.InsertAndSign(db, &dd))

	query := gorpmapping.NewQuery("select * from test_encrypted_data where id IN ($1, $2) order by id").Args(d.ID, dd.ID)
	var dslice []TestEncryptedData
	err := gorpmapping.GetAll(context.TODO(), db, query, &dslice, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.Len(t, dslice, 2)

	d2 := dslice[0]
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
