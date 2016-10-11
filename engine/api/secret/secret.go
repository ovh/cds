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

	"github.com/ovh/cds/engine/api/vault"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// AES key fetched from Vault
var key []byte
var prefix string

const (
	nonceSize = aes.BlockSize
	macSize   = 32
	ckeySize  = 32
)

var (
	testingKey    = []byte("78eKVxCGLm6gwoH9LAQ15ZD5AOABo1Xf")
	testingPrefix = "3IFCC4Ib"
)

func init() {
	prefix = "3DICC3It"
}

// Init password manager
// If vaultKey is empty, use default testing key
// otherwise, fetch AES key from vault
func Init(appKey, vaultHostname, vaultTOTP, vaultTokenHeader string) error {

	if vaultHostname == "local-insecure" {
		log.Warning("Using default AES key")
		key = testingKey
		prefix = testingPrefix
		return nil
	}

	// Fetch key from vault
	vaultClient, err := vault.GetClient(vaultHostname, appKey, vaultTOTP, "", "", vaultTokenHeader)
	if err != nil {
		log.Warning("secret.Init> Unable to get a Vault client %s\n", err)
		return err
	}

	secrets, err := vaultClient.GetSecrets()
	if err != nil {
		log.Warning("secret.Init> Unable to fetch Vault secrets %s\n", err)
		return sdk.ErrSecretKeyFetchFailed
	}

	aesKey, ok := secrets["cds/aes-key"]
	if !ok {
		log.Critical("secret.Init> cds/aes-key not found\n")
		return sdk.ErrSecretKeyFetchFailed
	}

	key = []byte(aesKey)
	return nil
}

// Encrypt data using aes+hmac algorithm
// Init() must be called before any encryption
func Encrypt(data []byte) ([]byte, error) {
	// Check key is ready
	if key == nil {
		log.Critical("Missing key, init failed?")
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
		log.Critical("Missing key, init failed?")
		return nil, sdk.ErrSecretKeyFetchFailed
	}

	if len(data) < (nonceSize + macSize) {
		log.Critical("cannot decrypt secret, got invalid data")
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

// DecryptS wrap Decrypt and:
// - return Placeholder instead of value if not needed
// - cast returned value in string
func DecryptS(ptype sdk.VariableType, val sql.NullString, data []byte, clear bool) (string, error) {
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
func EncryptS(ptype sdk.VariableType, value string) (sql.NullString, []byte, error) {
	var n sql.NullString

	if !sdk.NeedPlaceholder(ptype) {
		n.String = value
		n.Valid = true
		return n, nil, nil
	}

	// Check their is no bug and data is not a password placholder
	if value == sdk.PasswordPlaceholder {
		log.Critical("secret.Encrypt> Don't encrypt PasswordPlaceholder !\n")
		return n, nil, sdk.ErrInvalidSecretValue
	}

	d, err := Encrypt([]byte(value))
	return n, d, err
}
