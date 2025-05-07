package glob

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlob_Match(t *testing.T) {
	type testcase struct {
		args      []string
		want      []string
		wantError bool
	}
	tests := []struct {
		name       string
		expression string
		testcase   testcase
	}{

		{"Wildcard (?)",
			"path/to/artifact/foo?.txt", testcase{
				args: []string{
					"path/to/artifact/foo1.txt", "path/to/artifact/foo2.txt", "path/to/artifact/foooo.txt",
				},
				want: []string{
					"foo1.txt", "foo2.txt",
				},
			},
		},
		{"Wildcard (*)",
			"path/to/artifact/foo*.txt", testcase{
				args: []string{
					"path/to/artifact/foo1.txt", "path/to/artifact/foo2.txt", "path/to/artifact/foooo.txt",
				},
				want: []string{
					"foo1.txt", "foo2.txt", "foooo.txt",
				},
			},
		},
		{"Wildcard ([a-z])",
			"path/to/artifact/fo[a-z]?.txt", testcase{
				args: []string{
					"path/to/artifact/foo1.txt", "path/to/artifact/foo2.txt",
				},
				want: []string{
					"foo1.txt", "foo2.txt",
				},
			},
		},
		{"Wildcard (**)",
			"path/**/[abc]rtifac?/*", testcase{
				args: []string{
					"path/to/artifact/foo1.txt", "path/to/artifact/foo2.txt",
				},
				want: []string{
					"to/artifact/foo1.txt", "to/artifact/foo2.txt",
				},
			},
		},
		{"Wildcard (**/*)",
			"**/*", testcase{
				args: []string{
					"path/to/artifact/foo1.txt", "path/to/artifact/foo2.txt",
				},
				want: []string{
					"path/to/artifact/foo1.txt", "path/to/artifact/foo2.txt",
				},
			},
		},
		{"Wildcard (**/*.txt)",
			"path/**/*.txt", testcase{
				args: []string{
					"path/to/artifact", "path/to/artifact/foo1.txt", "path/to/artifact/foo2", "path/to/artifact/foo2.tmp",
				},
				want: []string{
					"to/artifact/foo1.txt",
				},
			},
		},
		{"Wildcard (**/*.txt)",
			"path/**/*.txt", testcase{
				args: []string{
					"path/to/artifact", "path/to/artifact/foo1.txt", "path/to/artifact/foo2", "path/to/artifact/foo2.tmp",
				},
				want: []string{
					"to/artifact/foo1.txt",
				},
			},
		},
		{"Multiple Paths and Exclusions",
			`path/output/bin/*
      path/output/test-results
      !path/**/*.tmp`, testcase{
				args: []string{
					"path/output/bin/foo", "path/output/bin/foo.tmp", "path/output/bin/bar/fizz", "path/output/test-results", "path/to/artifact/foo2.tmp",
				},
				want: []string{
					"foo", "test-results",
				},
			},
		},
		{
			"With colon",
			`docker:path/to/image:* helm:**/*`, testcase{
				args: []string{
					"docker:path/to/image:latest", "helm:chart",
				},
				want: []string{
					"image:latest",
					"helm:chart",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(tt.expression)
			results, err := g.Match(tt.testcase.args...)
			if tt.testcase.wantError {
				require.Error(t, err)
			}
			t.Log("results: ", results)
			for _, expectedResult := range tt.testcase.want {
				var found bool
				for _, actualResult := range results {
					if actualResult.Result == expectedResult {
						found = true
					}
				}
				require.True(t, found, "result %q not found in results", expectedResult)
			}

			for _, actualResult := range results {
				var found bool
				for _, expectedResult := range tt.testcase.want {
					if actualResult.Result == expectedResult {
						found = true
					}
				}
				require.True(t, found, "result %q returned but not expected", actualResult)
			}
		})
	}
}

func TestGlob_MatchFiles(t *testing.T) {
	pattern := "path/to/**/* !path/to/**/*.tmp"
	g := New(pattern)
	result, err := g.MatchFiles(os.DirFS("tests/fixtures"))
	require.NoError(t, err)
	t.Logf("%s matches %s", pattern, result.String())
}

func TestGlob_Relative(t *testing.T) {
	pattern := "./path/to/artifacts/bar"
	cwd := fmt.Sprintf("%s", os.DirFS("tests/fixtures"))
	result, err := Glob(cwd, pattern)
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Results))
}
