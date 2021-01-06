package encryption

import (
	"io"
	"sync"

	"github.com/ovh/symmecrypt/convergent"
	"github.com/ovh/symmecrypt/keyloader"
	"github.com/ovh/symmecrypt/stream"

	"github.com/ovh/cds/sdk"
)

func New(config []convergent.ConvergentEncryptionConfig) ConvergentEncryption {
	if len(config) == 0 {
		return &noEncryption{}
	}
	return &convergentEncryption{config: config}
}

func NewNoConvergentEncryption(config []*keyloader.KeyConfig) NoConvergentEncryption {
	if len(config) == 0 {
		return &noEncryption{}
	}
	return &noConvergentEncryption{config: config}
}

type ConvergentEncryption interface {
	NewLocator(h string) (string, error)
	Write(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
	Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
}

type NoConvergentEncryption interface {
	NewLocator(h string) (string, error)
	Write(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
	Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
}

type noEncryption struct{}

type noConvergentEncryption struct {
	keys   map[string]stream.Key
	config []*keyloader.KeyConfig
	mutex  sync.Mutex
}

func (s *noConvergentEncryption) NewLocator(h string) (string, error) {
	return h, nil
}

func (s *noConvergentEncryption) Write(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	k, err := s.getKey(i.Item.Hash)
	if err != nil {
		return err
	}
	err = k.EncryptPipe(r, w)
	return sdk.WrapError(err, "[%T] unable to write item %s/%s: %+v", s, i.ID, i.ItemID, i.Item.APIRef)
}

func (s *noConvergentEncryption) getKey(h string) (stream.Key, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.keys == nil {
		s.keys = make(map[string]stream.Key)
	}
	k, has := s.keys[h]
	if !has {
		var err error
		symk, err := keyloader.NewKey(s.config...)
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		k = stream.NewKey(symk)
		s.keys[h] = k
	}
	return k, nil
}

func (s *noConvergentEncryption) Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	k, err := s.getKey(i.Item.Hash)
	if err != nil {
		return err
	}
	err = k.DecryptPipe(r, w)
	return sdk.WrapError(err, "[%T] unable to read item %s/%s: %+v", s, i.ID, i.ItemID, i.Item.APIRef)
}

func (s *noEncryption) NewLocator(h string) (string, error) {
	return h, nil
}

func (s *noEncryption) Write(_ sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return sdk.WithStack(err)
}

func (*noEncryption) Read(_ sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return sdk.WithStack(err)
}

type convergentEncryption struct {
	keys   map[string]convergent.Key
	config []convergent.ConvergentEncryptionConfig
	mutex  sync.Mutex
}

func (s *convergentEncryption) getKey(h string) (convergent.Key, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

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

func (s *convergentEncryption) Write(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	k, err := s.getKey(i.Item.Hash)
	if err != nil {
		return err
	}
	err = k.EncryptPipe(r, w)
	return sdk.WrapError(err, "[%T] unable to write item %s/%s: %+v", s, i.ID, i.ItemID, i.Item.APIRef)
}

func (s *convergentEncryption) Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	k, err := s.getKey(i.Item.Hash)
	if err != nil {
		return err
	}
	err = k.DecryptPipe(r, w)
	return sdk.WrapError(err, "[%T] unable to read item %s/%s: %+v", s, i.ID, i.ItemID, i.Item.APIRef)
}
