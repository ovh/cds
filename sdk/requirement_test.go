package sdk

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRequirementListDeduplicate(t *testing.T) {
	type args struct {
		l RequirementList
	}
	tests := []struct {
		name string
		args args
		want RequirementList
	}{
		{
			name: "test1",
			args: args{
				l: RequirementList{
					{
						Name:  "namea",
						Type:  NetworkAccessRequirement,
						Value: "valuea",
					},
					{
						Name:  "namea",
						Type:  NetworkAccessRequirement,
						Value: "valuea",
					},
					{
						Name:  "nameb",
						Type:  NetworkAccessRequirement,
						Value: "valueb",
					},
				},
			},
			want: RequirementList{
				{
					Name:  "namea",
					Type:  NetworkAccessRequirement,
					Value: "valuea",
				},
				{
					Name:  "nameb",
					Type:  NetworkAccessRequirement,
					Value: "valueb",
				},
			},
		},
		{
			name: "test2",
			args: args{
				l: RequirementList{
					{
						Name:  "namea",
						Type:  NetworkAccessRequirement,
						Value: "valuea",
					},
					{
						Name:  "nameb",
						Type:  NetworkAccessRequirement,
						Value: "valueb",
					},
					{
						Name:  "nameb",
						Type:  NetworkAccessRequirement,
						Value: "valueb",
					},
				},
			},
			want: RequirementList{
				{
					Name:  "namea",
					Type:  NetworkAccessRequirement,
					Value: "valuea",
				},
				{
					Name:  "nameb",
					Type:  NetworkAccessRequirement,
					Value: "valueb",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RequirementListDeduplicate(tt.args.l); !cmp.Equal(got, tt.want) {
				t.Errorf("RequirementListDeduplicate() = %v, want %v", got, tt.want)
			}
		})
	}
}
