package readfile

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/mattn/go-zglob"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

// Name for test readfile
const Name = "readfile"

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// Result represents a step result
type Result struct {
	Executor    Executor          `json:"executor,omitempty" yaml:"executor,omitempty"`
	Content     string            `json:"content,omitempty" yaml:"content,omitempty"`
	ContentJSON interface{}       `json:"contentjson,omitempty" yaml:"contentjson,omitempty"`
	Err         string            `json:"error" yaml:"error"`
	TimeSeconds float64           `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
	TimeHuman   string            `json:"timeHuman,omitempty" yaml:"timeHuman,omitempty"`
	Md5sum      map[string]string `json:"md5sum,omitempty" yaml:"md5sum,omitempty"`
	Size        map[string]int64  `json:"size,omitempty" yaml:"size,omitempty"`
	ModTime     map[string]int64  `json:"modtime,omitempty" yaml:"modtime,omitempty"`
	Mod         map[string]string `json:"mod,omitempty" yaml:"mod,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldNotExist"}}
}

// Run execute TestStep of type exec
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep) (venom.ExecutorResult, error) {

	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	if e.Path == "" {
		return nil, fmt.Errorf("Invalid path")
	}

	start := time.Now()

	result, errr := e.readfile(e.Path)
	if errr != nil {
		result.Err = errr.Error()
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = fmt.Sprintf("%s", elapsed)

	return executors.Dump(result)
}

func (e *Executor) readfile(path string) (Result, error) {

	result := Result{Executor: *e}

	fileInfo, _ := os.Stat(path)
	if fileInfo != nil && fileInfo.IsDir() {
		path = filepath.Dir(path)
	}

	filesPath, errg := zglob.Glob(path)
	if errg != nil {
		return result, fmt.Errorf("Error reading files on path:%s :%s", path, errg)
	}

	if len(filesPath) == 0 {
		return result, fmt.Errorf("Invalid path '%s' or file not found", path)
	}

	var content string
	md5sum := make(map[string]string)
	size := make(map[string]int64)
	modtime := make(map[string]int64)
	mod := make(map[string]string)

	for _, f := range filesPath {
		f, erro := os.Open(f)
		if erro != nil {
			return result, fmt.Errorf("Error while opening file: %s", erro)
		}
		defer f.Close()

		b, errr := ioutil.ReadAll(f)
		if errr != nil {
			return result, fmt.Errorf("Error while reading file: %s", errr)
		}
		content += string(b)

		h := md5.New()
		if _, err := io.Copy(h, f); err != nil {
			return result, fmt.Errorf("Error while compute md5sum: %s", err)
		}

		md5sum[f.Name()] = hex.EncodeToString(h.Sum(nil))

		stat, errs := f.Stat()
		if errs != nil {
			return result, fmt.Errorf("Error while compute file size: %s", errs)
		}

		size[f.Name()] = stat.Size()
		modtime[f.Name()] = stat.ModTime().Unix()
		mod[f.Name()] = stat.Mode().String()

	}

	result.Content = content

	bodyJSONArray := []interface{}{}
	if err := json.Unmarshal([]byte(content), &bodyJSONArray); err != nil {
		bodyJSONMap := map[string]interface{}{}
		if err2 := json.Unmarshal([]byte(content), &bodyJSONMap); err2 == nil {
			result.ContentJSON = bodyJSONMap
		}
	} else {
		result.ContentJSON = bodyJSONArray
	}
	result.Md5sum = md5sum
	result.Size = size
	result.ModTime = modtime
	result.Mod = mod

	return result, nil
}
