package shredder

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"

	"github.com/maxwellhealth/go-gpg"
)

func GPGEncrypt(publicKey []byte, content io.Reader) (io.Reader, error) {
	buf := new(bytes.Buffer)
	if err := gpg.Encode(publicKey, content, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func GPGDecrypt(privateKey, passphrase []byte, content io.Reader) (io.Reader, error) {
	buf := new(bytes.Buffer)
	if err := gpg.Decode(privateKey, passphrase, content, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func AESEncrypt(key []byte, content io.Reader) (io.Reader, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	btes, err := ioutil.ReadAll(content)
	if err != nil {
		return nil, err
	}
	s := base64.StdEncoding.EncodeToString(btes)
	b := []byte(s)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return bytes.NewBuffer(ciphertext), nil
}

func AESDecrypt(key []byte, content io.Reader) (io.Reader, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	text, err := ioutil.ReadAll(content)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}
