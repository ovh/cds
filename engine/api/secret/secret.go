package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/secret/filesecretbackend"
	"github.com/ovh/cds/engine/api/secret/secretbackend"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// AES key fetched
const (
	nonceSize = aes.BlockSize
	macSize   = 32
	ckeySize  = 32
)

var (
	key                            []byte
	prefix                         = "3DICC3It"
	defaultKey                     = []byte("78eKVxCGLm6gwoH9LAQ15ZD5AOABo1Xf")
	testingPrefix                  = "3IFCC4Ib"
	SecretUsername, SecretPassword string
	//Client is a shared instance
	Client secretbackend.Driver
)

type databaseInstance struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

type databaseCredentials struct {
	Readers  []databaseInstance `json:"readers"`
	Writers  []databaseInstance `json:"writers"`
	Database string             `json:"database"`
	Password string             `json:"password"`
	User     string             `json:"user"`
	Type     string             `json:"type"`
}

// Init password manager
// if secretBackendBinary is empty, use default AES key and default file secret backend
func Init(secretBackendBinary string, opts map[string]string) error {
	//Initializing secret backend
	var err error
	if secretBackendBinary == "" {
		//Default is embedded file secretbackend
		log.Warning("Using default AES key")
		key = defaultKey
		prefix = testingPrefix
		log.Warning("Using default file secret backend")
		Client = filesecretbackend.Client(opts)
	} else {
		//Load the secretbackend plugin
		log.Notice("Loading Secret Backend Plugin %s", secretBackendBinary)
		client := secretbackend.NewClient(secretBackendBinary, opts)
		Client, err = client.Instance()
		if err != nil {
			return err
		}
	}

	secrets := Client.GetSecrets()
	if secrets.Err() != nil {
		log.Critical("Error: %v", secrets.Err())
		return secrets.Err()
	}

	//If key hasn't been initilized with default key
	if len(key) == 0 {
		aesKey, _ := secrets.Get("cds/aes-key")
		if aesKey == "" {
			log.Critical("secret.Init> cds/aes-key not found\n")
			return sdk.ErrSecretKeyFetchFailed
		}
		key = []byte(aesKey)

	}

	dbKey := viper.GetString("DB_SECRET")
	if dbKey == "" {
		return nil
	}
	cdsDBCredS, _ := secrets.Get(dbKey)
	if cdsDBCredS == "" {
		log.Critical("secret.Init> %s not found", dbKey)
		return nil
	}

	var cdsDBCred = databaseCredentials{}
	if err := json.Unmarshal([]byte(cdsDBCredS), &cdsDBCred); err != nil {
		log.Critical("secret.Init> Unable to unmarshal secret %s", err)
		return nil
	}

	log.Notice("secret.Init> Database credentials found")
	SecretUsername = cdsDBCred.User
	SecretPassword = cdsDBCred.Password

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
