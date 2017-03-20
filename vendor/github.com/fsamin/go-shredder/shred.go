package shredder

import (
	"bytes"
)

const (
	defaultChunkSize = int64(512)
)

func shred(ctx *Ctx) ([]Chunk, error) {
	var content = ctx.content
	if ctx.Opts != nil {
		if ctx.Opts.AESEncryption != nil {
			b, err := aesEncrypt(ctx.Opts.AESEncryption, bytes.NewBuffer(content))
			if err != nil {
				return nil, err
			}
			content = b
		} else if ctx.Opts.GPGEncryption != nil {
			b, err := gpgEncrypt(ctx.Opts.GPGEncryption, bytes.NewBuffer(content))
			if err != nil {
				return nil, err
			}
			content = b
		}
	}
	return shredContent(ctx, content), nil
}

func shredContent(ctx *Ctx, content []byte) []Chunk {
	var size = defaultChunkSize
	if ctx.Opts != nil && ctx.Opts.ChunkSize != 0 {
		size = ctx.Opts.ChunkSize
	}
	var chunks []Chunk
	var offset int64
	var offSetNumber int
	for {
		data := make([]byte, size)
		var length = offset + size
		//Resize last chunk
		if offset+size > int64(len(content)) {
			length = int64(len(content))
			data = make([]byte, len(content[offset:length]))
		}
		n := copy(data, content[offset:length])
		if n == 0 {
			break
		}
		offset += int64(n)
		c := Chunk{
			Ctx:    ctx,
			Offset: offSetNumber,
			Data:   data,
		}
		chunks = append(chunks, c)
		offSetNumber++
	}
	ctx.ChunksNumber = len(chunks)
	return chunks
}
