package encryption

import (
	"io"

	"github.com/ovh/symmecrypt/convergent"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
)

func New(config []convergent.ConvergentEncryptionConfig) ConvergentEncryption {
	if len(config) == 0 {
		return &noEncryption{}
	}
	return &convergentEncryption{config: config}
}

type ConvergentEncryption interface {
	NewLocator(h string) (string, error)
	Write(i storage.ItemUnit, r io.Reader, w io.Writer) error
	Read(i storage.ItemUnit, r io.Reader, w io.Writer) error
}

type noEncryption struct{}

func (s *noEncryption) NewLocator(h string) (string, error) {
	return h, nil
}

func (s *noEncryption) Write(i storage.ItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}

func (*noEncryption) Read(i storage.ItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}

type convergentEncryption struct {
	keys   map[string]convergent.Key
	config []convergent.ConvergentEncryptionConfig
}

func (s *convergentEncryption) getKey(h string) (convergent.Key, error) {
	if s.keys == nil {
		s.keys = make(map[string]convergent.Key)
	}
	k, has := s.keys[h]
	if !has {
		var err error
		k, err = convergent.NewKey(h, s.config...)
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		s.keys[h] = k
	}
	return k, nil
}

func (s *convergentEncryption) NewLocator(h string) (string, error) {
	k, err := s.getKey(h)
	if err != nil {
		return "", err
	}
	return k.Locator()
}

func (s *convergentEncryption) Write(i storage.ItemUnit, r io.Reader, w io.Writer) error {
	k, err := s.getKey(i.Item.Hash)
	if err != nil {
		return err
	}
	err = k.EncryptPipe(r, w, []byte(i.ID))
	return sdk.WrapError(err, "[%T] unable to write item %s", s, i.ID)
}

func (s *convergentEncryption) Read(i storage.ItemUnit, r io.Reader, w io.Writer) error {
	k, err := s.getKey(i.Item.Hash)
	if err != nil {
		return err
	}
	err = k.DecryptPipe(r, w, []byte(i.ID))
	return sdk.WrapError(err, "[%T] unable to read item %s", s, i.ItemID)
}
