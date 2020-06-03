package kafkapublisher

import (
	"encoding/base64"
	"io/ioutil"
	"testing"

	"github.com/fsamin/go-shredder"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
)

func TestKafkaMessages(t *testing.T) {
	chunks, err := shredder.ShredFile("chunks.go", "", nil)
	test.NoError(t, err)

	msgs, err := KafkaMessages(chunks)
	test.NoError(t, err)

	filenameB64 := base64.StdEncoding.EncodeToString([]byte("chunks.go"))
	var chunks2 shredder.Chunks
	for i, msg := range msgs {
		c, err := ReadBytes(msg)
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.Equal(t, filenameB64, c.Ctx.UUID)
		assert.Equal(t, shredder.FileContentType, c.Ctx.ContentType)
		assert.Equal(t, 7, c.Ctx.ChunksNumber)
		assert.Equal(t, i, c.Offset)
		assert.NotEmpty(t, c.Data)
		chunks2 = append(chunks2, *c)
	}
	ctx, err := shredder.Reassemble(chunks2, nil)
	test.NoError(t, err)

	btes, err := ioutil.ReadFile("chunks.go")
	test.NoError(t, err)

	assert.EqualValues(t, btes, ctx.Bytes())

}
