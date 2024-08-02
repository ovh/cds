package glob

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type Globber struct {
	patterns []pattern
}

func New(expression ...string) *Globber {
	if len(expression) == 1 {
		expression = splitListExpression(expression[0])
	}
	var expressionItems = expression
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

type FileResults struct {
	DirFS   fs.FS
	Results Results
}

func (f *FileResults) String() string {
	if f == nil {
		return ""
	}
	return f.Results.String()
}

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

func (g *Globber) MatchFiles(_fs fs.FS) (*FileResults, error) {
	var fileList []string
	if err := fs.WalkDir(_fs, ".", g.walkdirFunc(&fileList)); err != nil {
		return nil, err
	}

	results, err := g.Match(fileList...)
	if err != nil {
		return nil, err
	}
	return &FileResults{
		DirFS:   _fs,
		Results: results,
	}, nil
}

func (g *Globber) walkdirFunc(target *[]string) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			Debug(err.Error())
			return nil
		}
		Debug("path: %s name: %s, isDir:%v", path, d.Name(), d.IsDir())
		if !d.IsDir() {
			*target = append(*target, path)
		}
		return nil
	}
}

func Glob(cwd string, pattern string) (*FileResults, error) {
	splittedExpression := splitListExpression(pattern)
	var absoluteExpressions []string
	var retaliveExpressions []string
	for i := range splittedExpression {
		expression := splittedExpression[i]
		s := strings.TrimPrefix(expression, "!")
		if filepath.IsAbs(s) {
			absoluteExpressions = append(absoluteExpressions, s)
		} else {
			retaliveExpressions = append(retaliveExpressions, s)
		}
	}

	if len(absoluteExpressions) > 0 && len(retaliveExpressions) > 0 {
		return nil, errors.New("mixing absolute and relative patterns is not supported")
	}

	if len(absoluteExpressions) > 0 {
		cwd = LongestCommonPathPrefix(absoluteExpressions)
		Debug("longest of %v prefix is %s", absoluteExpressions, cwd)

		for i := range splittedExpression {
			expression := splittedExpression[i]
			s := strings.TrimPrefix(expression, "!")
			s = strings.TrimPrefix(s, cwd)
			if isExcludeExpression(splittedExpression[i]) {
				s = "!" + s
			}
			splittedExpression[i] = s
		}
	}

	Debug("cwd is %s", cwd)

	return New(splittedExpression...).MatchFiles(os.DirFS(cwd))
}

func LongestCommonPathPrefix(strs []string) string {
	var longestPrefix string
	var endPrefix bool
	var lastPath string

	if len(strs) > 0 {
		sort.Strings(strs)
		first := string(strs[0])
		last := string(strs[len(strs)-1])

		for i := 0; i < len(first); i++ {
			if !endPrefix && string(last[i]) == string(first[i]) {
				longestPrefix += string(last[i])
				if _, err := os.ReadDir(longestPrefix); last[i] == '/' && err == nil {
					lastPath = longestPrefix
				}
			} else {
				endPrefix = true
			}
		}

	}
	return lastPath
}
