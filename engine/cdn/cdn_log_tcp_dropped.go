package cdn

import (
	"bytes"
	"fmt"

	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
)

// maxDroppedTailSize bounds the tail kept from a message that is being discarded for being too
// large. Only the TAIL can name the sender: the gelf marshalling writes the message body first,
// with its oversized full_message, and appends the extra fields at the very end, so that is
// where the signature is. The head of such a frame carries nothing but content.
const maxDroppedTailSize = 4096

// droppedTail keeps the last maxDroppedTailSize bytes of the message currently being discarded.
// It is a ring: appending is O(1), which matters because an oversized message can be tens of
// megabytes and every byte of it goes through here.
type droppedTail struct {
	buf  []byte
	pos  int
	full bool
}

func (d *droppedTail) add(c byte) {
	if d.buf == nil {
		d.buf = make([]byte, maxDroppedTailSize)
	}
	d.buf[d.pos] = c
	d.pos++
	if d.pos == len(d.buf) {
		d.pos = 0
		d.full = true
	}
}

func (d *droppedTail) reset() {
	d.pos = 0
	d.full = false
}

// bytes returns the tail in write order.
func (d *droppedTail) bytes() []byte {
	if d.buf == nil {
		return nil
	}
	if !d.full {
		return d.buf[:d.pos]
	}
	out := make([]byte, 0, len(d.buf))
	out = append(out, d.buf[d.pos:]...)
	return append(out, d.buf[:d.pos]...)
}

// describeDroppedMessage names the sender of a discarded message from the tail of its frame.
// The signature is parsed WITHOUT verification: the message is dropped either way, this only
// turns an unusable remote address into the job that has to be fixed. It returns an empty
// string when the tail carries no readable signature, and the caller then logs nothing more.
func describeDroppedMessage(tail []byte) string {
	needle := []byte(`"_` + cdslog.ExtraFieldSignature + `":"`)
	i := bytes.Index(tail, needle)
	if i < 0 {
		return ""
	}
	rest := tail[i+len(needle):]
	j := bytes.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}

	var sign cdn.Signature
	if err := jws.UnsafeParse(string(rest[:j]), &sign); err != nil {
		return ""
	}

	switch {
	case sign.RunJobID != "":
		desc := fmt.Sprintf("project=%s workflow=%s run_job_id=%s job=%s", sign.ProjectKey, sign.WorkflowName, sign.RunJobID, sign.JobName)
		if sign.Worker != nil {
			desc += fmt.Sprintf(" worker=%s step=%s", sign.Worker.WorkerName, sign.Worker.StepName)
		}
		if sign.HatcheryService != nil {
			desc += fmt.Sprintf(" hatchery=%s service=%s", sign.HatcheryService.HatcheryName, sign.HatcheryService.ServiceName)
		}
		return desc
	case sign.JobID != 0:
		desc := fmt.Sprintf("project=%s workflow=%s job_id=%d", sign.ProjectKey, sign.WorkflowName, sign.JobID)
		if sign.Worker != nil {
			desc += fmt.Sprintf(" worker=%s step=%s", sign.Worker.WorkerName, sign.Worker.StepName)
		}
		return desc
	}
	return ""
}
