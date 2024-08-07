// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
// inspired from github.com/gemnasium/logrus-graylog-hook

package graylog

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
)

type Reader struct {
	conn net.Conn
	ln   net.Listener
	addr string
}

func (r *Reader) Addr() string {
	return r.addr
}

func (r *Reader) Close() error {
	if r.ln != nil {
		if err := r.ln.Close(); err != nil {
			return err
		}
	}
	return r.conn.Close()
}

// FIXME: this will discard data if p isn't big enough to hold the
// full message.
func (r *Reader) Read(p []byte) (int, error) {
	msg, err := r.ReadMessage()
	if err != nil {
		return -1, err
	}

	var data string

	if msg.Full == "" {
		data = msg.Short
	} else {
		data = msg.Full
	}

	return strings.NewReader(data).Read(p)
}

func (r *Reader) ReadMessage() (*Message, error) {
	cBuf := make([]byte, ChunkSize)
	var (
		err        error
		n, length  int
		buf        bytes.Buffer
		cid, ocid  []byte
		seq, total uint8
		cHead      []byte
		cReader    io.Reader
		chunks     [][]byte
	)

	for got := 0; got < 128 && (total == 0 || got < int(total)); got++ {
		if n, err = r.conn.Read(cBuf); err != nil {
			return nil, fmt.Errorf("Read: %s", err)
		}
		cHead, cBuf = cBuf[:2], cBuf[:n]

		if bytes.Equal(cHead, magicChunked) {
			cid, seq, total = cBuf[2:2+8], cBuf[2+8], cBuf[2+8+1]
			if ocid != nil && !bytes.Equal(cid, ocid) {
				return nil, fmt.Errorf("out-of-band message %v (awaited %v)", cid, ocid)
			} else if ocid == nil {
				ocid = cid
				chunks = make([][]byte, total)
			}
			n = len(cBuf) - chunkedHeaderLen
			chunks[seq] = append(make([]byte, 0, n), cBuf[chunkedHeaderLen:]...)
			length += n
		} else { //not chunked
			if total > 0 {
				return nil, fmt.Errorf("out-of-band message (not chunked)")
			}
			break
		}
	}

	if length > 0 {
		if cap(cBuf) < length {
			cBuf = append(cBuf, make([]byte, 0, length-cap(cBuf))...)
		}
		cBuf = cBuf[:0]
		for i := range chunks {
			cBuf = append(cBuf, chunks[i]...)
		}
		cHead = cBuf[:2]
	}

	// the data we get from the wire is compressed
	if bytes.Equal(cHead, magicGzip) {
		cReader, err = gzip.NewReader(bytes.NewReader(cBuf))
	} else if cHead[0] == magicZlib[0] &&
		(int(cHead[0])*256+int(cHead[1]))%31 == 0 {
		// zlib is slightly more complicated, but correct
		cReader, err = zlib.NewReader(bytes.NewReader(cBuf))
	} else {
		return nil, fmt.Errorf("unknown magic: %x %v", cHead, cHead)
	}

	if err != nil {
		return nil, fmt.Errorf("NewReader: %s", err)
	}

	if _, err = io.Copy(&buf, cReader); err != nil {
		return nil, fmt.Errorf("io.Copy: %s", err)
	}

	msg := new(Message)
	if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %s", err)
	}

	return msg, nil
}
