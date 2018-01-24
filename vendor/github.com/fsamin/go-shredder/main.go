package shredder

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
)

func newUUID() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return ""
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

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
		id = newUUID()
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
