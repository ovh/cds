package keys

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ovh/cds/sdk"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

//GetOpenPGPEntity returns a single entity from an armored entity list
func GetOpenPGPEntity(r io.Reader) (*openpgp.Entity, error) {
	entityList, err := openpgp.ReadArmoredKeyRing(r)
	if err != nil {
		return nil, sdk.WrapError(err, "GetOpenPGPEntity> Unable to read armored key ring")
	}

	if len(entityList) != 1 {
		return nil, errors.New("GetOpenPGPEntity> Invalid PGP entity list")
	}

	keys := entityList.DecryptionKeys()
	if len(keys) != 1 {
		return nil, errors.New("GetOpenPGPEntity> Invalid PGP decryption keys")
	}

	return entityList[0], nil
}

//NewOpenPGPEntity create an openpgp Entity
func NewOpenPGPEntity(keyname string) (*openpgp.Entity, error) {
	key, errE := openpgp.NewEntity(keyname, keyname, "cds@locahost", nil)
	if errE != nil {
		return nil, sdk.WrapError(errE, "NewOpenPGPEntity> Cannot create new entity")
	}

	if len(key.Subkeys) != 1 {
		return nil, fmt.Errorf("Wrong key generation")
	}

	// Self sign Identity
	for _, id := range key.Identities {
		id.SelfSignature.PreferredSymmetric = []uint8{
			uint8(packet.CipherAES256),
			uint8(packet.Cipher3DES),
		}
		id.SelfSignature.PreferredHash = []uint8{
			sha512,
		}
		if err := id.SelfSignature.SignUserId(id.UserId.Id, key.PrimaryKey, key.PrivateKey, nil); err != nil {
			return nil, sdk.WrapError(err, "NewOpenPGPEntity> Cannot sign identity")
		}
	}

	return key, nil
}

// generatePGPPrivateKey generates a private key
func generatePGPPrivateKey(key *openpgp.Entity) (io.Reader, error) {
	bufPrivate := new(bytes.Buffer)
	w, errPrivEncode := armor.Encode(bufPrivate, openpgp.PrivateKeyType, nil)
	if errPrivEncode != nil {
		return nil, sdk.WrapError(errPrivEncode, "generatePGPPrivateKey> Cannot encode private key")
	}
	defer w.Close()
	if err := key.SerializePrivate(w, &packet.Config{}); err != nil {
		return nil, sdk.WrapError(err, "generatePGPPrivateKey> Cannot serialize private key")
	}
	return bufPrivate, nil
}

// generatePGPPublicKey generates a public key
func generatePGPPublicKey(key *openpgp.Entity) (io.Reader, error) {
	bufPublic := new(bytes.Buffer)
	w, errEncode := armor.Encode(bufPublic, openpgp.PublicKeyType, nil)
	if errEncode != nil {
		return nil, sdk.WrapError(errEncode, "generatePGPPublicKey> Cannot encode public key")
	}
	defer w.Close()
	if err := key.Serialize(w); err != nil {
		return nil, sdk.WrapError(err, "generatePGPPublicKey> Cannot serialize public key")
	}
	return bufPublic, nil
}

// GeneratePGPKeyPair generates a private / public PGP key
func GeneratePGPKeyPair(name string) (sdk.Key, error) {
	k := sdk.Key{
		Name: name,
		Type: sdk.KeyTypePGP,
	}
	key, err := NewOpenPGPEntity(name)
	if err != nil {
		return k, err
	}
	k.KeyID = key.PrimaryKey.KeyIdShortString()

	bufPrivate, err := generatePGPPrivateKey(key)
	if err != nil {
		return k, err
	}

	bufPublic, err := generatePGPPublicKey(key)
	if err != nil {
		return k, err
	}

	pub, errPub := ioutil.ReadAll(bufPublic)
	if errPub != nil {
		return k, sdk.WrapError(errPub, "GeneratePGPKeyPair> Unable to read public key")
	}

	priv, errPriv := ioutil.ReadAll(bufPrivate)
	if errPriv != nil {
		return k, sdk.WrapError(errPriv, "GeneratePGPKeyPair>  Unable to read private key")
	}
	k.Private = string(priv)
	k.Public = string(pub)
	return k, err
}
