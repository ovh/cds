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
	"runtime"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
)

// EncryptFunc is a common type
type EncryptFunc func(gorp.SqlExecutor, int64, string, string) (string, error)

// IDName is generally used when you want to get basic informations from db
type IDName struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
	Icon        string `json:"icon,omitempty" db:"icon"`
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
func FileMd5sum(filePath string) (string, error) {
	file, errop := os.Open(filePath)
	if errop != nil {
		return "", fmt.Errorf("unable to copy file content to md5: %v", errop)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	hash := md5.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", fmt.Errorf("error computing md5: %v", err)
	}

	hashInBytes := hash.Sum(nil)[:16]
	sum := hex.EncodeToString(hashInBytes)

	return sum, nil
}

// FileSHA512sum returns the sha512sum of a file
func FileSHA512sum(filePath string) (string, error) {
	file, errop := os.Open(filePath)
	if errop != nil {
		return "", fmt.Errorf("error opening file for computing sha512: %v", errop)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	hash := sha512.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", fmt.Errorf("error computing sha512: %v", err)
	}

	hashInBytes := hash.Sum(nil)[:64]
	sum := hex.EncodeToString(hashInBytes)
	return sum, nil
}

// GoRoutine runs the function within a goroutine with a panic recovery
func GoRoutine(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 1<<16)
				runtime.Stack(buf, true)
				log.Error("[PANIC] %s Failed", name)
				log.Error("[PANIC] %s> %s", name, string(buf))
			}
		}()
		fn()
	}()

}
