package exportentities_test

import (
	"reflect"
	"testing"

	"github.com/ovh/cds/sdk/exportentities"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
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
  - type: jabber
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
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w exportentities.Workflow
			test.NoError(t, yaml.Unmarshal([]byte(tt.args.yaml), &w))
			if got := exportentities.CheckWorkflowNotificationsValidity(w); got != tt.want {
				t.Errorf("checkWorkflowNotificationsValidity() = %#v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processNotificationValues(t *testing.T) {
	type args struct {
		notif exportentities.NotificationEntry
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
			got, err := exportentities.ProcessNotificationValues(tt.args.notif)
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

func TestFromYAMLToYAMLWithNotif(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "test one pipeline with one notif",
			yaml: `name: test-notif-1
version: v1.0
pipeline: test
notify:
- type: jabber
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
`,
		}, {
			name: "test one pipeline with two notif",
			yaml: `name: test-notif-2
version: v1.0
pipeline: test
notify:
- type: jabber
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
- type: email
  settings:
    on_success: always
    on_failure: change
    on_start: true
    send_to_groups: true
    send_to_author: false
    recipients:
    - a
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
`,
		}, {
			name: "two pipelines with one notif",
			yaml: `name: test-notif-1
version: v1.0
workflow:
  test:
    pipeline: test
  test_2:
    depends_on:
    - test
    when:
    - success
    pipeline: test
notifications:
  test,test_2:
  - type: jabber
`,
		}, {
			name: "two pipelines with two notifs",
			yaml: `name: test-notif-1
version: v1.0
workflow:
  test:
    pipeline: test
  test_2:
    depends_on:
    - test
    when:
    - success
    pipeline: test
notifications:
  test:
  - type: email
    settings:
      on_success: always
      on_failure: change
      on_start: true
      send_to_groups: true
      send_to_author: false
      recipients:
      - a
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
  test,test_2:
  - type: jabber
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
`,
		},
	}
	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			var yamlWorkflow exportentities.Workflow
			err := exportentities.Unmarshal([]byte(tst.yaml), exportentities.FormatYAML, &yamlWorkflow)
			if err != nil {
				if !tst.wantErr {
					t.Error("Unmarshal raised an error", err)
					return
				}
			}
			if tst.wantErr {
				t.Error("Unmarshal should return an error but it doesn't")
				return
			}
			w, err := yamlWorkflow.GetWorkflow()
			if err != nil {
				if !tst.wantErr {
					t.Error("GetWorkflow raised an error", err)
					return
				}
			}
			if tst.wantErr {
				t.Error("GetWorkflow should return an error but it doesn't")
				return
			}

			// Set the hook and outgoing hook models properly before export all the things
			w.VisitNode(func(n *sdk.Node, w *sdk.Workflow) {
				for i := range n.Hooks {
					for _, m := range sdk.BuiltinHookModels {
						if n.Hooks[i].HookModelName == m.Name {
							break
						}
					}
				}
				if n.OutGoingHookContext != nil {
					for _, m := range sdk.BuiltinOutgoingHookModels {
						if n.OutGoingHookContext.HookModelName == m.Name {
							n.OutGoingHookContext.HookModelID = m.ID
							break
						}
					}
				}
			})
			exportedWorkflow, err := exportentities.NewWorkflow(*w)
			if err != nil {
				if !tst.wantErr {
					t.Error("NewWorkflow raised an error", err)
					return
				}
			}
			if tst.wantErr {
				t.Error("NewWorkflow should return an error but it doesn't")
				return
			}
			b, err := yaml.Marshal(exportedWorkflow)
			if err != nil {
				if !tst.wantErr {
					t.Error("Marshal raised an error", err)
					return
				}
			}
			if tst.wantErr {
				t.Error("Marshal should return an error but it doesn't")
				return
			}
			assert.Equal(t, tst.yaml, string(b))
		})
	}
}
