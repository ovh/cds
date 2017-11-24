package shredder

import (
	"bytes"
	"io/ioutil"
)

const (
	defaultChunkSize = int64(512)
)

func shred(ctx *Ctx) ([]Chunk, error) {
	var content = ctx.content
	if ctx.Opts != nil {
		if ctx.Opts.AESEncryption != nil {
			b, err := AESEncrypt(ctx.Opts.AESEncryption.Key, bytes.NewBuffer(content))
			if err != nil {
				return nil, err
			}
			content, err = ioutil.ReadAll(b)
			if err != nil {
				return nil, err
			}
		} else if ctx.Opts.GPGEncryption != nil {
			b, err := GPGEncrypt(ctx.Opts.GPGEncryption.PublicKey, bytes.NewBuffer(content))
			if err != nil {
				return nil, err
			}
			content, err = ioutil.ReadAll(b)
			if err != nil {
				return nil, err
			}
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
