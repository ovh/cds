package sdk

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/cds/sdk/artifact_manager/artifactory/xray"
	"github.com/pkg/errors"
	"github.com/rockbears/log"
)

var (
	registeredV2WorkflowRunResultDetail     map[string]V2WorkflowRunResultDetailInterface = map[string]V2WorkflowRunResultDetailInterface{}
	registeredV2WorkflowRunResultDetailLock sync.Mutex
)

func registerV2WorkflowRunResultDetail(datas ...V2WorkflowRunResultDetailInterface) {
	registeredV2WorkflowRunResultDetailLock.Lock()
	defer registeredV2WorkflowRunResultDetailLock.Unlock()
	for _, x := range datas {
		if !IsPointer(x) {
			panic(fmt.Sprintf("%T is not a pointer", x))
		}
		name := strings.TrimLeft(fmt.Sprintf("%T", x), "*.sdk")
		registeredV2WorkflowRunResultDetail[name] = x
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
		&V2WorkflowRunResultStaticFilesDetail{},
	)
}

type V2WorkflowRunResultDetailInterface interface {
	Cast(i any) error
	GetName() string
	GetLabel() string
	GetMetadata() map[string]V2WorkflowRunResultDetailMetadata
}

type V2WorkflowRunResultDetailMetadataType string

const (
	V2WorkflowRunResultDetailMetadataTypeText   = "TEXT"
	V2WorkflowRunResultDetailMetadataTypeNUMBER = "NUMBER"
	V2WorkflowRunResultDetailMetadataTypeURL    = "URL"
)

type V2WorkflowRunResultDetailMetadata struct {
	Type  V2WorkflowRunResultDetailMetadataType `json:"type"` // number, text, url
	Value string                                `json:"value"`
}

type V2WorkflowRunResultDetail struct {
	Data interface{} `json:"data"`
	Type string      `json:"type"`
}

func (s *V2WorkflowRunResult) CastDetail() error {
	return s.Detail.castDetail()
}

func (s *V2WorkflowRunResultDetail) castDetail() error {
	data := s.Data

	if IsPointer(data) {
		data = reflect.ValueOf(data).Elem().Interface()
	}

	typeOfData := reflect.TypeOf(data)
	if typeOfData.Kind() == reflect.Struct {
		for k := range registeredV2WorkflowRunResultDetail {
			if k == typeOfData.Name() {
				if s.Type == "" {
					s.Type = typeOfData.Name()
				}
				return nil
			}
		}
	}

	for k, v := range registeredV2WorkflowRunResultDetail {
		if k == s.Type {
			typeOfV := reflect.TypeOf(v)
			x := reflect.New(typeOfV.Elem())
			xi, ok := x.Interface().(V2WorkflowRunResultDetailInterface)
			if !ok {
				return errors.Errorf("unknow type %q (%T): unable to cast to V2WorkflowRunResultDetailInterface", s.Type, x)
			}
			if err := xi.Cast(s.Data); err != nil {
				return err
			}
			s.Data = xi
			return nil
		}
	}
	return errors.Errorf("unknow type %q (%s)", s.Type, typeOfData.Name())
}

func GetConcreteDetail[T any](s *V2WorkflowRunResult) (t T, err error) {
	i, err := s.GetDetail()
	if err != nil {
		return t, err
	}
	x, ok := i.(T)
	if !ok {
		var tt T
		return t, errors.Errorf("unable to get concrete detail for type %q (expected: %T, actual: %T)", s.Detail.Type, tt, i)
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

	btes, err := json.Marshal(content)
	return btes, err
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
	V2WorkflowRunResultTypeStaticFiles       = "staticFiles"
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTestDetail) GetLabel() string {
	return fmt.Sprintf("Filename: %s", v.Name)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTestDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Filename":      {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"Total":         {Type: V2WorkflowRunResultDetailMetadataTypeNUMBER, Value: strconv.Itoa(v.TestStats.Total)},
		"Total passed":  {Type: V2WorkflowRunResultDetailMetadataTypeNUMBER, Value: strconv.Itoa(v.TestStats.TotalOK)},
		"Total failed":  {Type: V2WorkflowRunResultDetailMetadataTypeNUMBER, Value: strconv.Itoa(v.TestStats.TotalKO)},
		"Total skipped": {Type: V2WorkflowRunResultDetailMetadataTypeNUMBER, Value: strconv.Itoa(v.TestStats.TotalSkipped)},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTestDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, v)
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformModuleDetail) GetLabel() string {
	return fmt.Sprintf("Name: %s - Version: %s - Provider: %s - Namespace: %s", v.Name, v.Version, v.Provider, v.Namespace)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformModuleDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Name":      {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"Version":   {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Version},
		"Provider":  {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Provider},
		"Namespace": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Namespace},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformModuleDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, v)
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformProviderDetail) GetLabel() string {
	return fmt.Sprintf("Name: %s - Version: %s - Namespace: %s - Flavor: %s", v.Name, v.Version, v.Namespace, v.Flavor)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformProviderDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Name":      {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"Version":   {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Version},
		"Namespace": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Namespace},
		"Flavor":    {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Flavor},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultTerraformProviderDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, v)
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultGenericDetail) GetLabel() string {
	return fmt.Sprintf("Filename: %s - Size: %s", v.Name, humanize.Bytes(uint64(v.Size)))
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultGenericDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Filename":     {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"Size (bytes)": {Type: V2WorkflowRunResultDetailMetadataTypeNUMBER, Value: strconv.FormatInt(v.Size, 10)},
		"MD5":          {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.MD5},
		"SHA1":         {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.SHA1},
		"SHA256":       {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.SHA256},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultGenericDetail) Cast(i any) error {
	if err := castV2WorkflowRunResultDetailWithMapStructure(i, v); err != nil {
		return err
	}
	return nil
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultArsenalDeploymentDetail) GetLabel() string {
	return fmt.Sprintf("Name: %s - %s - version: %s", v.DeploymentName, v.IntegrationName, v.Version)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultArsenalDeploymentDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	x := map[string]V2WorkflowRunResultDetailMetadata{
		"Deployment name": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.DeploymentName},
		"Deployment ID":   {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.DeploymentID},
		"Version":         {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Version},
		"Stack name":      {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.StackName},
		"Stack ID":        {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.StackID},
		"Stack platform":  {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.StackPlatform},
		"Namespace":       {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Namespace},
	}
	if v.Alternative != nil {
		x["Alternative"] = V2WorkflowRunResultDetailMetadata{Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Alternative.Name}
	}
	return x
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultArsenalDeploymentDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, v)
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (x *V2WorkflowRunResultDockerDetail) GetLabel() string {
	return fmt.Sprintf("Image: %s", x.Name)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (x *V2WorkflowRunResultDockerDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Image": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: x.Name},
		"ID":    {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: x.ID},
	}
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultDebianDetail) GetLabel() string {
	return fmt.Sprintf("Package: %s - Size: %s", v.Name, humanize.Bytes(uint64(v.Size)))
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultDebianDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Package":       {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"Size (bytes)":  {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: strconv.FormatInt(v.Size, 10)},
		"MD5":           {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.MD5},
		"SHA1":          {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.SHA1},
		"SHA256":        {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.SHA256},
		"Components":    {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: strings.Join(v.Components, " ")},
		"Architectures": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: strings.Join(v.Architectures, " ")},
		"Distributions": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: strings.Join(v.Distributions, " ")},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultDebianDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, v)
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultPythonDetail) GetLabel() string {
	return fmt.Sprintf("Package: %s - Version: %s", v.Name, v.Extension)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultPythonDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Package":   {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"Version":   {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Version},
		"Extension": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Extension},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultPythonDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, v)
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultHelmDetail) GetLabel() string {
	return fmt.Sprintf("Chart: %s - Version: %s", v.Name, v.ChartVersion)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultHelmDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Chart":        {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"AppVersion":   {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.AppVersion},
		"ChartVersion": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.ChartVersion},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultHelmDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, v)
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultVariableDetail) GetLabel() string {
	return fmt.Sprintf("Name: %q - Value: %q", v.Name, v.Value)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultVariableDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Name":  {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"Value": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Value},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultVariableDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, v)
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

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (x *V2WorkflowRunResultReleaseDetail) GetLabel() string {
	return fmt.Sprintf("Name: %q - Version: %q", x.Name, x.Version)
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (x *V2WorkflowRunResultReleaseDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	var sbom xray.CycloneDXReport
	if err := json.Unmarshal(x.SBOM, &sbom); err != nil {
		log.Error(context.Background(), "unable to parse sbom as CycloneDXReport: %v", err)
		return map[string]V2WorkflowRunResultDetailMetadata{
			"Name":    {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: x.Name},
			"Version": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: x.Version},
		}
	}

	results := map[string]V2WorkflowRunResultDetailMetadata{
		"Name":    {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: x.Name},
		"Version": {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: x.Version},
	}

	for i, c := range sbom.Components {
		results[fmt.Sprintf("Component[%d]", i)] = V2WorkflowRunResultDetailMetadata{Type: V2WorkflowRunResultDetailMetadataTypeText, Value: fmt.Sprintf("[%s] %s %s", c.Type, c.Name, c.Version)}
	}

	return results
}

func (x *V2WorkflowRunResultReleaseDetail) Cast(i any) error {
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
		log.ErrorWithStackTrace(context.Background(), err)
		panic(err)
	}

	if err := decoder.Decode(i); err != nil {
		log.ErrorWithStackTrace(context.Background(), err)

		return WrapError(err, "cannot unmarshal V2WorkflowRunResultReleaseDetail")
	}
	return nil
}

func (x *V2WorkflowRunResultReleaseDetail) GetName() string {
	return x.Name + ":" + x.Version
}

func (x *V2WorkflowRunResultDockerDetail) Cast(i any) error {
	return castV2WorkflowRunResultDetailWithMapStructure(i, x)
}

func (x *V2WorkflowRunResultDockerDetail) GetName() string {
	return x.Name
}

func castV2WorkflowRunResultDetailWithMapStructure[T any](input any, output T) error {
	if !IsPointer(output) {
		return errors.New("unable to cast a non pointer")
	}
	ttI := reflect.TypeOf(input)
	ttO := reflect.TypeOf(output)
	if err := mapstructure.Decode(input, output); err != nil {
		return WrapError(err, "cannot unmarshal %s to %s", ttI.Name(), ttO.Name())
	}
	return nil
}

type V2WorkflowRunResultStaticFilesDetail struct {
	Name           string `json:"name" mapstructure:"name"`
	ArtifactoryURL string `json:"artifactory_url" mapstructure:"artifactory_url"`
	PublicURL      string `json:"public_url" mapstructure:"public_url"`
}

// GetLabel implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultStaticFilesDetail) GetLabel() string {
	return v.Name
}

// GetMetadata implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultStaticFilesDetail) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	return map[string]V2WorkflowRunResultDetailMetadata{
		"Name":            {Type: V2WorkflowRunResultDetailMetadataTypeText, Value: v.Name},
		"URL":             {Type: V2WorkflowRunResultDetailMetadataTypeURL, Value: v.PublicURL},
		"Artifactory URL": {Type: V2WorkflowRunResultDetailMetadataTypeURL, Value: v.ArtifactoryURL},
	}
}

// Cast implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultStaticFilesDetail) Cast(i any) error {
	if err := castV2WorkflowRunResultDetailWithMapStructure(i, v); err != nil {
		return err
	}
	return nil
}

// GetName implements V2WorkflowRunResultDetailInterface.
func (v *V2WorkflowRunResultStaticFilesDetail) GetName() string {
	return v.Name
}
