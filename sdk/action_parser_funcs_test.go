package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func Test_result_as_annotation_expression(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	// Usage as annotations expression
	a := ActionParser{
		contexts: map[string]interface{}{
			"jobs": map[string]interface{}{
				"myJob": map[string]interface{}{
					"results": map[string]interface{}{
						"JobRunResults": map[string]interface{}{
							"generic:foo.txt": V2WorkflowRunResultGenericDetail{},
						},
					},
				},
			},
		},
	}

	r, err := result(context.TODO(), &a, "generic", "foo.*")
	require.NoError(t, err)
	require.NotNil(t, r)
}

func Test_result_as_script_expression(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	// Usage as expression in script
	a := ActionParser{
		contexts: map[string]interface{}{
			"jobs": map[string]interface{}{
				"myJob": map[string]interface{}{
					"JobRunResults": map[string]interface{}{
						"generic:foo.txt": V2WorkflowRunResultGenericDetail{},
					},
				},
			},
		},
	}

	r, err := result(context.TODO(), &a, "generic", "foo.*")
	require.NoError(t, err)
	require.NotNil(t, r)
}

func Test_result_as_script_expression_multiple(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	// Usage as expression in script
	a := ActionParser{
		contexts: map[string]interface{}{
			"jobs": map[string]interface{}{
				"myJob": map[string]interface{}{
					"JobRunResults": map[string]interface{}{
						"generic:foo.txt": V2WorkflowRunResultGenericDetail{
							Name: "foo.txt",
						},
						"generic:foo.zip": V2WorkflowRunResultGenericDetail{
							Name: "foo.zip",
						},
					},
				},
			},
		},
	}

	r, err := result(context.TODO(), &a, "generic", "foo.*")
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Len(t, r, 2)
	t.Logf("==> %+v", r)
}

func Test_toArray(t *testing.T) {
	x, _ := toArray(nil, nil, "foo")
	t.Logf("%T %+v", x, x)
	require.Equal(t, reflect.Slice.String(), reflect.ValueOf(x).Kind().String())

	x, _ = toArray(nil, nil, []string{"foo"})
	t.Logf("%T %+v", x, x)
	require.Equal(t, reflect.Slice.String(), reflect.ValueOf(x).Kind().String())

	x, _ = toArray(nil, nil, []string{"foo", "bar"})
	t.Logf("%T %+v", x, x)
	require.Equal(t, reflect.Slice.String(), reflect.ValueOf(x).Kind().String())

	x, _ = toArray(nil, nil, "foo", "bar")
	t.Logf("%T %+v", x, x)
	require.Equal(t, reflect.Slice.String(), reflect.ValueOf(x).Kind().String())

	x, _ = toArray(nil, nil, map[string]string{"foo": "bar"})
	t.Logf("%T %+v", x, x)
	require.Equal(t, reflect.Slice.String(), reflect.ValueOf(x).Kind().String())
}

func TestHashFiles(t *testing.T) {
	path := filepath.Join(os.TempDir(), "testdata", t.Name())
	defer os.RemoveAll(path)
	require.NoError(t, os.MkdirAll(path, os.FileMode(0755)))

	log.Factory = log.NewTestingWrapper(t)
	a := ActionParser{
		contexts: map[string]interface{}{
			"cds": map[string]interface{}{
				"workspace": "/home/sguiheux/src/github.com/ovh/cds/sdk",
			},
		},
	}

	d1 := []byte("I'm file 1")
	err := os.WriteFile(path+"/file1", d1, 0755)
	require.NoError(t, err)

	d2 := []byte("I'm file 2")
	err = os.WriteFile(path+"/file2", d2, 0755)
	require.NoError(t, err)

	hashSum1, err := hashFiles(context.TODO(), &a, path+"/file1", path+"/file2")
	require.NoError(t, err)

	hashSum2, err := hashFiles(context.TODO(), &a, path+"/file2", path+"/file1")
	require.NoError(t, err)

	t.Logf("%s", hashSum1)
	t.Logf("%s", hashSum2)

	require.Equal(t, fmt.Sprintf("%s", hashSum1), fmt.Sprintf("%s", hashSum2))
}

func Test_newStringActionFunc(t *testing.T) {
	for _, tt := range []struct {
		name   string
		arg    string
		output string
	}{
		{
			"toLower",
			"FoOBaR",
			"foobar",
		},
		{
			"toUpper",
			"fooBaR",
			"FOOBAR",
		},
		{
			"toTitle",
			"хлеб",
			"ХЛЕБ",
		},
		{
			"title",
			"foo bar",
			"Foo Bar",
		},
		{
			"b64enc",
			"foo bar baz",
			"Zm9vIGJhciBiYXo=",
		},
		{
			"b64dec",
			"Zm9vIGJhciBiYXo=",
			"foo bar baz",
		},
		{
			"b32enc",
			"foo bar baz",
			"MZXW6IDCMFZCAYTBPI======",
		},
		{
			"b32dec",
			"MZXW6IDCMFZCAYTBPI======",
			"foo bar baz",
		},
	} {
		fn, ok := DefaultFuncs[tt.name]
		if !ok {
			t.Errorf("func %s not found", tt.name)
		}
		t.Run(tt.name, func(t *testing.T) {
			v, err := fn(context.TODO(), nil, tt.arg)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := v.(string)
			if !ok {
				t.Fatalf("expected string, got %T", v)
			}
			if tt.output != got {
				t.Errorf("got %q, want %q", got, tt.output)
			}
			t.Logf(got)
		})
	}
}

func Test_newStringStringActionFunc(t *testing.T) {
	for _, tt := range []struct {
		name   string
		argA   string
		argB   string
		output string
	}{
		{
			"trimAll",
			"$",
			"$foobar$",
			"foobar",
		},
		{
			"trimPrefix",
			"v",
			"v6.6.6-evil",
			"6.6.6-evil",
		},
		{
			"trimSuffix",
			".deb",
			"myFile.deb",
			"myFile",
		},
	} {
		fn, ok := DefaultFuncs[tt.name]
		if !ok {
			t.Errorf("func %s not found", tt.name)
		}
		t.Run(tt.name, func(t *testing.T) {
			v, err := fn(context.TODO(), nil, tt.argA, tt.argB)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := v.(string)
			if !ok {
				t.Fatalf("expected string, got %T", v)
			}
			if tt.output != got {
				t.Errorf("got %q, want %q", got, tt.output)
			}
			t.Logf(got)
		})
	}
}
