package sdk

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringIsAscii(t *testing.T) {
	assert.True(t, StringIsAscii("aaa"))
	assert.False(t, StringIsAscii("aaa 🚀"))
}

func TestRemoveNotPrintableChar(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{
			in:  "test",
			out: "test",
		},
		{
			in:  "test" + string([]byte{0x00}),
			out: "test ",
		},
		{
			in:  "test" + string([]byte{0xbf}),
			out: "test ",
		},
	}

	for i := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tests[i].out, RemoveNotPrintableChar(tests[i].in))
		})
	}
}

func TestPathIsAbs(t *testing.T) {
	GOOS = "windows"
	assert.True(t, PathIsAbs(`C:\Program Files (x86)\Foo`))
	assert.False(t, PathIsAbs(`Program Files (x86)\Foo`))
	GOOS = "linux"
	assert.True(t, PathIsAbs(`/tmp`))
	assert.False(t, PathIsAbs(`tmp`))
}

func TestCleanPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test1",
			args: args{
				path: "../../foobar",
			},
			want: "foobar",
		},
		{
			name: "test1",
			args: args{
				path: "./foobar",
			},
			want: "foobar",
		},
		{
			name: "test2",
			args: args{
				path: "/foo/bar/biz",
			},
			want: "/foo/bar/biz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanPath(tt.args.path); got != tt.want {
				t.Errorf("CleanPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test1",
			args: args{
				path: "/foo/bar/biz",
			},
			want: "biz",
		},
		{
			name: "test2",
			args: args{
				path: "",
			},
			want: "",
		},
		{
			name: "test3",
			args: args{
				path: ".",
			},
			want: ".",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NoPath(tt.args.path); got != tt.want {
				t.Errorf("NoPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
