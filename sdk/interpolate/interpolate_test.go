package interpolate

import (
	"fmt"
	"testing"
)

func BenchmarkDo(b *testing.B) {
	for n := 0; n < b.N; n++ {

		type args struct {
			input string
			vars  map[string]string
		}
		test := struct {
			args args
		}{
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
		}

		for i := 0; i < 60; i++ {
			test.args.vars[fmt.Sprintf("%d", i)] = fmt.Sprintf(">>%d<<", i)
		}

		Do(test.args.input, test.args.vars)
	}
}

func BenchmarkDoNothing(b *testing.B) {
	for n := 0; n < b.N; n++ {

		type args struct {
			input string
			vars  map[string]string
		}
		test := struct {
			args args
		}{
			args: args{
				input: `
				Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aliquam felis nulla, vulputate ac eros vel, placerat dignissim turpis. Sed et ex lectus. Donec viverra nisi vel dictum rhoncus. Sed dictum tempus quam, ut efficitur arcu viverra vitae. Suspendisse aliquam venenatis scelerisque. Praesent et mattis enim. In efficitur imperdiet nulla a sagittis. Maecenas aliquet magna in sollicitudin ornare.

				Suspendisse viverra enim nec ante blandit tempus. Sed ut erat suscipit, semper ex eu, eleifend neque. Sed orci justo, bibendum laoreet libero cursus, venenatis fringilla dui. Curabitur tristique odio ut neque sollicitudin ultrices. Integer metus nibh, dignissim non pellentesque et, volutpat vel ante. Pellentesque ultrices ante vel mauris aliquam porttitor. Nunc nec sem facilisis, ullamcorper ex sed, elementum elit. Nulla risus magna, tempor et ultricies id, vehicula ac massa. Mauris venenatis libero libero, id lobortis mi semper aliquam. Aenean neque turpis, feugiat vel rutrum vitae, auctor quis nisl. Donec placerat nec mauris vitae malesuada. Proin quis gravida nulla. Pellentesque in pellentesque metus, in finibus dui. Sed rutrum, libero sit amet cursus scelerisque, sem orci condimentum nunc, quis egestas tellus orci ac nisl. Mauris viverra tincidunt diam ac sollicitudin. Nunc venenatis, nibh at laoreet pellentesque, lacus tellus molestie lorem, et sollicitudin nunc neque ut turpis.
				`,
				vars: map[string]string{"cds.env.name": "", "cds.env.token": "aValidTokenString", "cds.env.addr": "", "cds.env.vAppKey": "aValue"},
			},
		}

		for i := 0; i < 60; i++ {
			test.args.vars[fmt.Sprintf("%d", i)] = fmt.Sprintf(">>%d<<", i)
		}

		Do(test.args.input, test.args.vars)

	}
}

func TestDo(t *testing.T) {
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
			name: "default value with empty default",
			args: args{
				input: `aa:{{.cds.app.foo | default ""}}end`,
				vars:  map[string]string{},
			},
			want: `aa:end`,
		},
		{
			name: "default value",
			args: args{
				input: `aa:{{.cds.app.foo | default "bar" }}end`,
				vars:  map[string]string{},
			},
			want: `aa:barend`,
		},
		{
			name: "default value with knowned var",
			args: args{
				input: `aa:{{.cds.app.foo | default "bar"}}end`,
				vars:  map[string]string{"cds.app.foo": "value"},
			},
			want: `aa:valueend`,
		},
		{
			name: "default empty value with knowned var",
			args: args{
				input: `aa:{{.cds.app.foo | default ""}}end`,
				vars:  map[string]string{"cds.app.foo": "value"},
			},
			want: `aa:valueend`,
		},
		{
			name: "unknown function",
			args: args{
				input: `echo '{{"conf"|uvault}}'`,
				vars:  map[string]string{},
			},
			want: `echo '{{"conf"|uvault}}'`,
		},
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
			name: "key with - and a unknown key",
			args: args{
				input: "a {{.cds.app.my-key}}.{{.cds.app.foo-key}} and another key value {{.cds.app.my-key}}",
				vars:  map[string]string{"cds.app.my-key": "value-key"},
			},
			want: "a value-key.{{.cds.app.foo-key}} and another key value value-key",
		},
		{
			name: "key with - and a empty key",
			args: args{
				input: "a {{.cds.app.my-key}}.{{.cds.app.foo-key}}.and another key value {{.cds.app.my-key}}",
				vars:  map[string]string{"cds.app.my-key": "value-key", "cds.app.foo-key": ""},
			},
			want: "a value-key..and another key value value-key",
		},
		{
			name: "tiret",
			args: args{
				input: `"METRICS_WRITE_TOKEN": "{{.cds.env.metrics-exposer.write.token}}"`,
				vars:  map[string]string{"cds.env.metrics-exposer.write.token": "valueKey"},
			},
			want: `"METRICS_WRITE_TOKEN": "valueKey"`,
		},
		{
			name: "espace func",
			args: args{
				input: `a {{.cds.foo}} here, {{.cds.title | title}}, {{.cds.upper | upper}}, {{.cds.lower | lower}}, {{.cds.escape | escape}}`,
				vars: map[string]string{
					"cds.foo":    "valbar",
					"cds.title":  "mytitle-bis",
					"cds.upper":  "toupper",
					"cds.lower":  "TOLOWER",
					"cds.escape": "a/b.c_d",
				},
			},
			want: `a valbar here, Mytitle-Bis, TOUPPER, tolower, a-b-c-d`,
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
		{
			name: "same prefix",
			args: args{
				input: `{"HOST": "customer{{.cds.env.lb.prefix}}.{{.cds.env.lb}}"}`,
				vars:  map[string]string{"cds.env.lb": "lb", "cds.env.lb.prefix": "myprefix"},
			},
			want: `{"HOST": "customermyprefix.lb"}`,
		},
		{
			name: "git.branch in payload should not be interpolated",
			args: args{
				input: `
name: "w{{.cds.pip.docker.image}}-generated"
version: v1.0
workflow:
  build-go:
    pipeline: build-go-generated
    payload:
      git.author: ""
      git.branch: master`,
				vars: map[string]string{
					"git.branch": "master",
					"git.author": "",
				},
			},
			want: `
name: "w{{.cds.pip.docker.image}}-generated"
version: v1.0
workflow:
  build-go:
    pipeline: build-go-generated
    payload:
      git.author: ""
      git.branch: master`,
		},
		{
			name: "- inside function parameter",
			args: args{
				input: `name: "coucou-{{ .name | default "0.0.1-dirty" }}"`,
				vars: map[string]string{
					"git.branch": "master",
					"git.author": "",
				},
			},
			want: `name: "coucou-0.0.1-dirty"`,
		},
		{
			name: "- inside function parameter but not used",
			args: args{
				input: `name: "coucou-{{ .name | default "0.0.1-dirty" }}"`,
				vars: map[string]string{
					"git.branch": "master",
					"git.author": "",
					"name":       "toi",
				},
			},
			want: `name: "coucou-toi"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Do(tt.args.input, tt.args.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Do() = %v, want %v", got, tt.want)
			}
		})
	}
}
