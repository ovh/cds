package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

var (
	registeredV2WorkflowRunResultDetail     map[string]V2WorkflowRunResultDetailInterface
	registeredV2WorkflowRunResultDetailLock sync.Mutex
)

func registerV2WorkflowRunResultDetail(i ...V2WorkflowRunResultDetailInterface) {
	registeredV2WorkflowRunResultDetailLock.Lock()
	defer registeredV2WorkflowRunResultDetailLock.Unlock()
	for _, x := range i {
		registeredV2WorkflowRunResultDetail[reflect.TypeOf(i).Name()] = x
	}
}

func init() {
	registerV2WorkflowRunResultDetail(
		&V2WorkflowRunResultDockerDetail{},
		&V2WorkflowRunResultReleaseDetail{},
		&V2WorkflowRunResultArsenalDeploymentDetail{},
		&V2WorkflowRunResultDebianDetail{},
		&V2WorkflowRunResultGenericDetail{},
		&V2WorkflowRunResultHelmDetail{},
		&V2WorkflowRunResultPythonDetail{},
		&V2WorkflowRunResultTerraformModuleDetail{},
		&V2WorkflowRunResultTerraformProviderDetail{},
		&V2WorkflowRunResultTestDetail{},
		&V2WorkflowRunResultVariableDetail{},
	)
}

type V2WorkflowRunResultDetailInterface interface {
	Cast(i any) error
	GetName() string
}

type V2WorkflowRunResultDetail struct {
	Data interface{} `json:"data"`
	Type string      `json:"type"`
}

func (s *V2WorkflowRunResult) CastDetail() error {
	return s.Detail.castDetail()
}

func (s *V2WorkflowRunResultDetail) castDetail() error {
	for k, v := range registeredV2WorkflowRunResultDetail {
		if k == s.Type {
			// instanciate a new V
			x := reflect.New(reflect.TypeOf(v)).Interface().(V2WorkflowRunResultDetailInterface)
			return x.Cast(s.Data)
		}
	}
	return errors.Errorf("unknow type %q", s.Type)
}

func GetConcreteDetail[T any](s *V2WorkflowRunResult) (t T, err error) {
	i, err := s.GetDetail()
	if err != nil {
		return t, err
	}

	x, ok := i.(T)
	if !ok {
		return t, errors.Errorf("unable to get concrete detail for type %q", s.Detail.Type)
	}

	return x, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *V2WorkflowRunResultDetail) UnmarshalJSON(source []byte) error {
	var content = struct {
		Data interface{}
		Type string
	}{}
	if err := JSONUnmarshal(source, &content); err != nil {
		return WrapError(err, "cannot unmarshal V2WorkflowRunResultDetail")
	}
	s.Data = content.Data
	s.Type = content.Type

	if err := s.castDetail(); err != nil {
		return err
	}

	return nil
}

// MarshalJSON implements json.Marshaler.
func (s *V2WorkflowRunResultDetail) MarshalJSON() ([]byte, error) {
	if s.Type == "" {
		s.Type = reflect.TypeOf(s.Data).Name()
	}

	var content = struct {
		Data interface{} `json:"data"`
		Type string      `json:"type"`
	}{
		Data: s.Data,
		Type: s.Type,
	}

	btes, _ := json.Marshal(content)
	return btes, nil
}

var (
	_ json.Marshaler   = new(V2WorkflowRunResultDetail)
	_ json.Unmarshaler = new(V2WorkflowRunResultDetail)
)

// Value returns driver.Value from V2WorkflowRunResultDetail
func (s V2WorkflowRunResultDetail) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal V2WorkflowRunResultDetail")
}

// Scan V2WorkflowRunResultDetail
func (s *V2WorkflowRunResultDetail) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	if err := JSONUnmarshal(source, s); err != nil {
		return WrapError(err, "cannot unmarshal V2WorkflowRunResultDetail")
	}
	return nil
}

type V2WorkflowRunResultType string

const (
	V2WorkflowRunResultTypeCoverage          = "coverage"
	V2WorkflowRunResultTypeTest              = "tests"
	V2WorkflowRunResultTypeRelease           = "release"
	V2WorkflowRunResultTypeGeneric           = "generic"
	V2WorkflowRunResultTypeVariable          = "variable"
	V2WorkflowRunResultTypeDocker            = "docker"
	V2WorkflowRunResultTypeDebian            = "debian"
	V2WorkflowRunResultTypePython            = "python"
	V2WorkflowRunResultTypeArsenalDeployment = "deployment"
	V2WorkflowRunResultTypeHelm              = "helm"
	V2WorkflowRunResultTypeTerraformProvider = "terraformProvider"
	V2WorkflowRunResultTypeTerraformModule   = "terraformModule"
	// Other values may be instantiated from Artifactory Manager repository type
)

type V2WorkflowRunResultTestDetail struct {
	Name        string           `json:"name" mapstructure:"name"`
	Size        int64            `json:"size" mapstructure:"size"`
	Mode        os.FileMode      `json:"mode" mapstructure:"mode"`
	MD5         string           `json:"md5" mapstructure:"md5"`
	SHA1        string           `json:"sha1" mapstructure:"sha1"`
	SHA256      string           `json:"sha256" mapstructure:"sha256"`
	TestsSuites JUnitTestsSuites `json:"tests_suites" mapstructure:"tests_suites"`
	TestStats   TestsStats       `json:"tests_stats" mapstructure:"tests_stats"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTestDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTestDetail) GetName() string {
	return v.Name
}

type V2WorkflowRunResultTerraformModuleDetail struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Namespace string `json:"namespace"`
	Provider  string `json:"provider"`
	Version   string `json:"version"`
	ID        string `json:"id"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformModuleDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformModuleDetail) GetName() string {
	return v.ID + "/" + v.Version
}

type V2WorkflowRunResultTerraformProviderDetail struct {
	Flavor    string `json:"flavor"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Version   string `json:"version"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformProviderDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformProviderDetail) GetName() string {
	return v.Namespace + "_" + v.Name + "_" + v.Version + "_" + v.Flavor
}

type V2WorkflowRunResultGenericDetail struct {
	Name   string      `json:"name" mapstructure:"name"`
	Size   int64       `json:"size" mapstructure:"size"`
	Mode   os.FileMode `json:"mode" mapstructure:"mode"`
	MD5    string      `json:"md5" mapstructure:"md5"`
	SHA1   string      `json:"sha1" mapstructure:"sha1"`
	SHA256 string      `json:"sha256" mapstructure:"sha256"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultGenericDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultGenericDetail) GetName() string {
	return v.Name
}

type V2WorkflowRunResultArsenalDeploymentDetail struct {
	IntegrationName string                              `json:"integration_name" mapstructure:"integration_name"`
	DeploymentID    string                              `json:"deployment_id" mapstructure:"deployment_id"`
	DeploymentName  string                              `json:"deployment_name" mapstructure:"deployment_name"`
	StackID         string                              `json:"stack_id" mapstructure:"stack_id"`
	StackName       string                              `json:"stack_name" mapstructure:"stack_name"`
	StackPlatform   string                              `json:"stack_platform" mapstructure:"stack_platform"`
	Namespace       string                              `json:"namespace" mapstructure:"namespace"`
	Version         string                              `json:"version" mapstructure:"version"`
	Alternative     *ArsenalDeploymentDetailAlternative `json:"alternative" mapstructure:"alternative"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultArsenalDeploymentDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultArsenalDeploymentDetail) GetName() string {
	return v.DeploymentName
}

type ArsenalDeploymentDetailAlternative struct {
	Name    string                 `json:"name" mapstructure:"name"`
	From    string                 `json:"from,omitempty" mapstructure:"from"`
	Config  map[string]interface{} `json:"config" mapstructure:"config"`
	Options map[string]interface{} `json:"options,omitempty" mapstructure:"options"`
}

type V2WorkflowRunResultDockerDetail struct {
	Name         string `json:"name" mapstructure:"name"`
	ID           string `json:"id" mapstructure:"id"`
	HumanSize    string `json:"human_size" mapstructure:"human_size"`
	HumanCreated string `json:"human_created" mapstructure:"human_created"`
}

type V2WorkflowRunResultDebianDetail struct {
	Name          string   `json:"name" mapstructure:"name"`
	Size          int64    `json:"size" mapstructure:"size"`
	MD5           string   `json:"md5" mapstructure:"md5"`
	SHA1          string   `json:"sha1" mapstructure:"sha1"`
	SHA256        string   `json:"sha256" mapstructure:"sha256"`
	Components    []string `json:"components" mapstructure:"components"`
	Distributions []string `json:"distributions" mapstructure:"distributions"`
	Architectures []string `json:"architectures" mapstructure:"architectures"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultDebianDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultDebianDetail) GetName() string {
	return v.Name
}

type V2WorkflowRunResultPythonDetail struct {
	Name      string `json:"name" mapstructure:"name"`
	Version   string `json:"version" mapstructure:"version"`
	Extension string `json:"extension" mapstructure:"extension"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultPythonDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultPythonDetail) GetName() string {
	return v.Name + ":" + v.Version
}

type V2WorkflowRunResultHelmDetail struct {
	Name         string `json:"name" mapstructure:"name"`
	AppVersion   string `json:"appVersion" mapstructure:"appVersion"`
	ChartVersion string `json:"chartVersion" mapstructure:"chartVersion"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultHelmDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultHelmDetail) GetName() string {
	return v.Name + ":" + v.ChartVersion
}

const V2WorkflowRunResultVariableDetailType = "V2WorkflowRunResultVariableDetail"

type V2WorkflowRunResultVariableDetail struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultVariableDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, v)
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultVariableDetail) GetName() string {
	return v.Name
}

type V2WorkflowRunResultReleaseDetail struct {
	Name    string          `json:"name" mapstructure:"name"`
	Version string          `json:"version" mapstructure:"version"`
	SBOM    json.RawMessage `json:"sbom" mapstructure:"sbom"`
}

func (x *V2WorkflowRunResultReleaseDetail) Cast(i any) error {
	if !IsPointer(i) {
		return errors.New("unable to cast a non pointer")
	}
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   x,
	}
	// Here is the trick to transform the map to a json.RawMessage for the SBOM itself
	decoderConfig.DecodeHook = mapstructure.ComposeDecodeHookFunc(
		func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
			if f.Kind() != reflect.Map {
				return data, nil
			}
			result := reflect.New(t).Interface()
			_, ok := result.(*json.RawMessage)
			if !ok {
				return data, nil
			}
			btes, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}
			return json.RawMessage(btes), nil
		},
	)
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		panic(err)
	}
	if err := decoder.Decode(i); err != nil {
		return WrapError(err, "cannot unmarshal V2WorkflowRunResultReleaseDetail")
	}
	return nil
}

func (x *V2WorkflowRunResultReleaseDetail) GetName() string {
	return x.Name + ":" + x.Version
}

func (x *V2WorkflowRunResultDockerDetail) Cast(i any) error {
	return castV2WorkflowRunResultReleaseDetailWithMapStructure(i, x)
}

func (x *V2WorkflowRunResultDockerDetail) GetName() string {
	return x.Name
}

func castV2WorkflowRunResultReleaseDetailWithMapStructure[T any](input any, output T) error {
	if !IsPointer(output) {
		return errors.New("unable to cast a non pointer")
	}
	if err := mapstructure.Decode(input, output); err != nil {
		return WrapError(err, "cannot unmarshal V2WorkflowRunResultVariableDetail")
	}
	return nil
}
