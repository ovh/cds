package kafkapublisher

import (
	"bytes"
	"fmt"
	"io"
	"sort"
)

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
// Even if Read returns n < len(p), it may use all of p as scratch space during the call. If some data is available but not len(p) bytes, Read conventionally returns what is available instead of waiting for more.
// When Read encounters an error or end-of-file condition after successfully reading n > 0 bytes, it returns the number of bytes read. It may return the (non-nil) error from the same call or return the error (and n == 0) from a subsequent call. An instance of this general case is that a Reader
// returning a non-zero number of bytes at the end of the input stream may return either err == EOF or err == nil. The next Read should return 0, EOF.
// Callers should always process the n > 0 bytes returned before considering the error err. Doing so correctly handles I/O errors that happen after reading some bytes and also both of the allowed EOF behaviors.
func (c *Chunk) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if len(c.Content) <= len(p) {
		return copy(p, c.Content), io.EOF
	}

	return copy(p, c.Content[:len(p)]), nil
}

//Close close the chunk
func (c *Chunk) Close() error {
	return nil
}
func (s Chunks) Len() int {
	return len(s)
}
func (s Chunks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Chunks) Less(i, j int) bool {
	return s[i].Offset < s[j].Offset
}

//IsFileComplete checks if the chunks are enough to build the file
func (s Chunks) IsFileComplete(f *File) bool {
	return f.ChunksNumber == len(s)
}

//Reassemble compute the file content from a chunks list
func (s Chunks) Reassemble(f *File) error {
	sort.Sort(s)
	//Check id and filenames
	for _, c := range s {
		if c.FileID != f.ID || c.Filename != f.Name {
			return fmt.Errorf("Chunks doesn't match")
		}
	}

	//Check chunks
	if f.ChunksNumber != len(s) {
		return fmt.Errorf("Missing chunks for file %s (%d %d)", f.Name, f.ChunksNumber, len(s))
	}

	//Construct file
	var content = []byte{}
	for _, c := range s {
		content = append(content, c.Content...)
	}
	f.Content = bytes.NewBuffer(content)
	return nil
}
