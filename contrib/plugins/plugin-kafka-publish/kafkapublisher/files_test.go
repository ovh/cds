package kafkapublisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKafkaMessages(t *testing.T) {
	f, _ := OpenFile("files.go")

	msgs, err := f.KafkaMessages(100)
	assert.NoError(t, err)

	filename := "files.go"
	var f1 *File
	var chunks Chunks
	for _, msg := range msgs {
		g, c, err := ReadBytes(msg)
		assert.NoError(t, err)
		assert.NotNil(t, g)
		assert.NotNil(t, c)
		assert.Equal(t, filename, g.Name)
		assert.Equal(t, filename, c.Filename)
		if f1 == nil {
			f1 = g
		}
		chunks = append(chunks, *c)
	}
	err = chunks.Reassemble(f1)
	assert.NoError(t, err)

	assert.EqualValues(t, f.Content.Bytes(), f1.Content.Bytes())

}
