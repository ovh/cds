package glob

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGlob_PatternCleaningNormalizesToSlash verifies that filepath.Clean followed
// by filepath.ToSlash produces forward-slash only patterns, which is critical on
// Windows where filepath.Clean produces backslashes.
func TestGlob_PatternCleaningNormalizesToSlash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple relative", "dist/foo.zip", "dist/foo.zip"},
		{"dotslash prefix", "./dist/foo.zip", "dist/foo.zip"},
		{"double dotslash", "a/./b/../c/foo.zip", "a/c/foo.zip"},
		{"trailing slash", "dist/", "dist"},
		{"double slash", "dist//foo.zip", "dist/foo.zip"},
		{"star pattern", "dist/*.zip", "dist/*.zip"},
		{"doublestar pattern", "path/to/**/*.txt", "path/to/**/*.txt"},
		{"pattern with dot segments", "./path/../path/to/*.txt", "path/to/*.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filepath.ToSlash(filepath.Clean(tt.input))
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestGlob_BackslashPatternNormalization verifies that patterns containing
// backslashes (as produced by filepath.Clean on Windows) are normalized to
// forward slashes before matching against io/fs paths.
// On Linux, backslash is a valid filename character and filepath.ToSlash is a
// no-op, so we use strings.ReplaceAll to simulate the Windows behavior.
func TestGlob_BackslashPatternNormalization(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		cleaned string
	}{
		{"backslash in pattern", `dist\*.zip`, "dist/*.zip"},
		{"multiple backslashes", `path\to\artifact\*.zip`, "path/to/artifact/*.zip"},
		{"mixed separators", `path/to\artifact/*.zip`, "path/to/artifact/*.zip"},
		{"backslash with doublestar", `path\**\*.txt`, "path/**/*.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// strings.ReplaceAll simulates what filepath.ToSlash does on Windows
			// (replacing os.PathSeparator '\' with '/')
			result := strings.ReplaceAll(tt.input, `\`, "/")
			require.Equal(t, tt.cleaned, result)
		})
	}
}

// TestGlob_MatchWithNormalizedPattern verifies that patterns that went through
// the normalization pipeline can still match files from io/fs (which always uses
// forward slashes). We use forward-slash patterns directly since that is what
// the Glob function produces after filepath.ToSlash on Windows.
func TestGlob_MatchWithNormalizedPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		files   []string
		want    []string
	}{
		{
			"star after normalized backslash",
			"dist/*.zip", // what filepath.ToSlash("dist\*.zip") produces on Windows
			[]string{"dist/foo.zip", "dist/bar.zip", "dist/readme.txt"},
			[]string{"foo.zip", "bar.zip"},
		},
		{
			"doublestar after normalized backslash",
			"path/to/**/*.txt", // what filepath.ToSlash("path\to\**\*.txt") produces on Windows
			[]string{"path/to/a/file.txt", "path/to/b/c/deep.txt", "path/to/nope.bin"},
			[]string{"a/file.txt", "b/c/deep.txt"},
		},
		{
			"question mark after normalized backslash",
			"dist/foo?.zip", // what filepath.ToSlash("dist\foo?.zip") produces on Windows
			[]string{"dist/foo1.zip", "dist/foo2.zip", "dist/foooo.zip"},
			[]string{"foo1.zip", "foo2.zip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(tt.pattern)
			results, err := g.Match(tt.files...)
			require.NoError(t, err)
			require.Equal(t, len(tt.want), len(results), "unexpected number of results")
			for _, expected := range tt.want {
				var found bool
				for _, r := range results {
					if r.Result == expected {
						found = true
						break
					}
				}
				require.True(t, found, "expected result %q not found", expected)
			}
		})
	}
}

// TestGlob_GlobFunctionWithRelativePatternContainingBackslash tests the full
// Glob() function with patterns that could come from Windows (backslashes).
// On Linux, this simulates the scenario since io/fs always uses forward slashes.
func TestGlob_GlobFunctionWithRelativePatternContainingBackslash(t *testing.T) {
	// The fixtures have: path/to/artifacts/bar, path/to/artifacts/foo, path/to/results/foo.bin, path/to/results/foo.tmp
	cwd := "tests/fixtures"

	tests := []struct {
		name        string
		pattern     string
		wantCount   int
		wantResults []string
	}{
		{
			"backslash in relative pattern - star",
			`path/to/artifacts/*`, // forward slashes (normal)
			2,
			[]string{"path/to/artifacts/bar", "path/to/artifacts/foo"},
		},
		{
			"dotslash prefix cleaned",
			`./path/to/artifacts/*`,
			2,
			[]string{"path/to/artifacts/bar", "path/to/artifacts/foo"},
		},
		{
			"pattern with unnecessary dot segments",
			`./path/to/../to/artifacts/*`,
			2,
			[]string{"path/to/artifacts/bar", "path/to/artifacts/foo"},
		},
		{
			"doublestar pattern",
			`path/to/**/*`,
			4,
			[]string{"path/to/artifacts/bar", "path/to/artifacts/foo", "path/to/results/foo.bin", "path/to/results/foo.tmp"},
		},
		{
			"doublestar with exclusion",
			"path/to/**/* !path/to/**/*.tmp",
			3,
			[]string{"path/to/artifacts/bar", "path/to/artifacts/foo", "path/to/results/foo.bin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Glob(cwd, tt.pattern)
			require.NoError(t, err)
			require.Equal(t, tt.wantCount, len(result.Results), "for pattern %q, got results: %s", tt.pattern, result.String())
			for _, expected := range tt.wantResults {
				var found bool
				for _, r := range result.Results {
					if r.Path == expected {
						found = true
						break
					}
				}
				require.True(t, found, "expected path %q not found in results: %s", expected, result.String())
			}
		})
	}
}

// TestGlob_GlobFunctionWithAbsolutePattern tests the Glob() function with
// absolute paths (the other branch in the Glob function).
func TestGlob_GlobFunctionWithAbsolutePattern(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Dir(filename)
	fixturePath := filepath.Join(path, "tests", "fixtures")

	// Build absolute pattern using forward slashes (as io/fs expects)
	absPattern := filepath.ToSlash(fixturePath) + "/path/to/artifacts/*"
	result, err := Glob(fixturePath, absPattern)
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Results), "got: %s", result.String())
}

// TestGlob_GlobFunctionWithAbsolutePatternAndExclusion tests absolute path
// patterns with exclusions.
func TestGlob_GlobFunctionWithAbsolutePatternAndExclusion(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Dir(filename)
	fixturePath := filepath.Join(path, "tests", "fixtures")

	absBase := filepath.ToSlash(fixturePath)
	absPattern := absBase + "/path/to/**/* !" + absBase + "/path/to/**/*.tmp"
	result, err := Glob(fixturePath, absPattern)
	require.NoError(t, err)
	require.Equal(t, 3, len(result.Results), "got: %s", result.String())
}

// TestLongestCommonPathPrefix_ForwardSlash tests the base case with forward
// slashes only.
func TestLongestCommonPathPrefix_ForwardSlash(t *testing.T) {
	// This test uses real paths that exist on disk
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Dir(filename)
	fixturePath := filepath.Join(path, "tests", "fixtures", "path", "to")

	strs := []string{
		fixturePath + "/artifacts/foo",
		fixturePath + "/artifacts/bar",
	}

	// Normalize to forward slashes for the test
	for i := range strs {
		strs[i] = filepath.ToSlash(strs[i])
	}

	result := LongestCommonPathPrefix(strs)
	// Should return the path up to the last valid directory separator
	require.NotEmpty(t, result)
	require.True(t, len(result) > 0)
	// The result should end with a /
	require.Equal(t, "/", string(result[len(result)-1]))
}

// TestLongestCommonPathPrefix_EmptyInput tests edge case with empty input.
func TestLongestCommonPathPrefix_EmptyInput(t *testing.T) {
	result := LongestCommonPathPrefix([]string{})
	require.Equal(t, "", result)
}

// TestLongestCommonPathPrefix_SinglePath tests with a single path.
func TestLongestCommonPathPrefix_SinglePath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Dir(filename)
	fixturePath := filepath.Join(path, "tests", "fixtures", "path", "to")

	p := filepath.ToSlash(fixturePath) + "/artifacts/foo"
	result := LongestCommonPathPrefix([]string{p})
	// With a single path, the prefix is everything up to the last slash that is a valid dir
	require.NotEmpty(t, result)
}

// TestLongestCommonPathPrefix_NoCommonPrefix tests paths with no common
// directory prefix.
func TestLongestCommonPathPrefix_NoCommonPrefix(t *testing.T) {
	// These paths diverge immediately after "/" which exists on Linux.
	// The common prefix "/" is valid, so we use paths that don't start with
	// a valid directory at all.
	strs := []string{
		"abc_nonexistent/foo/bar",
		"xyz_nonexistent/baz/qux",
	}
	result := LongestCommonPathPrefix(strs)
	// No common directory exists on disk, result should be empty
	require.Equal(t, "", result)
}

// TestGlob_SplitListExpression tests the various separators for list expressions.
func TestGlob_SplitListExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"newline separated", "a\nb", []string{"a", "b"}},
		{"space separated", "a b", []string{"a", "b"}},
		{"comma separated", "a,b", []string{"a", "b"}},
		{"single expression", "a/b/c", []string{"a/b/c"}},
		{"newline with multiple", "*.zip\n*.tar.gz\n!*.tmp", []string{"*.zip", "*.tar.gz", "!*.tmp"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitListExpression(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestGlob_IsExcludeExpression tests the exclusion detection.
func TestGlob_IsExcludeExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"!path/to/exclude", true},
		{"path/to/include", false},
		{"path/with!bang", true}, // any ! in the string
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.expected, isExcludeExpression(tt.input))
		})
	}
}

// TestGlob_MixedAbsoluteAndRelativePatterns tests that mixing absolute and
// relative patterns returns an error.
func TestGlob_MixedAbsoluteAndRelativePatterns(t *testing.T) {
	var pattern string
	if runtime.GOOS == "windows" {
		pattern = `C:\Users\test\file.txt relative/path.txt`
	} else {
		pattern = "/absolute/path.txt relative/path.txt"
	}
	_, err := Glob(".", pattern)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mixing absolute and relative patterns is not supported")
}

// TestGlob_PatternCleanRemovesDotSegments verifies that the Glob function
// correctly handles dot segments in patterns via filepath.Clean.
func TestGlob_PatternCleanRemovesDotSegments(t *testing.T) {
	// "./path/to/../to/artifacts/*" should clean to "path/to/artifacts/*"
	cwd := "tests/fixtures"
	result, err := Glob(cwd, "./path/to/../to/artifacts/*")
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Results), "got: %s", result.String())
}

// TestGlob_EmptyPattern tests behavior with an empty pattern (matches current dir).
func TestGlob_EmptyPattern(t *testing.T) {
	cwd := "tests/fixtures"
	result, err := Glob(cwd, ".")
	require.NoError(t, err)
	// "." after clean is "." which is not a file, so no results expected
	require.Equal(t, 0, len(result.Results))
}

// TestGlob_SimulateWindowsUploadArtifact simulates the exact scenario from the
// bug report: a user on Windows runs uploadArtifact with pattern "dist/*.zip"
// which after filepath.Clean becomes "dist\*.zip" on Windows. Our fix with
// filepath.ToSlash converts it back to "dist/*.zip" so it matches correctly.
func TestGlob_SimulateWindowsUploadArtifact(t *testing.T) {
	// Create a temp directory structure simulating the Windows scenario
	tmpDir := t.TempDir()

	// Create dist/OmisimO-win32-x64.zip
	distDir := filepath.Join(tmpDir, "dist")
	require.NoError(t, os.MkdirAll(distDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "OmisimO-win32-x64.zip"), []byte("fake zip"), 0644))

	// The user's original pattern: "dist/*.zip"
	// On Windows, filepath.Clean("dist/*.zip") would produce "dist\*.zip"
	// Our fix: filepath.ToSlash(filepath.Clean("dist/*.zip"))
	pattern := "dist/*.zip"

	result, err := Glob(tmpDir, pattern)
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Results), "expected 1 result, got: %s", result.String())
	require.Equal(t, "dist/OmisimO-win32-x64.zip", result.Results[0].Path)
}

// TestGlob_SimulateWindowsAbsolutePath simulates the scenario where a Windows
// absolute path is used as a pattern, like
// "C:\Users\...\run\dist\*.zip"
// This verifies that our ToSlash normalization works in the absolute path branch too.
func TestGlob_SimulateWindowsAbsolutePath(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()

	distDir := filepath.Join(tmpDir, "run", "dist")
	require.NoError(t, os.MkdirAll(distDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "app.zip"), []byte("fake"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "readme.txt"), []byte("fake"), 0644))

	// Use an absolute pattern with forward slashes (simulating ToSlash on Windows path)
	absPattern := filepath.ToSlash(distDir) + "/*.zip"
	result, err := Glob(tmpDir, absPattern)
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Results), "expected 1 result, got: %s", result.String())
}

// TestGlob_MultipleFilesWithStar tests that * correctly matches multiple files
// in a single directory.
func TestGlob_MultipleFilesWithStar(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "output")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.zip"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.zip"), []byte("b"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.tar.gz"), []byte("c"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "d.zip"), []byte("d"), 0644))

	result, err := Glob(tmpDir, "output/*.zip")
	require.NoError(t, err)
	require.Equal(t, 3, len(result.Results), "got: %s", result.String())
}

// TestGlob_DeepRelativePattern tests patterns with deep relative paths.
func TestGlob_DeepRelativePattern(t *testing.T) {
	tmpDir := t.TempDir()

	deep := filepath.Join(tmpDir, "a", "b", "c", "d")
	require.NoError(t, os.MkdirAll(deep, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "file.txt"), []byte("x"), 0644))

	result, err := Glob(tmpDir, "a/b/c/d/*.txt")
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Results))
	require.Equal(t, "a/b/c/d/file.txt", result.Results[0].Path)
}

// TestGlob_DoubleStarWithExtensionFilter tests ** combined with extension filter
// across multiple directory levels.
func TestGlob_DoubleStarWithExtensionFilter(t *testing.T) {
	tmpDir := t.TempDir()

	dirs := []string{
		filepath.Join(tmpDir, "src"),
		filepath.Join(tmpDir, "src", "pkg"),
		filepath.Join(tmpDir, "src", "pkg", "sub"),
	}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(d, 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dirs[0], "main.go"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dirs[1], "util.go"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dirs[2], "helper.go"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dirs[2], "helper_test.go"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dirs[0], "readme.md"), []byte("x"), 0644))

	result, err := Glob(tmpDir, "src/**/*.go")
	require.NoError(t, err)
	require.Equal(t, 4, len(result.Results), "got: %s", result.String())
}

// TestGlob_ExclusionPattern tests that exclusion patterns work correctly after
// path normalization.
func TestGlob_ExclusionPattern(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "build")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.zip"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "debug.zip"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.log"), []byte("x"), 0644))

	result, err := Glob(tmpDir, "build/* !build/*.log")
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Results), "got: %s", result.String())
	for _, r := range result.Results {
		require.NotContains(t, r.Path, ".log")
	}
}

// TestGlob_NoMatch tests that a pattern that doesn't match any file returns
// empty results without error.
func TestGlob_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(dir, 0755))

	result, err := Glob(tmpDir, "empty/*.zip")
	require.NoError(t, err)
	require.Equal(t, 0, len(result.Results))
}

// TestGlob_PatternWithDotInFilename tests matching files that have dots in their
// names (common for versioned artifacts).
func TestGlob_PatternWithDotInFilename(t *testing.T) {
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "dist")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app-1.2.3.zip"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app-1.2.3-beta.zip"), []byte("x"), 0644))

	result, err := Glob(tmpDir, "dist/app-*.zip")
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Results), "got: %s", result.String())
}

// TestGlob_PatternToSlashIdempotentOnLinux verifies that filepath.ToSlash is a
// no-op on Linux (where filepath.Separator is already '/'), ensuring we don't
// break anything.
func TestGlob_PatternToSlashIdempotentOnLinux(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test only applies to non-Windows")
	}
	input := "path/to/artifact/*.zip"
	require.Equal(t, input, filepath.ToSlash(filepath.Clean(input)))
}

// TestGlob_RelativeBackslashPatternNormalized verifies that a relative pattern
// containing literal backslashes (as produced by CDS expansion of
// ${{ cds.workspace }} on Windows) is normalized at the Glob() entry point and
// matches files emitted by io/fs in forward-slash form. filepath.ToSlash is a
// no-op on Linux (backslash is a valid filename character), so the test is
// Windows-only.
func TestGlob_RelativeBackslashPatternNormalized(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("filepath.ToSlash is a no-op on non-Windows; backslash stays literal")
	}
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")
	require.NoError(t, os.MkdirAll(distDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "foo.zip"), []byte("x"), 0644))

	// Pattern as produced by CDS workspace expansion on Windows
	pattern := `dist\foo.zip`
	result, err := Glob(tmpDir, pattern)
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Results), "got: %s", result.String())
	require.Equal(t, "dist/foo.zip", result.Results[0].Path)
}

// TestGlob_AbsoluteBackslashPattern reproduces the original bug report:
// ${{ cds.workspace }}\dist\foo.zip expanded by CDS on a Windows worker
// produces an absolute path with backslashes. After normalization at the
// Glob() entry point, the pattern must still match the file.
func TestGlob_AbsoluteBackslashPattern(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("absolute backslash patterns are Windows-specific")
	}
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")
	require.NoError(t, os.MkdirAll(distDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "foo.zip"), []byte("x"), 0644))

	// filepath.Join produces backslashes on Windows
	pattern := filepath.Join(tmpDir, "dist", "foo.zip")
	require.Contains(t, pattern, `\`, "precondition: pattern must contain backslashes on Windows")

	result, err := Glob(tmpDir, pattern)
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Results), "got: %s", result.String())
}

// TestGlob_AbsoluteBackslashPatternWithStar covers the wildcard variant of the
// absolute Windows path scenario (e.g. ${{ cds.workspace }}\dist\*.zip), which
// is a common usage in workflows.
func TestGlob_AbsoluteBackslashPatternWithStar(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific")
	}
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")
	require.NoError(t, os.MkdirAll(distDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "a.zip"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "b.zip"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "c.txt"), []byte("x"), 0644))

	pattern := filepath.Join(tmpDir, "dist") + `\*.zip`
	result, err := Glob(tmpDir, pattern)
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Results), "got: %s", result.String())
}
