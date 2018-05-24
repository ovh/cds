package sdk

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"

	"github.com/go-gorp/gorp"
)

// EncryptFunc is a common type
type EncryptFunc func(gorp.SqlExecutor, int64, string, string) (string, error)

// IDName is generally used when you want to get basic informations from db
type IDName struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// NamePattern  Pattern for project/application/pipeline/group name
const NamePattern = "^[a-zA-Z0-9._-]{1,}$"

// NamePatternRegex  Pattern regexp
var NamePatternRegex = regexp.MustCompile(NamePattern)

// InterfaceSlice cast a untyped slice into a slice of untypes things. It will panic if the parameter is not a slice
func InterfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("interfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

// JSONWithoutHTMLEncode return byte array of a struct into json without HTML encode
func JSONWithoutHTMLEncode(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

// FileMd5sum returns the md5sum ofr a file
func FileMd5sum(file *os.File) (string, error) {
	reader := bufio.NewReader(file)
	hash := md5.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", fmt.Errorf("FileMd5sum> error: %v", err)
	}

	hashInBytes := hash.Sum(nil)[:16]
	sum := hex.EncodeToString(hashInBytes)
	return sum, nil
}

// FileSHA512sum returns the sha512sum of a file
func FileSHA512sum(file *os.File) (string, error) {
	reader := bufio.NewReader(file)
	hash := sha512.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", fmt.Errorf("FileSHA512sum> error: %v", err)
	}

	hashInBytes := hash.Sum(nil)[:16]
	sum := hex.EncodeToString(hashInBytes)
	return sum, nil
}
