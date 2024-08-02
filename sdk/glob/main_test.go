package glob_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ovh/cds/sdk/glob"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func TestGlob(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	pattern := "path/to/**/* !path/to/**/*.tmp"
	result, err := glob.Glob("tests/fixtures", pattern)
	require.NoError(t, err)
	require.Equal(t, "path/to/artifacts/bar, path/to/artifacts/foo, path/to/results/foo.bin", result.String())
}

func TestGlobAbsolute(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	_, filename, _, _ := runtime.Caller(0)
	t.Logf("Current test filename: %s", filename)

	path := filepath.Dir(filename)
	fixturePath := filepath.Join(path, "tests", "fixtures")

	pattern := fmt.Sprintf("%s/path/to/**/* !%s/path/to/**/*.tmp", fixturePath, fixturePath)
	result, err := glob.Glob(fixturePath, pattern)
	require.NoError(t, err)
	require.Equal(t, "artifacts/bar, artifacts/foo, results/foo.bin", result.String())
}

func TestGlobWitDoubleStar0(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "bar"
	pattern := "**/*"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, file, r.String())
	require.Equal(t, "bar", r.Result)
}

func TestGlobWitDoubleStar1(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "path/to/artifacts/bar"
	pattern := "path/to/**/*"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, file, r.String())
	require.Equal(t, "artifacts/bar", r.Result)
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

func TestGlobWithDoubleStar(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "path/to/artifact/foooo.txt"
	pattern := "path/to/**/foo*.txt"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, "artifact/foooo.txt", r.Result)
}

func TestGlobWithDoubleStar2(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	file := "path/to/artifact/a/foooo.txt"
	pattern := "path/to/**/foo*.txt"
	g := glob.New(pattern)
	r, e := g.MatchString(file)
	require.NoError(t, e)
	require.NotNil(t, r)
	require.Equal(t, "artifact/a/foooo.txt", r.Result)
}

func TestGlobDoubleStarMatchString0(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("**/newfile")
	result, err := g.MatchString("root/newfile")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "root/newfile", result.String())
}

func TestGlobDoubleStarMatchString(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("**/newfile")
	result, err := g.MatchString("root/app/sub/newfile")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "root/app/sub/newfile", result.String())
}

func TestGlobDoubleStarMatchString2(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("**/cd/rtgc")
	result, err := g.MatchString("root/app/sub/cd/rtgc")
	t.Logf("err= %v", err)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "root/app/sub/cd/rtgc", result.String())
	t.Logf("result= %s", result.String())

}

func TestGlobDoubleStarMatchString3(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("**/rtgc")
	result, err := g.MatchString("root/app/sub/cd/rtgc")
	t.Logf("result= %s", result.String())
	t.Logf("err= %v", err)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "root/app/sub/cd/rtgc", result.String())
}

func TestGlobDoubleStarMatchString4(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("root/**/rtgc")
	result, err := g.MatchString("root/app/sub/cd/rtgc")
	t.Logf("result= %s", result.String())
	t.Logf("err= %v", err)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "root/app/sub/cd/rtgc", result.String())
}

func TestGlobDoubleStarMatchString5(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("**/cd/rtgc")
	result, err := g.MatchString("root/app/sub/cc/cd/rtgc")
	t.Logf("err= %v", err)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "root/app/sub/cc/cd/rtgc", result.String())
	t.Logf("result= %s", result.String())

}

func TestGlobDoubleStarMatchString6(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("path/**/[abc]rtifac?/*")
	result, err := g.MatchString("path/to/artifact/foo")
	t.Logf("err= %v", err)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "path/to/artifact/foo", result.String())
	t.Logf("result= %s", result.String())
}

func TestGlobDoubleStarMatchString7(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("path/**/[abc]rtifac?/*")
	result, err := g.MatchString("path/to/my/artifact/foo")
	t.Logf("err= %v", err)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "path/to/my/artifact/foo", result.String())
	t.Logf("result= %s", result.String())
}

func TestSquareBracket(t *testing.T) {
	glob.DebugEnabled = true
	glob.DebugFunc = func(a ...any) (n int, err error) {
		t.Log(a...)
		return len(a), nil
	}
	log.Factory = log.NewTestingWrapper(t)

	g := glob.New("[abc]rtifac?/*")
	result, err := g.MatchString("artifact/foo")
	t.Logf("err= %v", err)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "artifact/foo", result.String())
	t.Logf("result= %s", result.String())
}
