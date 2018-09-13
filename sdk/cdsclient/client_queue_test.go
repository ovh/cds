package cdsclient

import (
	"reflect"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"
)

func Test_shrinkQueue(t *testing.T) {
	now := time.Now()
	t10, _ := time.Parse(time.RFC3339, "2018-09-01T10:00:00+00:00")
	t11, _ := time.Parse(time.RFC3339, "2018-09-01T11:00:00+00:00")
	t12, _ := time.Parse(time.RFC3339, "2018-09-01T12:00:00+00:00")
	t13, _ := time.Parse(time.RFC3339, "2018-09-01T13:00:00+00:00")
	t14, _ := time.Parse(time.RFC3339, "2018-09-01T14:00:00+00:00")
	t15, _ := time.Parse(time.RFC3339, "2018-09-01T15:00:00+00:00")

	type args struct {
		queue *sdk.WorkflowQueue
		l     int
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{
			name: "simple",
			args: args{queue: &sdk.WorkflowQueue{
				{
					ProjectID:     1,
					ID:            1,
					Queued:        t10,
					QueuedSeconds: now.Unix() - t10.Unix(),
				},
				{
					ProjectID:     1,
					ID:            2,
					Queued:        t11,
					QueuedSeconds: now.Unix() - t11.Unix(),
				},
				{
					ProjectID:     1,
					ID:            3,
					Queued:        t12,
					QueuedSeconds: now.Unix() - t12.Unix(),
				},
				{
					ProjectID:     2,
					ID:            4,
					Queued:        t13,
					QueuedSeconds: now.Unix() - t13.Unix(),
				},
				{
					ProjectID:     1,
					ID:            5,
					Queued:        t14,
					QueuedSeconds: now.Unix() - t14.Unix(),
				},
				{
					ProjectID:     3,
					ID:            6,
					Queued:        t15,
					QueuedSeconds: now.Unix() - t15.Unix(),
				},
			},
				l: 4,
			},
			want: t10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shrinkQueue(tt.args.queue, tt.args.l); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("shrinkQueue() = %v, want %v", got, tt.want)
			}
			for _, q := range *tt.args.queue {
				t.Logf("Project:%d ID:%d Queued: %s\n", q.ProjectID, q.ID, q.Queued)
			}
		})
	}
}
