package gpg

import (
	"bytes"
	"golang.org/x/crypto/openpgp"
	"io"
)

func Encode(key []byte, src io.Reader, dest io.Writer) error {
	enc := &Encoder{key}
	return enc.Encode(src, dest)
}

type Encoder struct {
	Key []byte
}

func (e *Encoder) Encode(r io.Reader, w io.Writer) error {
	entitylist, err := openpgp.ReadArmoredKeyRing(bytes.NewBuffer(e.Key))
	if err != nil {
		return err
	}

	// Encrypt message using public key
	buf := new(bytes.Buffer)
	encrypter, err := openpgp.Encrypt(buf, entitylist, nil, nil, nil)
	if err != nil {
		return err
	}

	_, err = io.Copy(encrypter, r)
	if err != nil {
		return err
	}

	err = encrypter.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(w, buf)
	return err
}
