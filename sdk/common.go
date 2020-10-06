package sdk

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha512"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"unicode"

	"github.com/go-gorp/gorp"
	"github.com/pkg/errors"
)

// MaxIconSize is the maximum size of the icon in octet
const MaxIconSize = 120000

// IconFormat is the format prefix accepted for icon
const IconFormat = "data:image/"

// True of false
var (
	True        = true
	False       = false
	TrueString  = "true"
	FalseString = "false"
)

// EncryptFunc is a common type
type EncryptFunc func(gorp.SqlExecutor, int64, string, string) (string, error)

// IDName is generally used when you want to get basic informations from db
type IDName struct {
	ID          int64   `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Description string  `json:"description,omitempty" db:"description"`
	Icon        string  `json:"icon,omitempty" db:"icon"`
	Labels      []Label `json:"labels,omitempty" db:"-"`
}

type IDNames []IDName

type EntitiesPermissions map[string]Permissions

func (e EntitiesPermissions) Level(s string) int {
	if e == nil {
		return 0
	}
	p, has := e[s]
	if !has {
		return 0
	}
	return p.Level()
}

func (e EntitiesPermissions) Permissions(s string) Permissions {
	if e == nil {
		return Permissions{}
	}
	p, has := e[s]
	if !has {
		return Permissions{}
	}
	return p
}

func (idNames IDNames) IDs() []int64 {
	res := make([]int64, len(idNames))
	for i := range idNames {
		res[i] = idNames[i].ID
	}
	return res
}

func (idNames IDNames) Names() []string {
	res := make([]string, len(idNames))
	for i := range idNames {
		res[i] = idNames[i].Name
	}
	return res
}

// NamePattern  Pattern for project/application/pipeline/group name
const NamePattern = "^[a-zA-Z0-9._-]{1,}$"

// NamePatternRegex  Pattern regexp
var NamePatternRegex = regexp.MustCompile(NamePattern)

// NamePatternSpace  Pattern for stage name
const NamePatternSpace = `^[\sa-zA-Z0-9._-]{1,}$`

// NamePatternSpaceRegex  Pattern regexp
var NamePatternSpaceRegex = regexp.MustCompile(NamePatternSpace)

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

func SHA512sum(s string) (string, error) {
	hash := sha512.New()
	if _, err := io.Copy(hash, strings.NewReader((s))); err != nil {
		return "", fmt.Errorf("error computing sha512: %v", err)
	}

	hashInBytes := hash.Sum(nil)[:64]
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

var rxURL = regexp.MustCompile(`http[s]?:\/\/(.*)`)

// IsURL returns if given path is a url according to the URL regex.
func IsURL(path string) bool {
	return rxURL.MatchString(path)
}

// DirectoryExists checks if the directory exists
func DirectoryExists(path string) (bool, error) {
	s, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return s.IsDir(), err
}

// StringSlice type used for database json storage.
type StringSlice []string

// Scan string slice.
func (s *StringSlice) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, s), "cannot unmarshal StringSlice")
}

// Value returns driver.Value from string slice.
func (s StringSlice) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal StringSlice")
}

func (s StringSlice) Contains(value string) bool {
	for i := range s {
		if s[i] == value {
			return true
		}
	}
	return false
}

// Int64Slice type used for database json storage.
type Int64Slice []int64

// Scan int64 slice.
func (s *Int64Slice) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, s), "cannot unmarshal Int64Slice")
}

// Value returns driver.Value from int64 slice.
func (s Int64Slice) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal Int64Slice")
}

// Contains return true if given value is in the slice.
func (s *Int64Slice) Contains(v int64) bool {
	if s == nil || len(*s) == 0 {
		return false
	}

	for _, i := range *s {
		if i == v {
			return true
		}
	}

	return false
}

// Remove all occurrences of given value from the slice.
func (s *Int64Slice) Remove(v int64) {
	filtered := make([]int64, 0, len(*s))
	for _, i := range *s {
		if i != v {
			filtered = append(filtered, i)
		}
	}
	*s = filtered
}

func StringIsAscii(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func RemoveNotPrintableChar(in string) string {
	m := func(r rune) rune {
		switch {
		case unicode.IsLetter(r),
			unicode.IsSpace(r),
			unicode.IsDigit(r),
			unicode.IsNumber(r),
			unicode.In(r, unicode.Pd, // Filter some punctuation categories
				unicode.Pe,
				unicode.Pf,
				unicode.Pi,
				unicode.Po,
				unicode.Ps,
			),
			unicode.IsSpace(r),
			unicode.IsTitle(r),
			unicode.IsUpper(r):
			return r
		default:
			return ' '
		}
	}
	return strings.Map(m, in)
}

var windowsPathRegex = regexp.MustCompile(`^[a-zA-Z]:\\[\\\S|*\S]?.*$`)

func PathIsAbs(s string) bool {
	if GOOS == "windows" || runtime.GOOS == "windows" {
		return windowsPathRegex.MatchString(s)
	}
	return path.IsAbs(s)
}
