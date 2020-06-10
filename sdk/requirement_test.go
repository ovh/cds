package sdk

import (
	"testing"
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
						Type:  RegionRequirement,
						Value: "valuea",
					},
					{
						Name:  "namea",
						Type:  RegionRequirement,
						Value: "valuea",
					},
					{
						Name:  "nameb",
						Type:  RegionRequirement,
						Value: "valueb",
					},
				},
			},
			want: RequirementList{
				{
					Name:  "namea",
					Type:  RegionRequirement,
					Value: "valuea",
				},
				{
					Name:  "nameb",
					Type:  RegionRequirement,
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
						Type:  RegionRequirement,
						Value: "valuea",
					},
					{
						Name:  "nameb",
						Type:  RegionRequirement,
						Value: "valueb",
					},
					{
						Name:  "nameb",
						Type:  RegionRequirement,
						Value: "valueb",
					},
				},
			},
			want: RequirementList{
				{
					Name:  "nameb",
					Type:  RegionRequirement,
					Value: "valueb",
				},
				{
					Name:  "namea",
					Type:  RegionRequirement,
					Value: "valuea",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RequirementListDeduplicate(tt.args.l)
			for _, r := range tt.want {
				var found bool
				for _, g := range got {
					if r.Type == g.Type && r.Value == g.Value && r.Name == g.Name {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("RequirementListDeduplicate() = %v, want %v - not found: %v", got, tt.want, r)
				}
			}
		})
	}
}
