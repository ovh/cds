package cdn

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook/graylog"
)

// The tail is the only part of an oversized frame that can name its sender, so it must survive a
// message far larger than the ring itself. A naive implementation shifting the buffer on every
// byte would also make an oversized message quadratic, which is why this is a ring.
func TestDroppedTail_KeepsTheLastBytes(t *testing.T) {
	var d droppedTail
	require.Empty(t, d.bytes(), "an untouched tail carries nothing")

	for _, c := range []byte("abc") {
		d.add(c)
	}
	require.Equal(t, "abc", string(d.bytes()), "below the bound the whole message is kept")

	d.reset()
	for i := 0; i < maxDroppedTailSize*3; i++ {
		d.add(byte('0' + i%10))
	}
	got := d.bytes()
	require.Len(t, got, maxDroppedTailSize, "the ring never grows past its bound")
	// The last byte written must be the last byte returned.
	require.Equal(t, byte('0'+(maxDroppedTailSize*3-1)%10), got[len(got)-1])

	d.reset()
	require.Empty(t, d.bytes(), "reset drops the previous message")
}

// This is the point of the whole change: an operator reading the drop warning gets the job to
// fix, not the address of the proxy in front of the intake.
func TestDescribeDroppedMessage(t *testing.T) {
	key, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)
	signer, err := jws.NewHMacSigner(key)
	require.NoError(t, err)

	frame := func(sign cdn.Signature) []byte {
		token, err := jws.Sign(signer, sign)
		require.NoError(t, err)
		m := graylog.Message{
			Version: "1.1",
			Host:    "host",
			Short:   "short",
			Full:    strings.Repeat("x", 512),
			Extra:   map[string]interface{}{"_" + cdslog.ExtraFieldSignature: token},
		}
		b, err := json.Marshal(&m)
		require.NoError(t, err)
		return b
	}

	t.Run("v2 worker", func(t *testing.T) {
		got := describeDroppedMessage(frame(cdn.Signature{
			ProjectKey: "KEY", WorkflowName: "wf", RunJobID: "run-job-id", JobName: "job",
			Worker: &cdn.SignatureWorker{WorkerName: "worker", StepName: "step"},
		}))
		require.Contains(t, got, "project=KEY")
		require.Contains(t, got, "workflow=wf")
		require.Contains(t, got, "run_job_id=run-job-id")
		require.Contains(t, got, "worker=worker")
		require.Contains(t, got, "step=step")
	})

	t.Run("v1 worker", func(t *testing.T) {
		got := describeDroppedMessage(frame(cdn.Signature{
			ProjectKey: "KEY", WorkflowName: "wf", JobID: 42,
			Worker: &cdn.SignatureWorker{WorkerName: "worker"},
		}))
		require.Contains(t, got, "job_id=42")
		require.Contains(t, got, "worker=worker")
	})

	t.Run("service log sent by a hatchery", func(t *testing.T) {
		got := describeDroppedMessage(frame(cdn.Signature{
			ProjectKey: "KEY", WorkflowName: "wf", RunJobID: "run-job-id",
			HatcheryService: &cdn.SignatureHatcheryService{HatcheryName: "hatch", ServiceName: "svc"},
		}))
		require.Contains(t, got, "hatchery=hatch")
		require.Contains(t, got, "service=svc")
	})

	// A frame whose tail was cut before the signature, or that is not a gelf message at all,
	// must degrade to nothing rather than to a wrong or partial identification.
	t.Run("no readable signature", func(t *testing.T) {
		require.Empty(t, describeDroppedMessage(nil))
		require.Empty(t, describeDroppedMessage([]byte(`{"version":"1.1","short_message":"x"}`)))
		require.Empty(t, describeDroppedMessage([]byte(`{"_`+cdslog.ExtraFieldSignature+`":"not-a-token"}`)))
	})
}
