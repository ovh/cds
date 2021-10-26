package gpg

import (
	"bytes"
	"fmt"
	"io"

	go_ed25519 "golang.org/x/crypto/ed25519"

	"github.com/keybase/go-crypto/ed25519"
	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/armor"
	"github.com/keybase/go-crypto/openpgp/packet"
	"github.com/pkg/errors"
)

type PrivateKey struct {
	*openpgp.Entity
}

func NewPrivateKeyFromPem(keyPem string, passphrase string) (*PrivateKey, error) {
	p, err := armor.Decode(bytes.NewBufferString(keyPem))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unarmor GPG key")
	}

	if p.Type != openpgp.PrivateKeyType {
		return nil, fmt.Errorf("Invalid key type, got: %q, wanted: %q", p.Type, openpgp.PrivateKeyType)
	}

	return NewPrivateKeyFromData(p.Body, passphrase)
}

func NewPrivateKeyFromData(data io.Reader, passphrase string) (*PrivateKey, error) {
	e, err := openpgp.ReadEntity(packet.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse GPG key")
	}

	if err := e.PrivateKey.Decrypt([]byte(passphrase)); err != nil {
		return nil, err
	}
	for _, subkey := range e.Subkeys {
		if err := subkey.PrivateKey.Decrypt([]byte(passphrase)); err != nil {
			return nil, err
		}
	}

	return &PrivateKey{Entity: e}, nil
}

func (k PrivateKey) GenerateSignature(data string) ([]byte, error) {
	var out bytes.Buffer

	err := openpgp.DetachSign(
		&out,
		k.Entity,
		bytes.NewBufferString(data),
		nil,
	)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate signature")
	}

	return out.Bytes(), nil
}

func (k PrivateKey) VerifySignature(data string, rawSig []byte) error {
	return PublicKey(k).VerifySignature(data, rawSig)
}

func (k PrivateKey) DecryptData(data []byte) ([]byte, error) {
	message, err := openpgp.ReadMessage(
		bytes.NewBuffer(data),
		openpgp.EntityList{
			k.Entity,
		},
		nil,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read GPG message")
	}

	out, err := io.ReadAll(message.UnverifiedBody)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read decrypted message")
	}

	return out, nil
}

func (k PrivateKey) EncryptData(data []byte) ([]byte, error) {
	return PublicKey(k).EncryptData(data)
}

func (k PrivateKey) ReadSignedMessage(msg []byte) ([]byte, *packet.Signature, error) {
	return PublicKey(k).ReadSignedMessage(msg)
}

type BufferNopCloser struct {
	bytes.Buffer
}

func (b BufferNopCloser) Close() error {
	return nil
}

func (k PrivateKey) SignMessage(msg []byte) ([]byte, error) {
	var out BufferNopCloser

	signer, err := openpgp.AttachedSign(
		&out,
		*k.Entity,
		&openpgp.FileHints{
			IsBinary: true,
		},
		&packet.Config{
			DefaultCompressionAlgo: packet.CompressionZLIB,
			CompressionConfig:      &packet.CompressionConfig{Level: 9},
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to start signing")
	}

	_, err = signer.Write(msg)
	if err != nil {
		return nil, errors.Wrap(err, "Failed write data to sign")
	}

	err = signer.Close()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to close signer")
	}

	return out.Bytes(), nil
}

func (k PrivateKey) KeyId() string {
	return PublicKey(k).KeyId()
}

func (k PrivateKey) GetKey() interface{} {
	pk := k.PrivateKey.PrivateKey
	switch v := pk.(type) {
	case *packet.EdDSAPrivateKey:
		return go_ed25519.NewKeyFromSeed(v.Seed())
	}
	return fmt.Errorf("unsupported key type/format: %T", pk)
}

func (k PrivateKey) GetPubKey() interface{} {
	return PublicKey(k).GetKey()
}

// IsSignedBy check if k is signed by signer key. Assume it is the case
// if at least one identity is signed by the signer key
func (k PrivateKey) IsSignedBy(signerKey *PublicKey) *packet.Signature {
	return PublicKey(k).IsSignedBy(signerKey)
}

func (k PrivateKey) Serialize() ([]byte, error) {
	return PublicKey(k).Serialize()
}

type PublicKey struct {
	*openpgp.Entity
}

func NewPublicKeyFromPem(keyPem string) (*PublicKey, error) {
	p, err := armor.Decode(bytes.NewBufferString(keyPem))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unarmor GPG key")
	}

	if p.Type != openpgp.PublicKeyType {
		return nil, fmt.Errorf("Invalid key type, got: %q, wanted: %q", p.Type, openpgp.PublicKeyType)
	}

	return NewPublicKeyFromData(p.Body)
}

func NewPublicKeyFromData(data io.Reader) (*PublicKey, error) {
	e, err := openpgp.ReadEntity(packet.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse GPG key")
	}

	return &PublicKey{Entity: e}, nil
}

func (k PublicKey) EncryptData(data []byte) ([]byte, error) {
	var out bytes.Buffer

	encrypter, err := openpgp.Encrypt(
		&out,
		openpgp.EntityList{
			k.Entity,
		},
		nil,
		nil,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to start encryption")
	}

	_, err = encrypter.Write(data)
	if err != nil {
		return nil, errors.Wrap(err, "Failed write data to encrypt")
	}

	err = encrypter.Close()
	if err != nil {
		return nil, errors.Wrap(err, "Failed close encrypter")
	}

	return out.Bytes(), nil
}

func (k PublicKey) VerifySignature(data string, rawSig []byte) error {
	_, err := openpgp.CheckDetachedSignature(
		openpgp.EntityList{
			k.Entity,
		},
		bytes.NewBufferString(data),
		bytes.NewReader(rawSig),
	)

	return err
}

// IsSignedBy check if k is signed by signer key. Assume it is the case
// if at least one identity is signed by the signer key
func (k PublicKey) IsSignedBy(signerKey *PublicKey) *packet.Signature {
	for _, i := range k.Identities {
		for _, s := range i.Signatures {
			// Skip signature if IssuerKeyId doesn't match signer's key
			if s.IssuerKeyId != nil && *s.IssuerKeyId != signerKey.PrimaryKey.KeyId {
				continue
			}
			// Return true if signature s is valid
			if err := signerKey.PrimaryKey.VerifyUserIdSignature(i.Name, k.PrimaryKey, s); err == nil {
				return s
			}
		}
	}
	return nil
}

func (k PublicKey) ReadSignedMessage(msg []byte) ([]byte, *packet.Signature, error) {
	md, err := openpgp.ReadMessage(
		bytes.NewReader(msg),
		openpgp.EntityList{
			k.Entity,
		},
		nil,
		nil,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to read PGP message")
	}

	if !md.IsSigned {
		return nil, nil, errors.New("Message is not signed")
	}

	if md.SignedByKeyId != k.Entity.PrimaryKey.KeyId {
		return nil, nil, errors.New("Message is signed with another key")
	}

	data, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to read body")
	}

	// We can now check SignatureError as we read the whole UnverifiedBody
	if md.SignatureError != nil {
		return nil, nil, errors.Wrap(err, "Failed to verify signature")
	}

	return data, md.Signature, nil
}

func (k PublicKey) KeyId() string {
	return fmt.Sprintf("%X", k.Entity.PrimaryKey.Fingerprint)
}

func (k PublicKey) Serialize() ([]byte, error) {
	var out bytes.Buffer
	err := k.Entity.Serialize(&out)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to serialize key")
	}
	return out.Bytes(), nil
}

func (k PublicKey) GetKey() interface{} {
	pk := k.PrimaryKey.PublicKey
	switch v := pk.(type) {
	case ed25519.PublicKey:
		return go_ed25519.PublicKey([]byte(v))
	}
	return fmt.Errorf("unsupported key type/format: %T", pk)
}
