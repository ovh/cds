package main

import "os"

type mockWriter struct {
	directories []directory
	files       []file
	copies      []copy
	extracts    []extract
}

type directory struct {
	path string
	perm os.FileMode
}

type file struct {
	path    string
	content string
	perm    os.FileMode
}

type copy struct {
	targetPath string
	path       string
	perm       os.FileMode
	sources    []string
}

type extract struct {
	targetPath string
	path       string
	archive    string
}

func (m *mockWriter) CreateDirectory(path string, perm os.FileMode) error {
	m.directories = append(m.directories, directory{path, perm})
	return nil
}

func (m *mockWriter) CreateFile(path string, content []byte, perm os.FileMode) error {
	m.files = append(m.files, file{path, string(content), perm})
	return nil
}

func (m *mockWriter) CopyFiles(targetPath string, path string, perm os.FileMode, sources ...string) error {
	m.copies = append(m.copies, copy{targetPath: targetPath, path: path, perm: perm, sources: sources})
	return nil
}

func (m *mockWriter) ExtractArchive(targetPath, path, archive string) error {
	m.extracts = append(m.extracts, extract{targetPath: targetPath, path: path, archive: archive})
	return nil
}
