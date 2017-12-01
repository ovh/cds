package sanity

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	_ "github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func Test_checkActionRequirements(t *testing.T) {
	log.SetLogger(t)

	type args struct {
		a    *sdk.Action
		proj string
		pip  string
		wms  []sdk.Model
	}
	tests := []struct {
		name    string
		args    args
		want    []sdk.Warning
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		got, err := checkActionRequirements(tt.args.a, tt.args.proj, tt.args.pip, tt.args.wms)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. checkActionRequirements() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkActionRequirements() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_checkMultipleWorkerModelWarning(t *testing.T) {
	log.SetLogger(t)

	type args struct {
		proj string
		pip  string
		a    *sdk.Action
	}
	tests := []struct {
		name           string
		args           args
		wantWarnings   []sdk.Warning
		wantModelCount int
		wantErr        bool
	}{
		{
			name: "With 2 worker models it should return 2 warnings",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "model-1",
							Type:  sdk.ModelRequirement,
							Value: "model-1",
						},
						{
							Name:  "model-2",
							Type:  sdk.ModelRequirement,
							Value: "model-2",
						},
					},
				},
			},
			wantWarnings: []sdk.Warning{
				{
					Action: sdk.Action{
						ID: 1,
					},
					ID: MultipleWorkerModelWarning,
					MessageParam: map[string]string{
						"ActionName":   "Action Name 1",
						"PipelineName": "pipeline",
						"ProjectKey":   "proj",
					},
				},
			},
			wantModelCount: 2,
			wantErr:        false,
		},
		{
			name: "With 1 worker model it should not return warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "model-1",
							Type:  sdk.ModelRequirement,
							Value: "model-1",
						},
					},
				},
			},
			wantWarnings:   nil,
			wantModelCount: 1,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		got, got1, _, err := checkMultipleWorkerModelWarning(tt.args.proj, tt.args.pip, tt.args.a)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. checkMultipleWorkerModelWarning() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		assert.EqualValues(t, tt.wantWarnings, got)
		assert.Equal(t, tt.wantModelCount, got1)
	}
}

func Test_checkMultipleHostnameWarning(t *testing.T) {
	log.SetLogger(t)

	type args struct {
		proj string
		pip  string
		a    *sdk.Action
	}
	tests := []struct {
		name    string
		args    args
		want    []sdk.Warning
		want1   int
		wantErr bool
	}{
		{
			name: "With 2 hostname it should return 1 warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "host-1",
							Type:  sdk.HostnameRequirement,
							Value: "host-1",
						},
						{
							Name:  "host-2",
							Type:  sdk.HostnameRequirement,
							Value: "host-2",
						},
					},
				},
			},
			want: []sdk.Warning{
				{
					Action: sdk.Action{
						ID: 1,
					},
					ID: MultipleHostnameRequirement,
					MessageParam: map[string]string{
						"ActionName":   "Action Name 1",
						"PipelineName": "pipeline",
						"ProjectKey":   "proj",
					},
				},
			},
			want1:   2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, got1, err := checkMultipleHostnameWarning(tt.args.proj, tt.args.pip, tt.args.a)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. checkMultipleHostnameWarning() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkMultipleHostnameWarning() got = %v, want %v", tt.name, got, tt.want)
		}
		if got1 != tt.want1 {
			t.Errorf("%q. checkMultipleHostnameWarning() got1 = %v, want %v", tt.name, got1, tt.want1)
		}
	}
}

func Test_checkNoWorkerModelMatchRequirement(t *testing.T) {
	log.SetLogger(t)

	type args struct {
		proj string
		pip  string
		a    *sdk.Action
		wms  []sdk.Model
	}
	tests := []struct {
		name    string
		args    args
		want    []sdk.Warning
		wantErr bool
	}{
		{
			name: "With 1 missing capa it should return 1 warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "binary-1",
							Type:  sdk.BinaryRequirement,
							Value: "binary-1",
						},
						{
							Name:  "binary-2",
							Type:  sdk.BinaryRequirement,
							Value: "binary-2",
						},
					},
				},
				wms: []sdk.Model{
					{
						Capabilities: []sdk.Requirement{
							{
								Value: "binary-1",
							},
							{
								Value: "binary-3",
							},
						},
					},
				},
			},
			want: []sdk.Warning{
				{
					Action: sdk.Action{
						ID: 1,
					},
					ID: NoWorkerModelMatchRequirement,
					MessageParam: map[string]string{
						"ActionName":   "Action Name 1",
						"PipelineName": "pipeline",
						"ProjectKey":   "proj",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "With no missing capa it should not return warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "binary-1",
							Type:  sdk.BinaryRequirement,
							Value: "binary-1",
						},
						{
							Name:  "binary-2",
							Type:  sdk.BinaryRequirement,
							Value: "binary-2",
						},
					},
				},
				wms: []sdk.Model{
					{
						Capabilities: []sdk.Requirement{
							{
								Value: "binary-1",
							},
							{
								Value: "binary-2",
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "With a service req but not a docker model it should return 1 warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "binary-1",
							Type:  sdk.BinaryRequirement,
							Value: "binary-1",
						},
						{
							Name:  "binary-2",
							Type:  sdk.BinaryRequirement,
							Value: "binary-2",
						},
						{
							Name:  "service-1",
							Type:  sdk.ServiceRequirement,
							Value: "service-1",
						},
					},
				},
				wms: []sdk.Model{
					sdk.Model{
						Type: sdk.Openstack,
						Capabilities: []sdk.Requirement{
							{
								Value: "binary-1",
							},
							{
								Value: "binary-2",
							},
						},
					},
				},
			},
			want: []sdk.Warning{
				{
					Action: sdk.Action{
						ID: 1,
					},
					ID: NoWorkerModelMatchRequirement,
					MessageParam: map[string]string{
						"ActionName":   "Action Name 1",
						"PipelineName": "pipeline",
						"ProjectKey":   "proj",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "With a service req with a docker model it should not return warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "binary-1",
							Type:  sdk.BinaryRequirement,
							Value: "binary-1",
						},
						{
							Name:  "binary-2",
							Type:  sdk.BinaryRequirement,
							Value: "binary-2",
						},
						{
							Name:  "service-1",
							Type:  sdk.ServiceRequirement,
							Value: "service-1",
						},
					},
				},
				wms: []sdk.Model{
					sdk.Model{
						Type: sdk.Docker,
						Capabilities: []sdk.Requirement{
							{
								Value: "binary-1",
							},
							{
								Value: "binary-2",
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := checkNoWorkerModelMatchRequirement(tt.args.proj, tt.args.pip, tt.args.a, tt.args.wms, 0, 0)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. checkNoWorkerModelMatchRequirement() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkNoWorkerModelMatchRequirement() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_checkIncompatibleBinaryWithModelRequirement(t *testing.T) {
	log.SetLogger(t)

	type args struct {
		proj      string
		pip       string
		a         *sdk.Action
		wms       []sdk.Model
		modelName string
	}
	tests := []struct {
		name    string
		args    args
		want    []sdk.Warning
		wantErr bool
	}{
		{
			name: "With a model matching all requirements it should not return warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "binary-1",
							Type:  sdk.BinaryRequirement,
							Value: "binary-1",
						},
						{
							Name:  "binary-2",
							Type:  sdk.BinaryRequirement,
							Value: "binary-2",
						},
					},
				},
				modelName: "model",
				wms: []sdk.Model{
					sdk.Model{
						Name: "model",
						Capabilities: []sdk.Requirement{
							{
								Type:  sdk.BinaryRequirement,
								Value: "binary-1",
							},
							{
								Type:  sdk.BinaryRequirement,
								Value: "binary-2",
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "With a model with missing capa it should return 1 warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "binary-1",
							Type:  sdk.BinaryRequirement,
							Value: "binary-1",
						},
						{
							Name:  "binary-2",
							Type:  sdk.BinaryRequirement,
							Value: "binary-2",
						},
					},
				},
				modelName: "model",
				wms: []sdk.Model{
					sdk.Model{
						Name: "model",
						Capabilities: []sdk.Requirement{
							{
								Type:  sdk.BinaryRequirement,
								Value: "binary-1",
							},
						},
					},
				},
			},
			want: []sdk.Warning{
				{
					Action: sdk.Action{
						ID: 1,
					},
					ID: IncompatibleBinaryAndModelRequirements,
					MessageParam: map[string]string{
						"ActionName":        "Action Name 1",
						"PipelineName":      "pipeline",
						"ProjectKey":        "proj",
						"ModelName":         "model",
						"BinaryRequirement": "binary-2",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		got, err := checkIncompatibleBinaryWithModelRequirement(tt.args.proj, tt.args.pip, tt.args.a, tt.args.wms, tt.args.modelName)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. checkIncompatibleBinaryWithModelRequirement() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		assert.EqualValues(t, tt.want, got)
	}
}

func Test_checkIncompatibleServiceWithModelRequirement(t *testing.T) {
	log.SetLogger(t)

	type args struct {
		proj      string
		pip       string
		a         *sdk.Action
		wms       []sdk.Model
		modelName string
	}
	tests := []struct {
		name    string
		args    args
		want    []sdk.Warning
		wantErr bool
	}{
		{
			name: "With a model matching all requirements it should not return warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "service-1",
							Type:  sdk.ServiceRequirement,
							Value: "service-1",
						},
					},
				},
				modelName: "model",
				wms: []sdk.Model{
					sdk.Model{
						Name: "model",
						Type: sdk.Docker,
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "With a model != docker it should return 1 warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "service-1",
							Type:  sdk.ServiceRequirement,
							Value: "service-1",
						},
					},
				},
				modelName: "model",
				wms: []sdk.Model{
					sdk.Model{
						Name: "model",
						Type: sdk.Openstack,
					},
				},
			},
			want: []sdk.Warning{
				{
					Action: sdk.Action{
						ID: 1,
					},
					ID: IncompatibleServiceAndModelRequirements,
					MessageParam: map[string]string{
						"ActionName":         "Action Name 1",
						"PipelineName":       "pipeline",
						"ProjectKey":         "proj",
						"ModelName":          "model",
						"ServiceRequirement": "service-1",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := checkIncompatibleServiceWithModelRequirement(tt.args.proj, tt.args.pip, tt.args.a, tt.args.wms, tt.args.modelName)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. checkIncompatibleServiceWithModelRequirement() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		assert.EqualValues(t, tt.want, got)
	}
}

func Test_checkIncompatibleMemoryWithModelRequirement(t *testing.T) {
	log.SetLogger(t)

	type args struct {
		proj      string
		pip       string
		a         *sdk.Action
		wms       []sdk.Model
		modelName string
	}
	tests := []struct {
		name    string
		args    args
		want    []sdk.Warning
		wantErr bool
	}{
		{
			name: "With a model matching all requirements it should not return warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "service-1",
							Type:  sdk.MemoryRequirement,
							Value: "service-1",
						},
					},
				},
				modelName: "model",
				wms: []sdk.Model{
					sdk.Model{
						Name: "model",
						Type: sdk.Docker,
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "With a model != docker it should return 1 warning",
			args: args{
				proj: "proj",
				pip:  "pipeline",
				a: &sdk.Action{
					ID:   1,
					Name: "Action Name 1",
					Requirements: []sdk.Requirement{
						{
							Name:  "service-1",
							Type:  sdk.MemoryRequirement,
							Value: "service-1",
						},
					},
				},
				modelName: "model",
				wms: []sdk.Model{
					sdk.Model{
						Name: "model",
						Type: sdk.Openstack,
					},
				},
			},
			want: []sdk.Warning{
				{
					Action: sdk.Action{
						ID: 1,
					},
					ID: IncompatibleMemoryAndModelRequirements,
					MessageParam: map[string]string{
						"ActionName":   "Action Name 1",
						"PipelineName": "pipeline",
						"ProjectKey":   "proj",
						"ModelName":    "model",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := checkIncompatibleMemoryWithModelRequirement(tt.args.proj, tt.args.pip, tt.args.a, tt.args.wms, tt.args.modelName)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. checkIncompatibleMemoryWithModelRequirement() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		assert.EqualValues(t, tt.want, got)
	}
}
