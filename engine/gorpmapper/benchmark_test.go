package gorpmapper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

/**

Benchmark resuls on 09/03/19

BenchmarkGetWithoutDecryption: 540722 ns/op 6403 B/op 191 allocs/op
BenchmarkGetWithDecryption: 1142721 ns/op 88750 B/op 802 allocs/op
BenchmarkInsertWithoutSignature: 1579614 ns/op 84759 B/op 726 allocs/op
BenchmarkInsertWithSignature: 2911001 ns/op 127447 B/op 1049 allocs/op
BenchmarkCheckSignature: 81943 ns/op 41161 B/op 290 allocs/op

Benchmark resuls on 20/03/19 - (with more data encrypted per op due to extras)

BenchmarkGetWithoutDecryption: 719757 ns/op 6353 B/op 191 allocs/op
BenchmarkGetWithDecryption: 960652 ns/op 9287 B/op 280 allocs/op
BenchmarkInsertWithoutSignature: 2002177 ns/op 5342 B/op 200 allocs/op
BenchmarkInsertWithSignature: 2452006 ns/op 8253 B/op 254 allocs/op
BenchmarkCheckSignature: 5283 ns/op 1344 B/op 21 allocs/op

*/

func BenchmarkGetWithoutDecryption(b *testing.B) {
	m := gorpmapper.New()

	m.Register(m.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	log.SetLogger(b)
	db, _, _, end := test.SetupPGToCancel(b, m, sdk.TypeAPI)
	b.Cleanup(end)

	var d = gorpmapper.TestEncryptedData{
		Data:                 "data",
		SensitiveData:        "sensitive-data",
		AnotherSensitiveData: "another-sensitive-data",
	}

	require.NoError(b, m.Insert(db, &d))

	for n := 0; n < b.N; n++ {
		query := gorpmapper.NewQuery("select * from test_encrypted_data where id = $1").Args(d.ID)
		var d2 gorpmapper.TestEncryptedData
		_, err := m.Get(context.TODO(), db, query, &d2)
		require.NoError(b, err)
	}
}

func BenchmarkGetWithDecryption(b *testing.B) {
	m := gorpmapper.New()

	m.Register(m.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	log.SetLogger(b)
	db, _, _, end := test.SetupPGToCancel(b, m, sdk.TypeAPI)
	b.Cleanup(end)

	var d = gorpmapper.TestEncryptedData{
		Data:                 "data",
		SensitiveData:        "sensitive-data",
		AnotherSensitiveData: "another-sensitive-data",
	}

	require.NoError(b, m.Insert(db, &d))

	for n := 0; n < b.N; n++ {
		query := gorpmapper.NewQuery("SELECT * FROM test_encrypted_data WHERE id = $1").Args(d.ID)
		var d2 gorpmapper.TestEncryptedData
		_, err := m.Get(context.TODO(), db, query, &d2, gorpmapping.GetOptions.WithDecryption)
		require.NoError(b, err)
	}
}

func BenchmarkInsertWithoutSignature(b *testing.B) {
	m := gorpmapper.New()

	m.Register(m.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	log.SetLogger(b)
	db, _, _, end := test.SetupPGToCancel(b, m, sdk.TypeAPI)
	b.Cleanup(end)

	for n := 0; n < b.N; n++ {
		var d = gorpmapper.TestEncryptedData{
			Data:                 "data",
			SensitiveData:        "sensitive-data",
			AnotherSensitiveData: "another-sensitive-data",
		}

		require.NoError(b, m.Insert(db, &d))
	}
}

func BenchmarkInsertWithSignature(b *testing.B) {
	m := gorpmapper.New()

	m.Register(m.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	log.SetLogger(b)
	db, _, _, end := test.SetupPGToCancel(b, m, sdk.TypeAPI)
	b.Cleanup(end)

	for n := 0; n < b.N; n++ {
		var d = gorpmapper.TestEncryptedData{
			Data:                 "data",
			SensitiveData:        "sensitive-data",
			AnotherSensitiveData: "another-sensitive-data",
		}

		require.NoError(b, m.InsertAndSign(context.TODO(), db, &d))
	}
}

func BenchmarkCheckSignature(b *testing.B) {
	m := gorpmapper.New()

	m.Register(m.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	log.SetLogger(b)
	db, _, _, end := test.SetupPGToCancel(b, m, sdk.TypeAPI)
	b.Cleanup(end)

	var d = gorpmapper.TestEncryptedData{
		Data:                 "data",
		SensitiveData:        "sensitive-data",
		AnotherSensitiveData: "another-sensitive-data",
	}

	require.NoError(b, m.InsertAndSign(context.TODO(), db, &d))

	for n := 0; n < b.N; n++ {
		_, _ = m.CheckSignature(d, d.GetSignature())
	}
}
