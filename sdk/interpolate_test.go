package sdk

import "testing"

func TestInterpolate(t *testing.T) {
	type args struct {
		input string
		vars  map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "simple",
			args: args{
				input: "a {{.cds.app.value}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want: "a value",
		},
		{
			name: "only unknown",
			args: args{
				input: "a value unknown {{.cds.app.foo}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want: "a value unknown {{.cds.app.foo}}",
		},
		{
			name: "simple with unknown",
			args: args{
				input: "a {{.cds.app.value}} and another value unknown {{.cds.app.foo}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want: "a value and another value unknown {{.cds.app.foo}}",
		},
		{
			name: "upper",
			args: args{
				input: "a {{.cds.app.value | upper}} and another value unknown {{.cds.app.foo}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want: "a VALUE and another value unknown {{.cds.app.foo}}",
		},
		{
			name: "title and filter on unknow",
			args: args{
				input: "a {{.cds.app.value | title }} and another value unknown {{.cds.app.foo | lower}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want: "a Value and another value unknown {{.cds.app.foo | lower}}",
		},
		{
			name: "many",
			args: args{
				input: "{{.cds.app.bar}} a {{.cds.app.valuea | upper }}, a {{.cds.app.valueb | title}}.{{.cds.app.valuec}}-{{.cds.app.foo}}",
				vars:  map[string]string{"cds.app.valuea": "valuea", "cds.app.valueb": "valueb", "cds.app.valuec": "valuec"},
			},
			want: "{{.cds.app.bar}} a VALUEA, a Valueb.valuec-{{.cds.app.foo}}",
		},
		{
			name: "two same unknown",
			args: args{
				input: `A:{{.cds.env.myenvpassword}} B:{{.cds.env.myenvpassword}}`,
				vars:  map[string]string{},
			},
			want: `A:{{.cds.env.myenvpassword}} B:{{.cds.env.myenvpassword}}`,
		},
		{
			name: "empty string",
			args: args{
				input: "a {{.cds.app.myKey}} and another key with empty value *{{.cds.app.myKeyAnother}}*",
				vars:  map[string]string{"cds.app.myKey": "valueKey", "cds.app.myKeyAnother": ""},
			},
			want: "a valueKey and another key with empty value **",
		},
		{
			name: "two keys with same first characters",
			args: args{
				input: "a {{.cds.app.myKey}} and another key value {{.cds.app.myKeyAnother}}",
				vars:  map[string]string{"cds.app.myKey": "valueKey", "cds.app.myKeyAnother": "valueKeyAnother"},
			},
			want: "a valueKey and another key value valueKeyAnother",
		},
		{
			name: "config",
			args: args{
				input: `
{
"env": {
"KEYA":"{{.cds.env.vAppKey}}",
"KEYB": "{{.cds.env.vAppKeyHatchery}}",
"ADDR":"{{.cds.env.addr}}"
},
"labels": {
"TOKEN": "{{.cds.env.token}}",
"HOST": "cds-hatchery-marathon-{{.cds.env.name}}.{{.cds.env.vHost}}",
}
}`,
				vars: map[string]string{"cds.env.name": "", "cds.env.token": "aValidTokenString", "cds.env.addr": "", "cds.env.vAppKey": "aValue"},
			},
			want: `
{
"env": {
"KEYA":"aValue",
"KEYB": "{{.cds.env.vAppKeyHatchery}}",
"ADDR":""
},
"labels": {
"TOKEN": "aValidTokenString",
"HOST": "cds-hatchery-marathon-.{{.cds.env.vHost}}",
}
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Interpolate(tt.args.input, tt.args.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("Interpolate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Interpolate() = %v, want %v", got, tt.want)
			}
		})
	}
}
