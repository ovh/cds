package gpg

import (
	"bytes"
	"errors"
	"golang.org/x/crypto/openpgp"
	"io"
)

func Decode(key, passphrase []byte, src io.Reader, dest io.Writer) error {
	dec := &Decoder{key, passphrase}
	return dec.Decode(src, dest)
}

type Decoder struct {
	Key        []byte
	Passphrase []byte
}

func (d *Decoder) Decode(r io.Reader, w io.Writer) error {
	entitylist, err := openpgp.ReadArmoredKeyRing(bytes.NewBuffer(d.Key))
	if err != nil {
		return err
	}
	entity := entitylist[0]

	if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
		if len(d.Passphrase) == 0 {
			return errors.New("Private key is encrypted but you did not provide a passphrase")
		}
		err := entity.PrivateKey.Decrypt(d.Passphrase)
		if err != nil {
			return errors.New("Failed to decrypt private key. Did you use the wrong passphrase? (" + err.Error() + ")")
		}
	}
	for _, subkey := range entity.Subkeys {
		if subkey.PrivateKey != nil && subkey.PrivateKey.Encrypted {
			err := subkey.PrivateKey.Decrypt(d.Passphrase)
			if err != nil {
				return errors.New("Failed to decrypt subkey. Did you use the wrong passphrase? (" + err.Error() + ")")
			}
		}
	}

	read, err := openpgp.ReadMessage(r, entitylist, nil, nil)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, read.LiteralData.Body)
	return err

}
