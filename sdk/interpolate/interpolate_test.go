package interpolate

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
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
		enable  bool
	}{
		{
			name: "default value with empty default",
			args: args{
				input: `aa:{{.cds.app.foo | default ""}}end`,
				vars:  map[string]string{},
			},
			want:   `aa:end`,
			enable: true,
		},
		{
			name: "default value",
			args: args{
				input: `aa:{{.cds.app.foo | default "bar" }}:end`,
				vars:  map[string]string{},
			},
			want:   `aa:bar:end`,
			enable: true,
		},
		{
			name: "default value with variables (easy)",
			args: args{
				input: `{{.cds.app.foo | default .val }}`,
				vars: map[string]string{
					"val": "biz",
				},
			},
			want:   `biz`,
			enable: true,
		},
		{
			name: "default value with variables (not so easy)",
			args: args{
				input: `{{.cds.app.foo | default .cds.app.bar }}`,
				vars: map[string]string{
					"cds.app.bar": "biz",
				},
			},
			want:   `biz`,
			enable: true,
		},
		{
			name: "default value with variables (so hard)",
			args: args{
				input: `{{.cds.app.foo | default .cds.app.bar .cds.app.biz }}`,
				vars: map[string]string{
					"cds.app.biz": "biz",
				},
			},
			want:   `biz`,
			enable: true,
		},
		{
			name: "default value with variables (with pipeline)",
			args: args{
				input: `{{.cds.app.foo | default .cds.app.bar | default .cds.app.biz | upper }}`,
				vars: map[string]string{
					"cds.app.biz": "biz",
				},
			},
			want:   `BIZ`,
			enable: true,
		},
		{
			name: "default value with pipeline",
			args: args{
				input: `{{.cds.app.foo | upper}}`,
			},
			want:   `{{.cds.app.foo | upper}}`,
			enable: true,
		},
		{
			name: "default value with pipeline",
			args: args{
				input: `{{.cds.app.foo | upper | lower}}`,
			},
			want:   `{{.cds.app.foo | upper | lower}}`,
			enable: true,
		},
		{
			name: "default value with knowned var",
			args: args{
				input: `aa:{{.cds.app.foo | default "bar"}}end`,
				vars:  map[string]string{"cds.app.foo": "value"},
			},
			want:   `aa:valueend`,
			enable: true,
		},
		{
			name: "default empty value with knowned var",
			args: args{
				input: `aa:{{.cds.app.foo | default ""}}end`,
				vars:  map[string]string{"cds.app.foo": "value"},
			},
			want:   `aa:valueend`,
			enable: true,
		},
		{
			name: "unknown function",
			args: args{
				input: `echo '{{"conf"|uvault}}'`,
				vars:  map[string]string{},
			},
			want:   `echo '{{"conf"|uvault}}'`,
			enable: true,
		},
		{
			name: "simple",
			args: args{
				input: "a {{.cds.app.value}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want:   "a value",
			enable: true,
		},
		{
			name: "only unknown",
			args: args{
				input: "a value unknown {{.cds.app.foo}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want:   "a value unknown {{.cds.app.foo}}",
			enable: true,
		},
		{
			name: "simple with unknown",
			args: args{
				input: "a {{.cds.app.value}} and another value unknown {{.cds.app.foo}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want:   "a value and another value unknown {{.cds.app.foo}}",
			enable: true,
		},
		{
			name: "upper",
			args: args{
				input: "a {{.cds.app.value | upper}} and another value unknown {{.cds.app.foo}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want:   "a VALUE and another value unknown {{.cds.app.foo}}",
			enable: true,
		},
		{
			name: "title and filter on unknow",
			args: args{
				input: "a {{.cds.app.value | title }} and another value unknown {{.cds.app.foo | lower}}",
				vars:  map[string]string{"cds.app.value": "value"},
			},
			want:   "a Value and another value unknown {{.cds.app.foo | lower}}",
			enable: true,
		},
		{
			name: "many",
			args: args{
				input: "{{.cds.app.bar}} a {{.cds.app.valuea | upper }}, a {{.cds.app.valueb | title}}.{{.cds.app.valuec}}-{{.cds.app.foo}}",
				vars:  map[string]string{"cds.app.valuea": "valuea", "cds.app.valueb": "valueb", "cds.app.valuec": "valuec"},
			},
			want:   "{{.cds.app.bar}} a VALUEA, a Valueb.valuec-{{.cds.app.foo}}",
			enable: true,
		},
		{
			name: "two same unknown",
			args: args{
				input: `A:{{.cds.env.myenvpassword}} B:{{.cds.env.myenvpassword}}`,
				vars:  map[string]string{},
			},
			want:   `A:{{.cds.env.myenvpassword}} B:{{.cds.env.myenvpassword}}`,
			enable: true,
		},
		{
			name: "two same unknown, but one with a filter",
			args: args{
				input: `A:{{.cds.env.myenvpassword}} B:{{.cds.env.myenvpassword | upper}}`,
				vars:  map[string]string{},
			},
			want:   `A:{{.cds.env.myenvpassword}} B:{{.cds.env.myenvpassword | upper}}`,
			enable: true,
		},
		{
			name: "empty string",
			args: args{
				input: "a {{.cds.app.myKey}} and another key with empty value *{{.cds.app.myKeyAnother}}*",
				vars:  map[string]string{"cds.app.myKey": "valueKey", "cds.app.myKeyAnother": ""},
			},
			want:   "a valueKey and another key with empty value **",
			enable: true,
		},
		{
			name: "two keys with same first characters",
			args: args{
				input: "a {{.cds.app.myKey}} and another key value {{.cds.app.myKeyAnother}}",
				vars:  map[string]string{"cds.app.myKey": "valueKey", "cds.app.myKeyAnother": "valueKeyAnother"},
			},
			want:   "a valueKey and another key value valueKeyAnother",
			enable: true,
		},
		{
			name: "key with - and a unknown key",
			args: args{
				input: "a {{.cds.app.my-key}}.{{.cds.app.foo-key}} and another key value {{.cds.app.my-key}}",
				vars:  map[string]string{"cds.app.my-key": "value-key"},
			},
			want:   "a value-key.{{.cds.app.foo-key}} and another key value value-key",
			enable: true,
		},
		{
			name: "key with - and a empty key",
			args: args{
				input: "a {{.cds.app.my-key}}.{{.cds.app.foo-key}}.and another key value {{.cds.app.my-key}}",
				vars:  map[string]string{"cds.app.my-key": "value-key", "cds.app.foo-key": ""},
			},
			want:   "a value-key..and another key value value-key",
			enable: true,
		},
		{
			name: "tiret",
			args: args{
				input: `"METRICS_WRITE_TOKEN": "{{.cds.env.metrics-exposer.write.token}}"`,
				vars:  map[string]string{"cds.env.metrics-exposer.write.token": "valueKey"},
			},
			want:   `"METRICS_WRITE_TOKEN": "valueKey"`,
			enable: true,
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
			want:   `a valbar here, Mytitle-Bis, TOUPPER, tolower, a-b-c-d`,
			enable: true,
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
			enable: true,
		},
		{
			name: "same prefix",
			args: args{
				input: `{"HOST": "customer{{.cds.env.lb.prefix}}.{{.cds.env.lb}}"}`,
				vars:  map[string]string{"cds.env.lb": "lb", "cds.env.lb.prefix": "myprefix"},
			},
			want:   `{"HOST": "customermyprefix.lb"}`,
			enable: true,
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
			enable: true,
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
			want:   `name: "coucou-0.0.1-dirty"`,
			enable: true,
		},
		{
			name: "- inside function parameter but not used",
			args: args{
				input: `name: "coucou-{{ .name | default "0.0.1-dirty" }}"`,
				vars: map[string]string{
					"name": "toi",
				},
			},
			want:   `name: "coucou-toi"`,
			enable: true,
		},
		{
			name: "- substring",
			args: args{
				input: `name: coucou-{{ .name | substr 0 5 }}`,
				vars: map[string]string{
					"name": "github",
				},
			},
			want:   `name: coucou-githu`,
			enable: true,
		},
		{
			name: "- trunc",
			args: args{
				input: `test_{{.cds.workflow}}_{{.git.hash | trunc 8 }}`,
				vars: map[string]string{
					"cds.workflow":    "myWorkflow",
					"git.hash":        "863ddke13bfef8043960b19cec790f8b9f5435ab",
					"git.hash.before": "863ddke13bfef8043960b19cec790f8b9f5435ab",
				},
			},
			want:   `test_myWorkflow_863ddke1`,
			enable: true,
		},
		{
			name: "add",
			args: args{
				input: "my value {{.cds.app.value | add 3}} {{ add 2 2 }}",
				vars:  map[string]string{"cds.app.value": "1"},
			},
			want:   "my value 4 4",
			enable: true,
		},
		{
			name: "sub",
			args: args{
				input: "my value {{.cds.app.value | sub 1}} {{ sub 5 1 }}",
				vars:  map[string]string{"cds.app.value": "5"},
			},
			want:   "my value 4 4",
			enable: true,
		},
		{
			name: "mul",
			args: args{
				input: "my value {{.cds.app.value | mul 2}} {{ mul 2 2 }}",
				vars:  map[string]string{"cds.app.value": "2"},
			},
			want:   "my value 4 4",
			enable: true,
		},
		{
			name: "div",
			args: args{
				input: "my value {{.cds.app.value | div 2}} {{ div 8 2 }}",
				vars:  map[string]string{"cds.app.value": "8"},
			},
			want:   "my value 4 4",
			enable: true,
		},
		{
			name: "mod",
			args: args{
				input: "my value {{.cds.app.value | mod 6}} {{ mod 10 6 }}",
				vars:  map[string]string{"cds.app.value": "10"},
			},
			want:   "my value 4 4",
			enable: true,
		},
		{
			name: "dirname",
			args: args{
				input: "{{.path | dirname}}",
				vars: map[string]string{
					"path": "/a/b/c",
				},
			},
			want:   "/a/b",
			enable: true,
		},
		{
			name: "basename",
			args: args{
				input: "{{.path | basename}}",
				vars: map[string]string{
					"path": "/ab/c",
				},
			},
			want:   "c",
			enable: true,
		},
		{
			name: "urlencode word",
			args: args{
				input: "{{.query | urlencode}}",
				vars: map[string]string{
					"query": "Trollhättan",
				},
			},
			want:   "Trollh%C3%A4ttan",
			enable: true,
		},
		{
			name: "urlencode query",
			args: args{
				input: "{{.query | urlencode}}",
				vars: map[string]string{
					"query": "zone:eq=Somewhere over the rainbow&name:like=%mydomain.localhost.local",
				},
			},
			want:   "zone%3Aeq%3DSomewhere+over+the+rainbow%26name%3Alike%3D%25mydomain.localhost.local",
			enable: true,
		},
		{
			name: "urlencode nothing to do",
			args: args{
				input: "{{.query | urlencode}}",
				vars: map[string]string{
					"query": "patrick",
				},
			},
			want:   "patrick",
			enable: true,
		},
		{
			name: "ternary truthy",
			args: args{
				input: "{{.assert | ternary .foo .bar}}",
				vars: map[string]string{
					"assert": "true",
					"bar":    "bar",
					"foo":    "foo",
				},
			},
			want:   "foo",
			enable: true,
		},
		{
			name: "ternary truthy integer",
			args: args{
				input: "{{ \"1\" | ternary .foo .bar}}",
				vars: map[string]string{
					"bar": "bar",
					"foo": "foo",
				},
			},
			want:   "foo",
			enable: true,
		},
		{
			name: "ternary falsy",
			args: args{
				input: "{{.assert | ternary .foo .bar}}",
				vars: map[string]string{
					"assert": "false",
					"bar":    "bar",
					"foo":    "foo",
				},
			},
			want:   "bar",
			enable: true,
		},
		{
			name: "ternary undef assert",
			args: args{
				input: "{{.assert | ternary .foo .bar}}",
				vars: map[string]string{
					"bar": "bar",
					"foo": "foo",
				},
			},
			want:   "bar",
			enable: true,
		},
	}
	for _, tt := range tests {
		if !tt.enable {
			continue
		}

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

func TestWrapHelpers(t *testing.T) {
	wrappedHelpers := wrapHelpers(template.FuncMap{
		"substr":  substring,
		"default": dfault,
		"toJSON":  toJSON,
		"trunc":   trunc,
	})

	tests := []struct {
		name  string
		input string
		want  string
		err   error
		data  map[string]interface{}
	}{
		{
			name:  "with native types",
			input: `{{"text" | substr 1 3 }}`,
			want:  `ex`,
		},
		{
			name:  "with nil values",
			input: `{{.one | default .two "biz" }}`,
			want:  `biz`,
		},
		{
			name:  "with unknown struct value",
			input: `{{.one | toJSON }}`,
			want:  `{"name":"myName","age":30}`,
			data: map[string]interface{}{
				"one": struct {
					Name string `json:"name"`
					Age  int    `json:"age"`
				}{
					Name: "myName",
					Age:  30,
				},
			},
		},
		{
			name:  "with a pointer to val struct",
			input: `{{.one.two | trunc 7 }}`,
			want:  `1234567`,
			data: map[string]interface{}{
				"one": &val{
					"two": &val{
						"_": "1234567890",
					},
				},
			},
		},
		{
			name:  "with a val struct",
			input: `{{.one.two | trunc 7 }}`,
			want:  `1234567`,
			data: map[string]interface{}{
				"one": val{
					"two": val{
						"_": "1234567890",
					},
				},
			},
		},
		{
			name:  "with a unknown value",
			input: `{{.one}}`,
			want:  `<no value>`,
		},
		{
			name:  "with missing params",
			input: `{{.one | trunc}}`,
			err:   fmt.Errorf(`template: input:1:9: executing "input" at <trunc>: error calling trunc: missing params (expected: int, string)`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp, err := template.New("input").Funcs(wrappedHelpers).Parse(tt.input)
			assert.NoError(t, err)

			var buff bytes.Buffer
			errexec := tmp.Execute(&buff, tt.data)
			if tt.err == nil {
				assert.NoError(t, errexec)
			} else {
				assert.Error(t, errexec, tt.err)
			}

			if buff.String() != tt.want {
				t.Errorf("Do() = %v, want %v", buff.String(), tt.want)
			}
		})
	}
}

func TestDashReplacementWithµµµ(t *testing.T) {
	vars := map[string]string{
		"result.headers.x-cache": "a",
	}
	got, err := Do("result.headers.x-cache is {{.result.headers.x-cache}}", vars)
	assert.NoError(t, err)
	assert.Equal(t, "result.headers.x-cache is a", got)
}

func TestStringQuote(t *testing.T) {
	vars := map[string]string{
		"content": `{"foo": "{\"bar\":\"baz\"}"}`,
	}
	got, err := Do("content is {{.content | stringQuote}}", vars)
	assert.NoError(t, err)
	assert.Equal(t, `content is {\"foo\": \"{\\\"bar\\\":\\\"baz\\\"}\"}`, got)
}
