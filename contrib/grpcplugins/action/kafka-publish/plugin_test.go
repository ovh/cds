package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func Test_tmplMessage(t *testing.T) {
	type args struct {
		a    *actionplugin.ActionQuery
		buff []byte
	}

	var checkFile = func(str string) string {
		b, err := ioutil.ReadFile(str)
		if err != nil {
			return err.Error()
		}
		return string(b)
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test 1",
			args: args{
				&actionplugin.ActionQuery{
					Options: map[string]string{
						"cds.app.name": "MonApplication",
						"cds.image":    "mon_image:mon_tag/5",
					},
				},
				[]byte(`cds/{{.cds.image}}, /{{.cds.app.name|lower}}`),
			},
			want:    "cds/mon_image:mon_tag/5, /monapplication",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := tmplMessage(tt.args.a, tt.args.buff)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. tmplMessage() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if checkFile(got) != tt.want {
			t.Errorf("%q. tmplMessage() = %v, want %v", tt.name, checkFile(got), tt.want)
		}
		os.RemoveAll(got)
	}
}
