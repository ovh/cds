package secretservice

/*
#cgo pkg-config: libsecret-1

#include "secretservice_linux.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/docker/docker-credential-helpers/credentials"
)

// Secretservice handles secrets using Linux secret-service as a store.
type Secretservice struct{}

// Add adds new credentials to the keychain.
func (h Secretservice) Add(creds *credentials.Credentials) error {
	if creds == nil {
		return errors.New("missing credentials")
	}
	server := C.CString(creds.ServerURL)
	defer C.free(unsafe.Pointer(server))
	username := C.CString(creds.Username)
	defer C.free(unsafe.Pointer(username))
	secret := C.CString(creds.Secret)
	defer C.free(unsafe.Pointer(secret))

	if err := C.add(server, username, secret); err != nil {
		defer C.g_error_free(err)
		errMsg := (*C.char)(unsafe.Pointer(err.message))
		return errors.New(C.GoString(errMsg))
	}
	return nil
}

// Delete removes credentials from the store.
func (h Secretservice) Delete(serverURL string) error {
	if serverURL == "" {
		return errors.New("missing server url")
	}
	server := C.CString(serverURL)
	defer C.free(unsafe.Pointer(server))

	if err := C.delete(server); err != nil {
		defer C.g_error_free(err)
		errMsg := (*C.char)(unsafe.Pointer(err.message))
		return errors.New(C.GoString(errMsg))
	}
	return nil
}

// Get returns the username and secret to use for a given registry server URL.
func (h Secretservice) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", errors.New("missing server url")
	}
	var username *C.char
	defer C.free(unsafe.Pointer(username))
	var secret *C.char
	defer C.free(unsafe.Pointer(secret))
	server := C.CString(serverURL)
	defer C.free(unsafe.Pointer(server))

	err := C.get(server, &username, &secret)
	if err != nil {
		defer C.g_error_free(err)
		errMsg := (*C.char)(unsafe.Pointer(err.message))
		return "", "", errors.New(C.GoString(errMsg))
	}
	user := C.GoString(username)
	pass := C.GoString(secret)
	if pass == "" {
		return "", "", credentials.NewErrCredentialsNotFound()
	}
	return user, pass, nil
}

func (h Secretservice) List() ([]string, []string, error) {
	var pathsC **C.char
	defer C.free(unsafe.Pointer(pathsC))
	var acctsC **C.char
	defer C.free(unsafe.Pointer(acctsC))
	var listLenC C.uint
	err := C.list(&pathsC, &acctsC, &listLenC)
	if err != nil {
		defer C.free(unsafe.Pointer(err))
		return nil, nil, errors.New("Error from list function in secretservice_linux.c likely due to error in secretservice library")
	}
	listLen := int(listLenC)
	pathTmp := (*[1 << 30]*C.char)(unsafe.Pointer(pathsC))[:listLen:listLen]
	acctTmp := (*[1 << 30]*C.char)(unsafe.Pointer(acctsC))[:listLen:listLen]
	paths := make([]string, listLen)
	accts := make([]string, listLen)
	for i := 0; i < listLen; i++ {
		paths[i] = C.GoString(pathTmp[i])
		accts[i] = C.GoString(acctTmp[i])
	}
	C.freeListData(&pathsC, listLenC)
	C.freeListData(&acctsC, listLenC)
	return paths, accts, nil
}
