package glob

import (
	"testing"
)

func Test_isPattern(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{".", args{'.'}, true},
		{" ", args{' '}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPattern(tt.args.ch); got != tt.want {
				t.Errorf("isPatternRune() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPattern_Match(t *testing.T) {
	DebugEnabled = true
	tests := []struct {
		name    string
		pattern string
		content string
		want    string
		wantErr bool
	}{
		{"simple Pattern shoud match", "foo", "foo", "foo", false},
		{"simple Pattern shoud not match because content is too long", "foo", "foo1", "", false},
		{"simple Pattern shoud not match because content is too short", "foo1", "foo", "", false},

		{"wildcard Pattern shoud not match because content is too short", "foo?", "foo", "", false},
		{"wildcard Pattern shoud match because '?' matches '1'", "foo?", "foo1", "foo1", false},
		{"wildcard Pattern shoud match because '?' matches '2'", "foo?", "foo2", "foo2", false},
		{"wildcard Pattern shoud not match because content is too long", "foo?", "foo22", "", false},
		{"wildcard Pattern shoud match because 'f?o' matches 'foo'", "f?o", "foo", "foo", false},

		{"wildcard Pattern shoud match because '*' matches '1'", "foo*", "foo1", "foo1", false},
		{"wildcard Pattern shoud match because '*' matches '11'", "foo*", "foo11", "foo11", false},
		{"wildcard Pattern shoud match because '*' matches '11'", "foo/*", "foo/11", "11", false},
		{"wildcard Pattern shoud match because 'foo*' matches 'foo.txt'", "foo*", "foo.txt", "foo.txt", false},
		{"wildcard Pattern shoud not match because '*' doesn't matches '11/22'", "foo/*", "foo/11/22", "", false},
		{"wildcard Pattern shoud not match because '*/bar' matches 'foo/bar'", "*/bar", "foo/bar", "foo/bar", false},

		{"glob Pattern shoud match because '**' matches 'foo'", "**", "foo", "foo", false},
		{"glob Pattern shoud match because '**' matches 'foo/bar'", "**", "foo/bar", "foo/bar", false},
		{"glob Pattern shoud match because '**/*' matches 'foo/bar'", "**/*", "foo/bar", "foo/bar", false},
		{"glob Pattern shoud match because 'path/**/*.txt' matches 'path/foo/bar.txt'", "path/**/*.txt", "path/foo/bar.txt", "foo/bar.txt", false},
		{"glob Pattern shoud not match because 'path/**/*.txt' matches 'path/foo/bar'", "path/**/*.txt", "path/foo/bar", "", false},

		{"wildcard Pattern preserves hierarchy", "path/to/*/directory/foo?.txt", "path/to/some/directory/foo1.txt", "some/directory/foo1.txt", false},
		{"wildcard Pattern preserves hierarchy", "path/to/*/directory/foo?.txt", "path/to/other/directory/foo1.txt", "other/directory/foo1.txt", false},

		{"a", "path/to/artifact/foo*.txt", "path/to/artifact/foooo.txt", "foooo.txt", false},

		{"glob Pattern shoud match because '**/*' matches 'foo/bar/buz.txt'", "**/*", "foo/bar/buz.txt", "foo/bar/buz.txt", false},

		{"wildcard Pattern with range", "path/**/[abc]rtifac?/*", "path/to/artifact/foo", "to/artifact/foo", false},
		{"glob Pattern shoud match because '**/*.txt' matches 'foo/bar.txt'", "**/*.txt", "foo/bar.txt", "foo/bar.txt", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &pattern{
				raw: tt.pattern,
			}
			got, err := f.Match(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Pattern.Match() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Pattern.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
