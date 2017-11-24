package shredder

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/satori/go.uuid"
)

// ShredFile shreds a file content. See Shred for more details
func ShredFile(filename string, id string, opts *Opts) (Chunks, error) {
	ctx := &Ctx{
		UUID:        id + "&filename=" + filename,
		ContentType: FileContentType,
		Opts:        opts,
	}

	var errr error
	ctx.content, errr = ioutil.ReadFile(filename)
	if errr != nil {
		return nil, errr
	}

	return shred(ctx)
}

// Shred shreds a byte array into a an array of chunks according to options. You can pass nil as option, chunks size will be 512 bytes.
// You can define Encryption option such as GPG or AES. See GPGEncryption and AESEncryption structures.
func Shred(content []byte, id string, opts *Opts) (Chunks, error) {
	if id == "" {
		id = uuid.NewV4().String()
	}
	ctx := &Ctx{
		UUID:        id,
		ContentType: BytesContentType,
		Opts:        opts,
		content:     content,
	}
	return shred(ctx)
}

// Filter filters a list of chunks and returns a map of chunks according to their context ID.
func Filter(chunks Chunks) map[string]Chunks {
	res := map[string]Chunks{}
	for _, c := range chunks {
		_, ok := res[c.Ctx.UUID]
		if !ok {
			res[c.Ctx.UUID] = []Chunk{}
		}
		res[c.Ctx.UUID] = append(res[c.Ctx.UUID], c)
	}
	return res
}

// Reassemble computes a list of chunks, eventually decrypt the content (according to options)
func Reassemble(s Chunks, opts *Opts) (*Ctx, error) {
	sort.Sort(s)
	//Check id and filenames
	ctx := s[0].Ctx
	ctx.Opts = opts
	for _, c := range s {
		if c.Ctx.UUID != ctx.UUID {
			return nil, fmt.Errorf("Chunks doesn't match")
		}
	}
	//Check chunks
	if ctx.ChunksNumber != len(s) {
		return nil, fmt.Errorf("Missing chunks (%d %d)", ctx.ChunksNumber, len(s))
	}

	//concat all chunks content
	ctx.content = []byte{}
	for _, c := range s {
		ctx.content = append(ctx.content, c.Data...)
	}

	if ctx.Opts != nil {
		if ctx.Opts.AESEncryption != nil {
			content, err := AESDecrypt(ctx.Opts.AESEncryption.Key, bytes.NewBuffer(ctx.Bytes()))
			if err != nil {
				return nil, err
			}

			ctx.content, err = ioutil.ReadAll(content)
			if err != nil {
				return nil, err
			}
		} else if ctx.Opts.GPGEncryption != nil {
			content, err := GPGDecrypt(ctx.Opts.GPGEncryption.PrivateKey, ctx.Opts.GPGEncryption.Passphrase, bytes.NewBuffer(ctx.Bytes()))
			if err != nil {
				return nil, err
			}
			ctx.content, err = ioutil.ReadAll(content)
			if err != nil {
				return nil, err
			}
		}
	}

	return ctx, nil
}
