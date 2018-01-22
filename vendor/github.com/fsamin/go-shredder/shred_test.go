package shredder

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_shredContent(t *testing.T) {
	type args struct {
		ctx     *Ctx
		content []byte
	}
	type testcase struct {
		name             string
		args             args
		wantChunksNumber int
		wantChunks       []Chunk
	}

	ctx := &Ctx{
		Opts: &Opts{
			ChunkSize: int64(100),
		},
	}

	b1, err := ioutil.ReadFile("fixtures/99bytesfile")
	if err != nil {
		t.Fatal(err)
	}
	test1 := testcase{
		name: "shred 99 bytes file with a chunk size of 100",
		args: args{
			ctx:     ctx,
			content: b1,
		},
		wantChunksNumber: 1,
		wantChunks: []Chunk{
			Chunk{
				Ctx:    ctx,
				Data:   b1,
				Offset: 0,
			},
		},
	}

	b2, err := ioutil.ReadFile("fixtures/100bytesfile")
	if err != nil {
		t.Fatal(err)
	}
	test2 := testcase{
		name: "shred 100 bytes file with a chunk size of 100",
		args: args{
			ctx:     ctx,
			content: b2,
		},
		wantChunksNumber: 1,
		wantChunks: []Chunk{
			Chunk{
				Ctx:    ctx,
				Data:   b2,
				Offset: 0,
			},
		},
	}
	b3, err := ioutil.ReadFile("fixtures/101bytesfile")
	if err != nil {
		t.Fatal(err)
	}
	test3 := testcase{
		name: "shred 101 bytes file with a chunk size of 100",
		args: args{
			ctx:     ctx,
			content: b3,
		},
		wantChunksNumber: 2,
		wantChunks: []Chunk{
			Chunk{
				Ctx:    ctx,
				Data:   b3[:100],
				Offset: 0,
			},
			Chunk{
				Ctx:    ctx,
				Data:   b3[100:],
				Offset: 1,
			},
		},
	}

	tests := []testcase{test1, test2, test3}
	for _, tt := range tests {
		got := shredContent(tt.args.ctx, tt.args.content)
		assert.Equal(t, tt.wantChunksNumber, len(got))
		assert.EqualValues(t, tt.wantChunks, got)
	}
}
