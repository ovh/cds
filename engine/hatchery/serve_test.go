package hatchery

import (
	"context"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
)

func TestSendServiceLogTruncatesOversizedLine(t *testing.T) {
	c := &Common{}
	secret, err := jws.NewRandomSymmetricKey(64)
	require.NoError(t, err)
	c.Signer, err = jws.NewHMacSigner(secret)
	require.NoError(t, err)

	l := logrus.New()
	l.SetOutput(io.Discard)
	entries := logrustest.NewLocal(l)
	c.ServiceLogger = l

	msg := cdslog.Message{
		Level: logrus.InfoLevel,
		Signature: cdn.Signature{
			RunJobID:        "the-run-job",
			HatcheryService: &cdn.SignatureHatcheryService{ServiceName: "db"},
		},
	}

	// One ASCII byte then 3-byte runes: the cut lands inside a rune and must step back to its start
	msg.Value = "a" + strings.Repeat("€", cdslog.MaxLogLineSize/3+10)
	c.SendServiceLog(context.TODO(), []cdslog.Message{msg}, false)
	require.Len(t, entries.Entries, 1)
	got := entries.LastEntry().Message
	require.True(t, strings.HasSuffix(got, "...truncated\n"))
	require.True(t, utf8.ValidString(got))
	require.Len(t, got, cdslog.MaxLogLineSize-2+len("...truncated\n"))

	entries.Reset()
	msg.Value = "short line\n"
	c.SendServiceLog(context.TODO(), []cdslog.Message{msg}, false)
	require.Len(t, entries.Entries, 1)
	require.Equal(t, "short line\n", entries.LastEntry().Message)
}
