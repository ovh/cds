package main

import "os"

type mockWriter struct {
	directories []directory
	files       []file
	copies      []copy
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
	path    string
	perm    os.FileMode
	sources []string
}

func (m *mockWriter) CreateDirectory(path string, perm os.FileMode) error {
	m.directories = append(m.directories, directory{path, perm})
	return nil
}

func (m *mockWriter) CreateFile(path string, content []byte, perm os.FileMode) error {
	m.files = append(m.files, file{path, string(content), perm})
	return nil
}

func (m *mockWriter) CopyFiles(path string, perm os.FileMode, sources ...string) error {
	m.copies = append(m.copies, copy{path, perm, sources})
	return nil
}
