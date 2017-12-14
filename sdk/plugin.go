package sdk

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
)

//DownloadPlugin download plugin from action
func DownloadPlugin(name string, destdir string) error {
	var lasterr error
	for retry := 5; retry >= 0; retry-- {
		uri := fmt.Sprintf("/plugin/download/%s?accept-redirect=true", name)
		reader, code, err := Stream("GET", uri, nil)
		if err != nil {
			lasterr = err
			continue
		}
		if code >= 300 {
			lasterr = fmt.Errorf("HTTP %d", code)
			continue
		}
		destPath := path.Join(destdir, name)
		//If the file already exists, remove it
		if _, errstat := os.Stat(destPath); errstat == nil {
			os.RemoveAll(destPath)
		}

		f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		if _, err = io.Copy(f, reader); err != nil {
			lasterr = err
		}

		if err := f.Close(); err == nil {
			fmt.Printf("Download %s completed\n", destPath)
			return nil
		}
	}

	return fmt.Errorf("x5: %s", lasterr)
}

//UploadPlugin uploads binary file to perform a new action
func UploadPlugin(filePath string, update bool) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}

	file, erro := os.Open(filePath)
	if erro != nil {
		return nil, erro
	}
	defer file.Close()

	//_, name := filepath.Split(filePath)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, errc := writer.CreateFormFile("UploadFile", filepath.Base(filePath))
	if errc != nil {
		return nil, errc
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	method := "POST"
	if update {
		method = "PUT"
	}
	btes, code, err := UploadMultiPart(method, "/plugin", body, SetHeader("uploadfile", filePath), SetHeader("Content-Type", writer.FormDataContentType()))
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP Error %d\n", code)
	}

	return btes, nil
}

//DeletePlugin delete plugin
func DeletePlugin(name string) error {
	path := fmt.Sprintf("/plugin/%s", name)

	_, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return nil
}
