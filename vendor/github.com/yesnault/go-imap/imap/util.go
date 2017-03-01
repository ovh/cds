// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// Default debug logging configuration for new Client instances.
var (
	DefaultLogger  = log.New(os.Stderr, "[imap] ", log.Ltime)
	DefaultLogMask = LogNone
)

// prng is a deterministic pseudo-random number generator seeded using the
// system clock.
var prng = rand.New(&prngSource{src: rand.NewSource(time.Now().UnixNano())})

// gotest is set to true when the package is being executed by the test command.
// It causes the Client to use predictable tag ids for scripting.
var gotest = false

// debugLog handles all logging operations for Client and transport.
type debugLog struct {
	log  *log.Logger // Message destination
	mask LogMask     // Enabled message categories
}

// newDebugLog returns a new debugLog instance.
func newDebugLog(log *log.Logger, mask LogMask) *debugLog {
	if log == nil {
		log = DefaultLogger
	}
	return &debugLog{log, mask}
}

// SetLogger sets the destination of debug messages and returns the previous
// logger.
func (d *debugLog) SetLogger(log *log.Logger) *log.Logger {
	if d == nil {
		return nil
	}
	prev := d.log
	d.log = log
	return prev
}

// SetLogMask enables/disables debug message categories and returns the previous
// mask.
func (d *debugLog) SetLogMask(mask LogMask) LogMask {
	if d == nil {
		return LogNone
	}
	prev := d.mask
	d.mask = mask
	return prev
}

// Log formats its arguments using default formatting, analogous to Print(), and
// records the text in the debug log if logging is enabled for the specified
// mask.
func (d *debugLog) Log(mask LogMask, v ...interface{}) {
	if d != nil && d.mask&mask == mask {
		d.log.Output(2, fmt.Sprint(v...))
	}
}

// Logf formats its arguments according to the format, analogous to Printf(),
// and records the text in the debug log if logging is enabled for the specified
// mask.
func (d *debugLog) Logf(mask LogMask, format string, v ...interface{}) {
	if d != nil && d.mask&mask == mask {
		d.log.Output(2, fmt.Sprintf(format, v...))
	}
}

// Logln formats its arguments using default formatting, analogous to Println(),
// and records the text in the debug log if logging is enabled for the specified
// mask.
func (d *debugLog) Logln(mask LogMask, v ...interface{}) {
	if d != nil && d.mask&mask == mask {
		d.log.Output(2, fmt.Sprintln(v...))
	}
}

// randStr returns a pseudo-random string of n upper case ASCII letters.
func randStr(n int) string {
	s := make([]byte, n)
	if gotest {
		for i := range s {
			s[i] = 'A'
		}
	} else {
		for i := range s {
			s[i] = 'A' + byte(prng.Intn(26))
		}
	}
	return string(s)
}

// tagGen is used to create unique command tags.
type tagGen struct {
	id  []byte
	seq uint64
}

// newTagGen returns a new tagGen instance with a random tag id consisting of n
// unique upper case ASCII letters.
func newTagGen(n int) *tagGen {
	if gotest {
		n = 1
	} else if n < 1 || 26 < n {
		n = 5
	}
	id := make([]byte, n, n+20)
	if gotest {
		id[0] = 'A'
	} else {
		for i, v := range prng.Perm(26)[:n] {
			id[i] = 'A' + byte(v)
		}
	}
	return &tagGen{id, 0}
}

// Next returns the tag that should be used by the next command.
func (t *tagGen) Next() string {
	t.seq++
	return string(strconv.AppendUint(t.id, t.seq, 10))
}

// defaultPort joins addr and port if addr contains just the host name or IP.
func defaultPort(addr, port string) string {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		addr = net.JoinHostPort(addr, port)
	}
	return addr
}

// setServerName returns a new TLS configuration with ServerName set to host if
// the original configuration was nil or config.ServerName was empty.
func setServerName(config *tls.Config, host string) *tls.Config {
	if config == nil {
		config = &tls.Config{ServerName: host}
	} else if config.ServerName == "" {
		c := *config
		c.ServerName = host
		config = &c
	}
	return config
}

var b64codec = base64.StdEncoding

// b64enc encodes src to Base64 representation, returning the result as a new
// byte slice.
func b64enc(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, b64codec.EncodedLen(len(src)))
	b64codec.Encode(dst, src)
	return dst
}

// b64dec decodes src from Base64 representation, returning the result as a new
// byte slice.
func b64dec(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	dst := make([]byte, b64codec.DecodedLen(len(src)))
	n, err := b64codec.Decode(dst, src)
	return dst[:n], err
}

// prngSource is a goroutine-safe implementation of rand.Source.
type prngSource struct {
	mu  sync.Mutex
	src rand.Source
}

func (r *prngSource) Int63() (n int64) {
	r.mu.Lock()
	n = r.src.Int63()
	r.mu.Unlock()
	return
}

func (r *prngSource) Seed(seed int64) {
	r.mu.Lock()
	r.src.Seed(seed)
	r.mu.Unlock()
}
