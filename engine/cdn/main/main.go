package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
)

// to get file's size
type Size interface {
	Size() int64
}

func main() {
	http.HandleFunc("/upload", uploadHandle)
	go http.ListenAndServe(":9099", nil)

	extraParams := map[string]string{
		"title":       "My Document",
		"author":      "Matt Aimonetti",
		"description": "A document with all the Go programming language secrets",
	}
	request, err := newfileUploadRequest("http://localhost:9099/upload", extraParams, "file", "/Users/sguiheux/blabla.toml")
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{}
	_, err = client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
}

func uploadHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf(">>>%+v", r.FormValue("dataJSON"))
	//r.ParseMultipartForm(128 << 20)        //annotate or not gets the same.
	reader, err := r.MultipartReader()
	if err != nil {
		log.Println(err)
		return
	}
	for {
		p, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		bts := &bytes.Buffer{}
		_, err = io.Copy(bts, p)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Printf("\n%s\n", bts.String())
	}
	return
}

func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create tar part
	dataFileHeader := make(textproto.MIMEHeader)
	dataFileHeader.Set("Content-Type", "application/toml")
	dataFileHeader.Set("Content-Disposition", "form-data; name=\"dataFiles\"; filename=\"blabla.toml\"")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	dataPart, err := writer.CreatePart(dataFileHeader)
	if _, err := io.Copy(dataPart, file); err != nil {
		return nil, err
	}
	/*
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile(paramName, filepath.Base(path))
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(part, file)
	*/

	bts, _ := json.Marshal(params)
	_ = writer.WriteField("dataJSON", string(bts))
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}
