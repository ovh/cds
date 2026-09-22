package internal

import (
	"context"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
)

func TestSendLogTruncatesOversizedLine(t *testing.T) {
	wk := &CurrentWorker{}
	secret, err := jws.NewRandomSymmetricKey(64)
	require.NoError(t, err)
	wk.signer, err = jws.NewHMacSigner(secret)
	require.NoError(t, err)
	wk.blur, err = sdk.NewBlur([]string{"the-secret"})
	require.NoError(t, err)
	wk.currentJobV2.runJob = &sdk.V2WorkflowRunJob{ID: "the-run-job", Region: "build"}

	l := logrus.New()
	l.SetOutput(io.Discard)
	entries := logrustest.NewLocal(l)
	wk.SetGelfLogger(nil, l)

	// One ASCII byte then 3-byte runes: the cut lands inside a rune and must step back to its start
	wk.SendLog(context.TODO(), workerruntime.LevelInfo, "a"+strings.Repeat("€", cdslog.MaxLogLineSize/3+10))
	require.Len(t, entries.Entries, 1)
	msg := entries.LastEntry().Message
	require.True(t, strings.HasSuffix(msg, "...truncated\n"))
	require.True(t, utf8.ValidString(msg))
	require.Len(t, msg, cdslog.MaxLogLineSize-2+len("...truncated\n"))

	entries.Reset()
	wk.SendLog(context.TODO(), workerruntime.LevelInfo, "short line\n")
	require.Len(t, entries.Entries, 1)
	require.Equal(t, "short line\n", entries.LastEntry().Message)
}
