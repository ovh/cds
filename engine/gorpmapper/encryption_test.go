package gorpmapper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestEncryption(t *testing.T) {
	m := gorpmapper.New()
	m.Register(m.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeAPI)

	var d = gorpmapper.TestEncryptedData{
		Data:                 "data",
		SensitiveData:        "sensitive-data",
		AnotherSensitiveData: "another-sensitive-data",
	}

	require.NoError(t, m.InsertAndSign(context.TODO(), db, &d))
	assert.Equal(t, sdk.PasswordPlaceholder, d.SensitiveData)
	assert.Equal(t, sdk.PasswordPlaceholder, d.AnotherSensitiveData)

	// UpdateAndSign should not save place holders
	require.NoError(t, m.UpdateAndSign(context.TODO(), db, &d))

	var d1 gorpmapper.TestEncryptedData
	query := gorpmapper.NewQuery("SELECT * FROM test_encrypted_data WHERE id = $1").Args(d.ID)
	_, err := m.Get(context.TODO(), db, query, &d1, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)

	isValid, err := m.CheckSignature(d1, d1.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d.ID, d1.ID)
	require.Equal(t, d.Data, d1.Data)
	require.Equal(t, "sensitive-data", d1.SensitiveData)
	require.Equal(t, "another-sensitive-data", d1.AnotherSensitiveData)

	// Test updates
	d.SensitiveData = "sensitive--data"
	d.AnotherSensitiveData = "another-sensitive-data"

	require.NoError(t, m.UpdateAndSign(context.TODO(), db, &d))
	assert.Equal(t, sdk.PasswordPlaceholder, d.SensitiveData)
	assert.Equal(t, sdk.PasswordPlaceholder, d.AnotherSensitiveData)

	query = gorpmapper.NewQuery("select * from test_encrypted_data where id = $1").Args(d.ID)

	var d2 gorpmapper.TestEncryptedData
	_, err = m.Get(context.TODO(), db, query, &d2)
	require.NoError(t, err)

	isValid, err = m.CheckSignature(d2, d2.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d.ID, d2.ID)
	require.Equal(t, d.Data, d2.Data)
	require.NotEqual(t, "sensitive--data", d2.SensitiveData)
	require.Equal(t, sdk.PasswordPlaceholder, d2.SensitiveData)
	require.NotEqual(t, "another-sensitive-data", d2.AnotherSensitiveData)
	require.Equal(t, sdk.PasswordPlaceholder, d2.AnotherSensitiveData)

	_, err = m.Get(context.TODO(), db, query, &d2, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)

	isValid, err = m.CheckSignature(d2, d2.Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d.ID, d2.ID)
	require.Equal(t, d.Data, d2.Data)
	require.Equal(t, "sensitive--data", d2.SensitiveData)
	require.Equal(t, "another-sensitive-data", d2.AnotherSensitiveData)
}

func TestEncryption_Multiple(t *testing.T) {
	m := gorpmapper.New()
	m.Register(m.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeAPI)

	var d1 = gorpmapper.TestEncryptedData{
		Data:                 "data-1",
		SensitiveData:        "sensitive-data-1",
		AnotherSensitiveData: "another-sensitive-data-1",
	}
	require.NoError(t, m.InsertAndSign(context.TODO(), db, &d1))

	var d2 = gorpmapper.TestEncryptedData{
		Data:                 "data-2",
		SensitiveData:        "sensitive-data-2",
		AnotherSensitiveData: "another-sensitive-data-2",
	}
	require.NoError(t, m.InsertAndSign(context.TODO(), db, &d2))

	// Test that GetAll replaces encrypted values with placeholders
	query := gorpmapper.NewQuery("SELECT * FROM test_encrypted_data WHERE id IN ($1, $2) ORDER BY id").Args(d1.ID, d2.ID)
	var dslice []gorpmapper.TestEncryptedData
	require.NoError(t, m.GetAll(context.TODO(), db, query, &dslice))
	require.Len(t, dslice, 2)

	require.Equal(t, d1.ID, dslice[0].ID)
	require.Equal(t, "data-1", dslice[0].Data)
	require.Equal(t, sdk.PasswordPlaceholder, dslice[0].SensitiveData)
	require.Equal(t, sdk.PasswordPlaceholder, dslice[0].AnotherSensitiveData)

	require.Equal(t, d2.ID, dslice[1].ID)
	require.Equal(t, "data-2", dslice[1].Data)
	require.Equal(t, sdk.PasswordPlaceholder, dslice[1].SensitiveData)
	require.Equal(t, sdk.PasswordPlaceholder, dslice[1].AnotherSensitiveData)

	// Test that GetAll replaces encrypted values with clearValue if WithDecryption options is used
	query = gorpmapper.NewQuery("SELECT * FROM test_encrypted_data WHERE id IN ($1, $2) ORDER BY id").Args(d1.ID, d2.ID)
	dslice = []gorpmapper.TestEncryptedData{}
	require.NoError(t, m.GetAll(context.TODO(), db, query, &dslice, gorpmapping.GetOptions.WithDecryption))
	require.Len(t, dslice, 2)

	isValid, err := m.CheckSignature(dslice[0], dslice[0].Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	isValid, err = m.CheckSignature(dslice[1], dslice[1].Signature)
	require.NoError(t, err)
	require.True(t, isValid)

	require.Equal(t, d1.ID, dslice[0].ID)
	require.Equal(t, "data-1", dslice[0].Data)
	require.Equal(t, "sensitive-data-1", dslice[0].SensitiveData)
	require.Equal(t, "another-sensitive-data-1", dslice[0].AnotherSensitiveData)

	require.Equal(t, d2.ID, dslice[1].ID)
	require.Equal(t, "data-2", dslice[1].Data)
	require.Equal(t, "sensitive-data-2", dslice[1].SensitiveData)
	require.Equal(t, "another-sensitive-data-2", dslice[1].AnotherSensitiveData)
}
