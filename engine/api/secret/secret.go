package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	vault "github.com/hashicorp/vault/api"
)

// AES key fetched
const (
	nonceSize = aes.BlockSize
	macSize   = 32
	ckeySize  = 32
)

var (
	key    []byte
	prefix = "3DICC3It"
)

type Secret struct {
	Token  string
	Client *vault.Client
}

// Init secrets: cipherKey
// cipherKey is set from viper configuration
func Init(cipherKey string) {
	key = []byte(cipherKey)
}

// Create new secret client
func New(token, addr string) (*Secret, error) {
	client, err := vault.NewClient(vault.DefaultConfig())
	if err != nil {
		return nil, err
	}

	client.SetToken(token)
	client.SetAddress(addr)
	return &Secret{
		Client: client,
		Token:  token,
	}, nil
}

// Get secret from vault
func (secret *Secret) GetFromVault(s string) (string, error) {
	conf, err := secret.Client.Logical().Read(s)
	if err != nil {
		return "", err
	} else if conf == nil {
		log.Warning("vault> no value found at %q", s)
		return "", nil
	}

	value, exists := conf.Data["data"]
	if !exists {
		log.Warning("vault> no 'data' field found for %q (you must add a field with a key named data)", s)
		return "", nil
	}

	return fmt.Sprintf("%v", value), nil
}

// Encrypt data using aes+hmac algorithm
// Init() must be called before any encryption
func Encrypt(data []byte) ([]byte, error) {
	// Check key is ready
	if key == nil {
		log.Error("Missing key, init failed?")
		return nil, sdk.ErrSecretKeyFetchFailed
	}
	// generate nonce
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	// init aes cipher
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ctr := cipher.NewCTR(c, nonce)
	// encrypt data
	ct := make([]byte, len(data))
	ctr.XORKeyStream(ct, data)
	// add hmac
	h := hmac.New(sha256.New, key[ckeySize:])
	ct = append(nonce, ct...)
	h.Write(ct)
	ct = h.Sum(ct)

	return append([]byte(prefix), ct...), nil
}

// Decrypt data using aes+hmac algorithm
// Init() must be called before any decryption
func Decrypt(data []byte) ([]byte, error) {

	if !strings.HasPrefix(string(data), prefix) {
		return data, nil
	}
	data = []byte(strings.TrimPrefix(string(data), prefix))

	if key == nil {
		log.Error("Missing key, init failed?")
		return nil, sdk.ErrSecretKeyFetchFailed
	}

	if len(data) < (nonceSize + macSize) {
		log.Error("cannot decrypt secret, got invalid data")
		return nil, sdk.ErrInvalidSecretFormat
	}

	// Split actual data, hmac and nonce
	macStart := len(data) - macSize
	tag := data[macStart:]
	out := make([]byte, macStart-nonceSize)
	data = data[:macStart]
	// check hmac
	h := hmac.New(sha256.New, key[ckeySize:])
	h.Write(data)
	mac := h.Sum(nil)
	if !hmac.Equal(mac, tag) {
		return nil, fmt.Errorf("invalid hmac")
	}
	// uncipher data
	c, err := aes.NewCipher(key[:ckeySize])
	if err != nil {
		return nil, err
	}
	ctr := cipher.NewCTR(c, data[:nonceSize])
	ctr.XORKeyStream(out, data[nonceSize:])
	return out, nil
}

//DecryptVariable decrypts variable value using aes+hmac algorithm
func DecryptVariable(v *sdk.Variable) error {
	if !sdk.NeedPlaceholder(v.Type) {
		return nil
	}

	// Empty
	if len(v.Value) == (nonceSize + macSize) {
		return nil
	}

	d, err := Decrypt([]byte(v.Value))
	if err != nil {
		return err
	}

	v.Value = string(d)
	return nil
}

// DecryptS wrap Decrypt and:
// - return Placeholder instead of value if not needed
// - cast returned value in string
func DecryptS(ptype string, val sql.NullString, data []byte, clear bool) (string, error) {
	// If not a password, return value
	if !sdk.NeedPlaceholder(ptype) && val.Valid {
		return val.String, nil
	}

	// Empty
	if len(data) == (nonceSize + macSize) {
		return "", nil
	}

	// If we don't want a clear password value, return placeholder
	if !clear {
		return sdk.PasswordPlaceholder, nil
	}

	if val.Valid {
		return val.String, nil
	}

	d, err := Decrypt(data)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

// EncryptS wrap Encrypt and:
// - return valid string if type is not a password
// - cipher and returned ciphered value in a []byte if password
func EncryptS(ptype string, value string) (sql.NullString, []byte, error) {
	var n sql.NullString

	if !sdk.NeedPlaceholder(ptype) {
		n.String = value
		n.Valid = true
		return n, nil, nil
	}

	// Check their is no bug and data is not a password placholder
	if value == sdk.PasswordPlaceholder {
		log.Error("secret.Encrypt> Don't encrypt PasswordPlaceholder !\n")
		return n, nil, sdk.ErrInvalidSecretValue
	}

	d, err := Encrypt([]byte(value))
	return n, d, err
}
