package exportentities

import (
	"reflect"
	"testing"

	"github.com/ovh/cds/sdk"
)

func Test_craftNotificationEntry(t *testing.T) {
	type args struct {
		w     sdk.Workflow
		notif sdk.WorkflowNotification
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		want1   NotificationEntry
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := craftNotificationEntry(tt.args.w, tt.args.notif)
			if (err != nil) != tt.wantErr {
				t.Errorf("craftNotificationEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("craftNotificationEntry() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("craftNotificationEntry() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_checkWorkflowNotificationsValidity(t *testing.T) {
	type args struct {
		w Workflow
	}
	tests := []struct {
		name string
		args args
		want *sdk.MultiError
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkWorkflowNotificationsValidity(tt.args.w); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkWorkflowNotificationsValidity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processNotificationValues(t *testing.T) {
	type args struct {
		notif NotificationEntry
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
			got, err := processNotificationValues(tt.args.notif)
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
