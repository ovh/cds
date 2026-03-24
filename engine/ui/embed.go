package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
)

//go:embed dist/*
var embeddedStaticFiles embed.FS

// hasEmbeddedFiles checks if the embedded FS contains real static files
// (not just the .gitkeep placeholder).
func hasEmbeddedFiles() bool {
	entries, err := fs.ReadDir(embeddedStaticFiles, "dist")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() != ".gitkeep" {
			return true
		}
	}
	return false
}

// embeddedHTTPFS returns an http.FileSystem backed by the embedded files,
// with index.html dynamically generated from index.tmpl with variable substitution.
// Returns nil if no embedded files are available.
func (s *Service) embeddedHTTPFS() http.FileSystem {
	if !hasEmbeddedFiles() {
		return nil
	}

	// Get the dist subdirectory
	distFS, err := fs.Sub(embeddedStaticFiles, "dist")
	if err != nil {
		return nil
	}

	// Read and transform index.tmpl to produce index.html content
	indexHTML := s.transformIndexTmpl(distFS)

	return &embeddedFileSystem{
		base:     http.FS(distFS),
		indexHTML: indexHTML,
		hasIndex: indexHTML != nil,
	}
}

// transformIndexTmpl reads index.tmpl from the FS and applies variable
// replacements (baseURL, version, sentryURL), returning the resulting HTML.
// Returns nil if index.tmpl is not found.
func (s *Service) transformIndexTmpl(fsys fs.FS) []byte {
	data, err := fs.ReadFile(fsys, "index.tmpl")
	if err != nil {
		// Try index.html directly
		data, err = fs.ReadFile(fsys, "index.html")
		if err != nil {
			return nil
		}
	}

	regexBaseHref, err := regexp.Compile(`<base href=".*">`)
	if err != nil {
		return data
	}

	content := regexBaseHref.ReplaceAllString(string(data), `<base href="`+s.Cfg.BaseURL+`">`)
	content = strings.Replace(content, "window.cds_version = '';", "window.cds_version='"+sdk.VERSION+"';", -1)
	if s.Cfg.SentryURL != "" {
		content = strings.Replace(content, "window.cds_sentry_url = '';", "window.cds_sentry_url = '"+s.Cfg.SentryURL+"';", -1)
	}

	return []byte(content)
}

// embeddedFileSystem wraps an http.FileSystem to override index.html with
// the dynamically generated content.
type embeddedFileSystem struct {
	base     http.FileSystem
	indexHTML []byte
	hasIndex bool
}

func (efs *embeddedFileSystem) Open(name string) (http.File, error) {
	// Serve the transformed index.html from memory
	if efs.hasIndex && (name == "/index.html" || name == "index.html") {
		return newMemoryFile("index.html", efs.indexHTML), nil
	}
	return efs.base.Open(name)
}

// memoryFile implements http.File for in-memory content.
type memoryFile struct {
	*strings.Reader
	name string
	size int64
}

func newMemoryFile(name string, content []byte) *memoryFile {
	return &memoryFile{
		Reader: strings.NewReader(string(content)),
		name:   name,
		size:   int64(len(content)),
	}
}

func (f *memoryFile) Close() error { return nil }

func (f *memoryFile) Readdir(_ int) ([]os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func (f *memoryFile) Stat() (os.FileInfo, error) {
	return &memoryFileInfo{name: f.name, size: f.size}, nil
}

// memoryFileInfo implements os.FileInfo for in-memory files.
type memoryFileInfo struct {
	name string
	size int64
}

func (fi *memoryFileInfo) Name() string      { return fi.name }
func (fi *memoryFileInfo) Size() int64        { return fi.size }
func (fi *memoryFileInfo) Mode() os.FileMode  { return 0444 }
func (fi *memoryFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *memoryFileInfo) IsDir() bool        { return false }
func (fi *memoryFileInfo) Sys() interface{}   { return nil }
