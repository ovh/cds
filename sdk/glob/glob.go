package glob

import (
	"io/fs"
	"sort"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

type Globber struct {
	patterns []pattern
}

func New(expression string) *Globber {
	var expressionItems = splitListExpression(expression)
	var g = new(Globber)
	for _, expression := range expressionItems {
		expression = strings.TrimSpace(expression)
		var isExclude = isExcludeExpression(expression)
		if isExclude {
			expression = strings.ReplaceAll(expression, "!", "")
		}
		var p = pattern{
			raw:       expression,
			isExclude: isExclude,
		}
		g.patterns = append(g.patterns, p)
	}
	return g
}

func (g *Globber) Len() int {
	return len(g.patterns)
}

type Results []Result

func (results *Results) String() string {
	var buf strings.Builder
	sort.Slice(*results, func(i, j int) bool {
		return (*results)[i].Path < (*results)[j].Path
	})

	for i, r := range *results {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(r.Path)
	}
	return buf.String()
}

type Result struct {
	Path   string
	Result string
}

func (r *Result) String() string {
	return r.Path
}

func (g *Globber) MatchString(s string) (*Result, error) {
	var res *Result
	for _, p := range g.patterns {
		if p.isExclude {
			continue
		}
		result, err := p.Match(s)
		if err != nil {
			return nil, err
		}
		if result == "" {
			continue
		}
		res = &Result{Path: s, Result: result}
	}
	if res == nil {
		return nil, nil
	}
	for _, p := range g.patterns {
		if !p.isExclude {
			continue
		}
		result, err := p.Match(s)
		if err != nil {
			return nil, err
		}
		if result != "" && res != nil {
			res = nil
		}
	}
	return res, nil
}

func (g *Globber) Match(s ...string) ([]Result, error) {
	var final = map[string]Result{}
	for _, s := range s {
		for _, p := range g.patterns {
			if p.isExclude {
				continue
			}
			result, err := p.Match(s)
			if err != nil {
				return nil, err
			}
			if result == "" {
				continue
			}
			final[s] = Result{Path: s, Result: result}
		}
		for _, p := range g.patterns {
			if !p.isExclude {
				continue
			}
			result, err := p.Match(s)
			if err != nil {
				return nil, err
			}
			if result != "" {
				delete(final, s)
			}
		}
	}
	var finalResult []Result
	for _, r := range final {
		finalResult = append(finalResult, r)
	}
	return finalResult, nil
}

func isExcludeExpression(s string) bool {
	return strings.ContainsRune(s, '!')
}

func splitListExpression(s string) []string {
	switch {
	case strings.ContainsRune(s, '\n'):
		return strings.Split(s, "\n")
	case strings.ContainsRune(s, ' '):
		return strings.Split(s, " ")
	case strings.ContainsRune(s, ','):
		return strings.Split(s, ",")
	default:
		return []string{s}
	}
}

func (g *Globber) MatchFiles(_fs fs.FS, root string) (Results, error) {
	root, err := homedir.Expand(root)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rootFile, err := _fs.Open(root)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fi, err := rootFile.Stat()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !fi.IsDir() {
		return g.Match(root)
	}

	rootFS, err := fs.Sub(_fs, root)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var fileList []string
	if err := fs.WalkDir(rootFS, ".", g.walkdirFunc(&fileList)); err != nil {
		return nil, err
	}

	return g.Match(fileList...)
}

func (g *Globber) walkdirFunc(target *[]string) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}
		Debug("path: %s name: %s, isDir:%v", path, d.Name(), d.IsDir())
		if !d.IsDir() {
			*target = append(*target, path)
		}
		return nil
	}
}

func Glob(_fs fs.FS, root string, pattern string) (Results, error) {
	return New(pattern).MatchFiles(_fs, root)
}
