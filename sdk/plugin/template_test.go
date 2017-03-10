package plugin

import "testing"

func TestApplyArguments(t *testing.T) {
	type args struct {
		variables map[string]string
		input     string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test template",
			args: args{
				variables: map[string]string{
					"cds.foo":    "valbar",
					"cds.title":  "mytitle",
					"cds.upper":  "toupper",
					"cds.lower":  "TOLOWER",
					"cds.escape": "a/b.c_d",
				},
				input: "a {{.cds.foo}} here, {{.cds.title | title}}, {{.cds.upper | upper}}, {{.cds.lower | lower}}, {{.cds.escape | escape}}",
			},
			want:    "a valbar here, Mytitle, TOUPPER, tolower, a-b-c-d",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyArguments(tt.args.variables, []byte(tt.args.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyArguments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("ApplyArguments() = %v, want %v", got, tt.want)
			}
		})
	}
}
