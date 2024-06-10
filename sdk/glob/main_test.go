package glob_test

import (
	"os"
	"testing"

	"github.com/ovh/cds/sdk/glob"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func TestGlob(t *testing.T) {
	pattern := "path/**/* !path/**/*.tmp"
	result, err := glob.Glob(os.DirFS("tests/"), "fixtures", pattern)
	require.NoError(t, err)
	require.Equal(t, "path/to/artifacts/bar, path/to/artifacts/foo, path/to/results/foo.bin", result.String())
}

func TestGlobDoubleStarMatchString(t *testing.T) {
	g := glob.New("**/newfile")
	result, err := g.MatchString("root/app/sub/newfile")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "root/app/sub/newfile", result.String())
}

func TestGlobDoubleStarMatchString2(t *testing.T) {
	g := glob.New("**/cd/rtgc")
	result, err := g.MatchString("root/app/sub/cd/rtgc")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "root/app/sub/cd/rtgc", result.String())
}

func TestGlobWithDot(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "k3s_1.29.2+k3s1-1~15bionic~d3c5923a48d07c247ffd80251f27c2fc7cfef226_all.deb"
	pattern := "*.deb"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, file, r.Result)
}

func TestGlobWithStar1(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "a.b.c"
	pattern := "*"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, file, r.Result)
}

func TestGlobWithStar2(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "a.b.c"
	pattern := "*.*"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, file, r.Result)
}

func TestGlobWithStar3(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "a.b.c"
	pattern := "*.*.*"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, file, r.Result)
}

func TestGlobWithStar4(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "a.b.c"
	pattern := "*."
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.Nil(t, r)
}

func TestGlobWithStar5(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "a.b.c"
	pattern := "*.c"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, file, r.Result)
}

func TestGlobWithStar6(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "path/to/artifact/foooo.txt"
	pattern := "path/to/artifact/foo*.txt"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, "foooo.txt", r.Result)
}
