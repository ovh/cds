package nfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/keyloader"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

// To run export the mentionned env variables: NFS_HOST, NFS_PARTITION
// If not set, the test is skipped
func TestNfs(t *testing.T) {
	t.Run("group", func(t *testing.T) {
		t.Run("Test1", TestNFSReadWrite)
		t.Run("Test2", TestNFSReadWrite)
		t.Run("Test3", TestNFSReadWrite)
		t.Run("Test4", TestNFSReadWrite)
		t.Run("Test5", TestNFSReadWrite)
		t.Run("Test6", TestNFSReadWrite)
		t.Run("Test7", TestNFSReadWrite)
		t.Run("Test8", TestNFSReadWrite)
		t.Run("Test9", TestNFSReadWrite)
		t.Run("Test10", TestNFSReadWrite)
		t.Run("Test11", TestNFSReadWrite)
		t.Run("Test12", TestNFSReadWrite)
		t.Run("Test13", TestNFSReadWrite)
		t.Run("Test14", TestNFSReadWrite)
		t.Run("Test15", TestNFSReadWrite)
		t.Run("Test16", TestNFSReadWrite)
		t.Run("Test17", TestNFSReadWrite)
		t.Run("Test18", TestNFSReadWrite)
		t.Run("Test19", TestNFSReadWrite)
		t.Run("Test20", TestNFSReadWrite)
	})

}

func TestNFSReadWrite(t *testing.T) {
	t.Parallel()
	log.Factory = log.NewTestingWrapper(t)
	t.Logf("%d", time.Now().Unix())
	nfsHost := os.Getenv("NFS_HOST")
	nfsTargetPath := os.Getenv("NFS_PARTITION")
	if nfsHost == "" || nfsTargetPath == "" {
		t.Logf("No env variables, skip the test")
		t.SkipNow()
	} else {
		t.Logf("[%s][%s]", nfsHost, nfsTargetPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	d := storage.GetDriver("nfs-buffer")
	require.NotNil(t, d)
	bd, is := d.(storage.BufferUnit)
	require.True(t, is)
	bd.New(sdk.NewGoRoutines(ctx), storage.AbstractUnitConfig{})
	err := bd.Init(ctx, &storage.NFSBufferConfiguration{
		Host:            nfsHost,
		TargetPartition: nfsTargetPath,
		GroupID:         0,
		UserID:          0,
		Encryption: []*keyloader.KeyConfig{
			{
				Cipher:     aesgcm.CipherName,
				Identifier: "nfs-bugger-id",
				Key:        "clejesuisuneclejesuisunecleclef",
				Sealed:     false,
			},
		},
	}, storage.CDNBufferTypeFile)
	require.NoError(t, err, "unable to initialiaze nfs driver")
	itemUnit := sdk.CDNItemUnit{
		Type:    sdk.CDNTypeItemRunResult,
		Locator: sdk.RandomString(10),
		Item: &sdk.CDNItem{
			Type:       sdk.CDNTypeItemRunResult,
			APIRefHash: sdk.RandomString(10),
		},
	}
	fileBufferUnit := bd.(storage.FileBufferUnit)

	w, err := fileBufferUnit.NewWriter(context.TODO(), itemUnit)
	require.NoError(t, err)
	require.NotNil(t, w)

	textContent := fmt.Sprintf("%s-%s", sdk.RandomString(10), sdk.RandomString(10))
	_, err = w.Write([]byte(textContent))
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	r, err := fileBufferUnit.NewReader(context.TODO(), itemUnit)
	require.NoError(t, err)
	require.NotNil(t, r)

	btes, err := io.ReadAll(r)
	require.NoError(t, err)
	err = r.Close()
	require.NoError(t, err)

	require.Equal(t, textContent, string(btes))
	t.Logf("%d", time.Now().Unix())
}
