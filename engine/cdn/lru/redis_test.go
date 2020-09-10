package lru

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/index"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestRedisLRU(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	db, cache := test.SetupPGWithMapper(t, m, sdk.TypeCDN)

	cdntest.ClearIndex(t, context.TODO(), m, db)

	a := AbstractLRU{
		db:      db.DbMap,
		maxSize: 100,
	}
	r := NewRedisLRU(a, cache)

	l, _ := r.Len()
	for i := 0; i < l; i++ {
		r.RemoveOldest()
	}

	item1 := index.Item{
		Size:       45,
		ID:         sdk.UUID(),
		Type:       index.TypeItemStepLog,
		APIRefHash: sdk.UUID(),
	}
	require.NoError(t, index.InsertItem(context.TODO(), m, db, &item1))
	item2 := index.Item{
		Size:       43,
		ID:         sdk.UUID(),
		Type:       index.TypeItemStepLog,
		APIRefHash: sdk.UUID(),
	}
	require.NoError(t, index.InsertItem(context.TODO(), m, db, &item2))
	item3 := index.Item{
		Size:       20,
		ID:         sdk.UUID(),
		Type:       index.TypeItemStepLog,
		APIRefHash: sdk.UUID(),
	}
	require.NoError(t, index.InsertItem(context.TODO(), m, db, &item3))

	// Add first item
	writer := r.NewWriter(item1.ID)
	_, err := io.Copy(writer, strings.NewReader("je suis la valeur"))
	writer.Close()
	require.NoError(t, err)

	length, err := r.Len()
	require.NoError(t, err)
	require.Equal(t, 1, length)

	size, err := r.Size()
	require.NoError(t, err)
	require.Equal(t, int64(45), size)

	// Add second item
	writer = r.NewWriter(item2.ID)
	_, err = io.Copy(writer, strings.NewReader("je suis la valeur 2"))
	writer.Close()
	require.NoError(t, err)

	length, err = r.Len()
	require.NoError(t, err)
	require.Equal(t, 2, length)

	size, err = r.Size()
	require.NoError(t, err)
	require.Equal(t, int64(88), size)

	// Get Item 1
	reader := r.NewReader(item1.ID, 0, 1)
	buf := new(strings.Builder)
	_, err = io.Copy(buf, reader)
	reader.Close()
	require.NoError(t, err)
	require.Equal(t, "je suis la valeur", buf.String())

	// Add third item
	writer = r.NewWriter(item3.ID)
	_, err = io.Copy(writer, strings.NewReader("je suis la valeur 3"))
	writer.Close()
	require.NoError(t, err)

	// Remove oldest
	cont, err := eviction(r)
	require.NoError(t, err)
	require.True(t, cont)
	cont, err = eviction(r)
	require.NoError(t, err)
	require.False(t, cont)

	length, err = r.Len()
	require.NoError(t, err)
	require.Equal(t, 2, length)

	size, err = r.Size()
	require.NoError(t, err)
	require.Equal(t, int64(65), size)

}
