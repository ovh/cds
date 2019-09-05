package hmac

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"

	"github.com/ovh/symmecrypt"
	"github.com/ovh/symmecrypt/symutils"
)

const (
	// CipherName is the name of the cipher as registered on symmecrypt
	CipherName = "hmac"

	// KeyLen is the number of raw bytes for a key
	KeyLen      = 32
	dataLenSize = 8
	rawTagSize  = sha512.Size
)

var (
	tagSize = base64.URLEncoding.EncodedLen(sha512.Size)
)

func init() {
	symmecrypt.RegisterCipher(CipherName, hmacFactory{})
}

type hmacFactory struct{}

func (f hmacFactory) NewKey(s string) (symmecrypt.Key, error) {
	k, err := symutils.RawKey([]byte(s), KeyLen)
	if err != nil {
		return nil, err
	}
	return Key(k), nil
}

func (f hmacFactory) NewRandomKey() (symmecrypt.Key, error) {
	b, err := symutils.Random(KeyLen)
	if err != nil {
		return nil, err
	}
	return Key(b), nil
}

// Key is a simple key which uses plain data + HMAC-sha512 for authentication
type Key []byte

func tag(h hash.Hash, data, s []byte) ([]byte, error) {
	al := make([]byte, dataLenSize)
	binary.BigEndian.PutUint64(al, uint64(len(data)*8)) // in bits
	_, err := h.Write(data)
	if err != nil {
		return nil, err
	}
	_, err = h.Write(s)
	if err != nil {
		return nil, err
	}
	_, err = h.Write(al)
	if err != nil {
		return nil, err
	}
	sum := h.Sum(nil)[:rawTagSize]
	ret := make([]byte, tagSize)
	base64.URLEncoding.Encode(ret, sum)
	return ret, nil
}

// Encrypt appends a base64 HMAC-sha512 (with fully printable characters) calculated from the plaintext + extra data, to the plaintext.
func (k Key) Encrypt(plain []byte, extra ...[]byte) ([]byte, error) {

	var extraData []byte
	for _, e := range extra {
		extraData = append(extraData, e...)
	}

	t, err := tag(hmac.New(sha512.New, []byte(k)), extraData, plain)
	if err != nil {
		return nil, err
	}

	return append(plain, t...), nil
}

// Decrypt checks the appended base64 HMAC-sha512 using the plaintext part + extra data, and returns the plaintext only.
func (k Key) Decrypt(data []byte, extra ...[]byte) ([]byte, error) {

	if len(data) < tagSize {
		return nil, errors.New("ciphertext too short")
	}

	plain := data[:len(data)-tagSize]
	t := data[len(data)-tagSize:]

	var extraData []byte
	for _, e := range extra {
		extraData = append(extraData, e...)
	}

	t2, err := tag(hmac.New(sha512.New, []byte(k)), extraData, plain)
	if err != nil {
		return nil, err
	}

	if !hmac.Equal(t, t2) {
		return nil, errors.New("message authentication failed")
	}

	return plain, nil
}

// EncryptMarshal appends a HMAC-sha512 to the JSON representation of the object, and returns an opaque base64 encoded string.
func (k Key) EncryptMarshal(v interface{}, extra ...[]byte) (string, error) {
	serialized, err := json.Marshal(v)
	if err != nil {
		return "", nil
	}
	ciphered, err := k.Encrypt(serialized, extra...)
	if err != nil {
		return "", nil
	}
	return base64.URLEncoding.EncodeToString(ciphered), nil
}

// DecryptMarshal takes as parameter an opaque base64 encoded string, checks the HMAC-sha512 appended to the JSON representation, and unmarshals it into the target.
func (k Key) DecryptMarshal(s string, v interface{}, extra ...[]byte) error {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	unciphered, err := k.Decrypt(data, extra...)
	if err != nil {
		return err
	}
	return json.Unmarshal(unciphered, v)
}

// Wait is a no-op
func (k Key) Wait() {
	// no-op
}

// String returns an hex encoded representation of the key
func (k Key) String() (string, error) {
	return hex.EncodeToString([]byte(k)), nil
}
