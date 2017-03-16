package shredder

import (
	"errors"
	"path/filepath"
	"strings"
)

const (
	BytesContentType = "bytes"
	FileContentType  = "file"
)

//Ctx is a context for a shredding
type Ctx struct {
	UUID         string
	ContentType  string
	content      []byte
	Opts         *Opts
	ChunksNumber int
}

// Bytes returns the content as a string
func (ctx *Ctx) String() string {
	return string(ctx.content)
}

// Bytes returns the content as a byte array
func (ctx *Ctx) Bytes() []byte {
	return ctx.content
}

func (ctx *Ctx) GetUUID() string {
	if ctx.ContentType != FileContentType || !strings.Contains(ctx.UUID, "&filename=") {
		return ctx.UUID
	}
	uuid := strings.Split(ctx.UUID, "&filename=")[0]
	return uuid
}

// File returns the filename and one content according to the context content type.
// If the context content type is not a file, an error will be returned
func (ctx *Ctx) File() (string, []byte, error) {
	if ctx.ContentType != FileContentType || !strings.Contains(ctx.UUID, "&filename=") {
		return "", nil, errors.New("Context is not a file content")
	}
	filename := strings.Split(ctx.UUID, "&filename=")[1]
	return filepath.Base(string(filename)), ctx.content, nil
}

// Opts is here to set option on shredder like Encryption of chunksize
type Opts struct {
	AESEncryption *AESEncryption
	GPGEncryption *GPGEncryption
	ChunkSize     int64
}

// AESEncryption use https://golang.org/pkg/crypto/aes/ for file content encryption/decryption
type AESEncryption struct {
	Key []byte
}

// GPGEncryption use GPG to file content encryption/decryption
type GPGEncryption struct {
	PrivateKey []byte
	Passphrase []byte
	PublicKey  []byte
}

// Chunk is a piece of schredded file
type Chunk struct {
	Ctx    *Ctx
	Data   []byte
	Offset int
}

// Chunks is an array of chunks
type Chunks []Chunk

func (s Chunks) Len() int {
	return len(s)
}
func (s Chunks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Chunks) Less(i, j int) bool {
	return s[i].Offset < s[j].Offset
}

// Context returns the context
func (s Chunks) Context() *Ctx {
	return s[0].Ctx
}

// Completed returns true when chunks is made of all needed chunks
func (s Chunks) Completed() bool {
	return len(s) == s.Context().ChunksNumber
}

// Delete removes a chunks from a chunks list
func (s *Chunks) Delete(c Chunk) {
	for i := range *s {
		if (*s)[i].Offset == c.Offset && string((*s)[i].Data) == string(c.Data) {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return
		}
	}
}
