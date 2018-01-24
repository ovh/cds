package zglob

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func check(got []string, expected []string) bool {
	sort.Strings(got)
	sort.Strings(expected)
	return reflect.DeepEqual(expected, got)
}

type testZGlob struct {
	pattern  string
	expected []string
	err      error
}

var testGlobs = []testZGlob{
	{`fo*`, []string{`foo`}, nil},
	{`foo`, []string{`foo`}, nil},
	{`foo/*`, []string{`foo/bar`, `foo/baz`}, nil},
	{`foo/**`, []string{`foo/bar`, `foo/baz`}, nil},
	{`f*o/**`, []string{`foo/bar`, `foo/baz`}, nil},
	{`*oo/**`, []string{`foo/bar`, `foo/baz`, `hoo/bar`}, nil},
	{`*oo/b*`, []string{`foo/bar`, `foo/baz`, `hoo/bar`}, nil},
	{`*oo/bar`, []string{`foo/bar`, `hoo/bar`}, nil},
	{`*oo/*z`, []string{`foo/baz`}, nil},
	{`foo/**/*`, []string{`foo/bar`, `foo/bar/baz`, `foo/bar/baz.txt`, `foo/bar/baz/noo.txt`, `foo/baz`}, nil},
	{`*oo/**/*`, []string{`foo/bar`, `foo/bar/baz`, `foo/bar/baz.txt`, `foo/bar/baz/noo.txt`, `foo/baz`, `hoo/bar`}, nil},
	{`*oo/*.txt`, []string{}, nil},
	{`*oo/*/*.txt`, []string{`foo/bar/baz.txt`}, nil},
	{`*oo/**/*.txt`, []string{`foo/bar/baz.txt`, `foo/bar/baz/noo.txt`}, nil},
	{`doo`, nil, os.ErrNotExist},
	{`./f*`, []string{`foo`}, nil},
	{`**/bar/**/*.txt`, []string{`foo/bar/baz.txt`, `foo/bar/baz/noo.txt`}, nil},
}

func setup(t *testing.T) string {
	tmpdir, err := ioutil.TempDir("", "zglob")
	if err != nil {
		t.Fatal(err)
	}

	os.MkdirAll(filepath.Join(tmpdir, "foo/baz"), 0755)
	os.MkdirAll(filepath.Join(tmpdir, "foo/bar"), 0755)
	ioutil.WriteFile(filepath.Join(tmpdir, "foo/bar/baz.txt"), []byte{}, 0644)
	os.MkdirAll(filepath.Join(tmpdir, "foo/bar/baz"), 0755)
	ioutil.WriteFile(filepath.Join(tmpdir, "foo/bar/baz/noo.txt"), []byte{}, 0644)
	os.MkdirAll(filepath.Join(tmpdir, "hoo/bar"), 0755)
	ioutil.WriteFile(filepath.Join(tmpdir, "foo/bar/baz.txt"), []byte{}, 0644)

	return tmpdir
}

func TestGlob(t *testing.T) {
	tmpdir := setup(t)
	defer os.RemoveAll(tmpdir)

	curdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(curdir)

	tmpdir = "."
	for _, test := range testGlobs {
		expected := make([]string, len(test.expected))
		for i, e := range test.expected {
			expected[i] = e
		}
		got, err := Glob(test.pattern)
		if err != nil {
			if test.err != err {
				t.Error(err)
			}
			continue
		}
		if !check(expected, got) {
			t.Errorf(`zglob failed: pattern %q(%q): expected %v but got %v`, test.pattern, tmpdir, expected, got)
		}
	}
}

func TestGlobAbs(t *testing.T) {
	tmpdir := setup(t)
	defer os.RemoveAll(tmpdir)

	curdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(curdir)

	for _, test := range testGlobs {
		pattern := filepath.ToSlash(filepath.Join(tmpdir, test.pattern))
		expected := make([]string, len(test.expected))
		for i, e := range test.expected {
			expected[i] = filepath.ToSlash(filepath.Join(tmpdir, e))
		}
		got, err := Glob(pattern)
		if err != nil {
			if test.err != err {
				t.Error(err)
			}
			continue
		}
		if !check(expected, got) {
			t.Errorf(`zglob failed: pattern %q(%q): expected %v but got %v`, pattern, tmpdir, expected, got)
		}
	}
}

func TestMatch(t *testing.T) {
	for _, test := range testGlobs {
		for _, f := range test.expected {
			got, err := Match(test.pattern, f)
			if err != nil {
				t.Error(err)
				continue
			}
			if !got {
				t.Errorf("%q should match with %q", f, test.pattern)
			}
		}
	}
}

func TestFollowSymlinks(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "zglob")
	if err != nil {
		t.Fatal(err)
	}

	os.MkdirAll(filepath.Join(tmpdir, "foo"), 0755)
	ioutil.WriteFile(filepath.Join(tmpdir, "foo/baz.txt"), []byte{}, 0644)
	defer os.RemoveAll(tmpdir)

	err = os.Symlink(filepath.Join(tmpdir, "foo"), filepath.Join(tmpdir, "bar"))
	if err != nil {
		t.Skip(err.Error())
	}

	curdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(curdir)

	got, err := GlobFollowSymlinks("**/*")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"foo", "foo/baz.txt", "bar/baz.txt"}

	if !check(expected, got) {
		t.Errorf(`zglob failed: expected %v but got %v`, expected, got)
	}
}
