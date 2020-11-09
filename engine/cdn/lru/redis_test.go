package lru

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestRedisLRU(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
	r, err := NewRedisLRU(db.DbMap, 100, cfg["redisHost"], cfg["redisPassword"])
	require.NoError(t, err)

	l, _ := r.Len()
	for i := 0; i < l; i++ {
		_ = r.RemoveOldest()
	}

	item1 := sdk.CDNItem{
		Size:       45,
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		APIRefHash: sdk.UUID(),
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &item1))
	item2 := sdk.CDNItem{
		Size:       43,
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		APIRefHash: sdk.UUID(),
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &item2))
	item3 := sdk.CDNItem{
		Size:       20,
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		APIRefHash: sdk.UUID(),
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &item3))

	// Add first item
	writer1 := r.NewWriter(item1.ID)
	_, err = io.Copy(writer1, strings.NewReader("this is the first line\nthis is the second line\nthis is the third line\n"))
	require.NoError(t, writer1.Close())
	require.NoError(t, err)
	length, err := r.Len()
	require.NoError(t, err)
	require.Equal(t, 1, length)
	size, err := r.Size()
	require.NoError(t, err)
	require.Equal(t, int64(45), size)

	// Get first item
	reader1 := r.NewReader(item1.ID, sdk.CDNReaderFormatText, 0, 2, 0)
	buf1 := new(strings.Builder)
	_, err = io.Copy(buf1, reader1)
	require.NoError(t, reader1.Close())
	require.NoError(t, err)
	require.Equal(t, "this is the first line\nthis is the second line\n", buf1.String())
	reader2 := r.NewReader(item1.ID, sdk.CDNReaderFormatText, 0, 0, 0)
	buf2 := new(strings.Builder)
	_, err = io.Copy(buf2, reader2)
	require.NoError(t, reader2.Close())
	require.NoError(t, err)
	require.Equal(t, "this is the first line\nthis is the second line\nthis is the third line\n", buf2.String())

	// Add second item with lzast line not ending by end of line char
	writer2 := r.NewWriter(item2.ID)
	_, err = io.Copy(writer2, strings.NewReader("this is the first line\nthis is the second line"))
	require.NoError(t, err)
	reader3 := r.NewReader(item2.ID, sdk.CDNReaderFormatText, 0, 0, 0)
	buf3 := new(strings.Builder)
	_, err = io.Copy(buf3, reader3)
	require.NoError(t, reader3.Close())
	require.NoError(t, err)
	require.Equal(t, "this is the first line\n", buf3.String())
	// close the writer should add the last line
	require.NoError(t, writer2.Close())
	reader4 := r.NewReader(item2.ID, sdk.CDNReaderFormatText, 0, 0, 0)
	buf4 := new(strings.Builder)
	_, err = io.Copy(buf4, reader4)
	require.NoError(t, reader4.Close())
	require.NoError(t, err)
	require.Equal(t, "this is the first line\nthis is the second line\n", buf4.String())
	length, err = r.Len()
	require.NoError(t, err)
	require.Equal(t, 2, length)
	size, err = r.Size()
	require.NoError(t, err)
	require.Equal(t, int64(88), size)

	// Add third item
	writer3 := r.NewWriter(item3.ID)
	_, err = io.Copy(writer3, strings.NewReader("this is the third value\n"))
	require.NoError(t, writer3.Close())
	require.NoError(t, err)
	length, err = r.Len()
	require.NoError(t, err)
	require.Equal(t, 3, length)

	// Remove oldest
	cont, err := r.eviction()
	require.NoError(t, err)
	require.True(t, cont)
	cont, err = r.eviction()
	require.NoError(t, err)
	require.False(t, cont)

	length, err = r.Len()
	require.NoError(t, err)
	require.Equal(t, 2, length)

	size, err = r.Size()
	require.NoError(t, err)
	require.Equal(t, int64(63), size)
}
