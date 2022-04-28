package v1_test

import (
	"reflect"
	"testing"

	v1 "github.com/ovh/cds/sdk/exportentities/v1"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	yaml "gopkg.in/yaml.v2"
)

func Test_checkWorkflowNotificationsValidity(t *testing.T) {

	type args struct {
		yaml string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "test multiple notifications",
			want: nil,
			args: args{
				yaml: `name: test1
version: v1.0
workflow:
  DDOS-me:
    pipeline: DDOS-me
    application: test1
    payload:
      git.author: ""
      git.branch: master
      git.hash: ""
      git.hash.before: ""
      git.message: ""
      git.repository: bnjjj/godevoxx
      git.tag: ""
  DDOS-me_2:
    depends_on:
    - DDOS-me
    when:
    - success
    pipeline: DDOS-me
metadata:
  default_tags: git.branch,git.author
notifications:
  DDOS-me,DDOS-me_2:
  - type: email
    settings:
      on_success: always
      on_failure: change
      on_start: true
      send_to_groups: true
      send_to_author: false
      recipients:
      - q
      template:
        subject: '{{.cds.project}}/{{.cds.application}} {{.cds.pipeline}} {{.cds.environment}}#{{.cds.version}}
          {{.cds.status}}'
        body: |-
          Project : {{.cds.project}}
          Application : {{.cds.application}}
          Pipeline : {{.cds.pipeline}}/{{.cds.environment}}#{{.cds.version}}
          Status : {{.cds.status}}
          Details : {{.cds.buildURL}}
          Triggered by : {{.cds.triggered_by.username}}
          Branch : {{.git.branch}}
  DDOS-me_2:
  - type: email
    settings:
      template:
        subject: '{{.cds.project}}/{{.cds.application}} {{.cds.pipeline}} {{.cds.environment}}#{{.cds.version}}
          {{.cds.status}}'
        body: |-
          Project : {{.cds.project}}
          Application : {{.cds.application}}
          Pipeline : {{.cds.pipeline}}/{{.cds.environment}}#{{.cds.version}}
          Status : {{.cds.status}}
          Details : {{.cds.buildURL}}
          Triggered by : {{.cds.triggered_by.username}}
          Branch : {{.git.branch}}
  - type: vcs
    settings:
      template:
        disable_comment: true
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w v1.Workflow
			test.NoError(t, yaml.Unmarshal([]byte(tt.args.yaml), &w))
			if got := v1.CheckWorkflowNotificationsValidity(w); got != tt.want {
				t.Errorf("checkWorkflowNotificationsValidity() = %#v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processNotificationValues(t *testing.T) {
	type args struct {
		notif v1.NotificationEntry
	}
	tests := []struct {
		name    string
		args    args
		want    sdk.WorkflowNotification
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.ProcessNotificationValues(tt.args.notif)
			if (err != nil) != tt.wantErr {
				t.Errorf("processNotificationValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processNotificationValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
