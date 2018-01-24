package wincred

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testTargetName        = "github.com/danieljoos/wincred/testing"
	testTargetNameMissing = "github.com/danieljoos/wincred/missing"
)

func TestGenericCredential_EndToEnd(t *testing.T) {
	// 1. Create new credential `foo`
	cred := NewGenericCredential(testTargetName)
	cred.CredentialBlob = []byte("my secret")
	cred.Persist = PersistSession
	err := cred.Write()
	assert.Nil(t, err)

	// 2. Get the credential from the store
	cred, err = GetGenericCredential(testTargetName)
	assert.Nil(t, err)
	assert.NotNil(t, cred)
	assert.Equal(t, "my secret", string(cred.CredentialBlob))

	// 3. Search it in the list
	creds, err := List()
	assert.Nil(t, err)
	assert.NotNil(t, creds)
	assert.NotEqual(t, 0, len(creds))
	found := false
	for i := range creds {
		found = found || creds[i].TargetName == testTargetName
	}
	assert.True(t, found)

	// 4. Delete it
	err = cred.Delete()
	assert.Nil(t, err)

	// 5. Search it again in the list. It should be gone.
	creds, err = List()
	assert.Nil(t, err)
	assert.NotNil(t, creds)
	found = false
	for i := range creds {
		found = found || creds[i].TargetName == testTargetName
	}
	assert.False(t, found)
}

func TestGetGenericCredential_NotFound(t *testing.T) {
	cred, err := GetGenericCredential(testTargetNameMissing)
	assert.Nil(t, cred)
	assert.NotNil(t, err)
	// ERROR_NOT_FOUND (1168):
	// MSDN: https://msdn.microsoft.com/en-us/library/windows/desktop/ms681383(v=vs.85).aspx
	assert.Equal(t, "Element not found.", err.Error())
}

func TestGetGenericCredential_Empty(t *testing.T) {
	cred, err := GetGenericCredential("")
	assert.Nil(t, cred)
	assert.NotNil(t, err)
	// ERROR_INVALID_PARAMETER (87):
	// MSDN: https://msdn.microsoft.com/en-us/library/windows/desktop/ms681382(v=vs.85).aspx
	assert.Equal(t, "The parameter is incorrect.", err.Error())
}

func TestGenericCredential_WriteEmpty(t *testing.T) {
	cred := NewGenericCredential("")
	err := cred.Write()
	assert.NotNil(t, err)
	// ERROR_INVALID_PARAMETER (87):
	// MSDN: https://msdn.microsoft.com/en-us/library/windows/desktop/ms681382(v=vs.85).aspx.
	assert.Equal(t, "The parameter is incorrect.", err.Error())
}

func TestGenericCredential_DeleteNotFound(t *testing.T) {
	cred := NewGenericCredential(testTargetNameMissing)
	err := cred.Delete()
	assert.NotNil(t, err)
	// ERROR_NOT_FOUND (1168):
	// MSDN: https://msdn.microsoft.com/en-us/library/windows/desktop/ms681383(v=vs.85).aspx
	assert.Equal(t, "Element not found.", err.Error())
}

func ExampleList() {
	if creds, err := List(); err == nil {
		for _, cred := range creds {
			fmt.Println(cred.TargetName)
		}
	}
}

func ExampleGetGenericCredential() {
	if cred, err := GetGenericCredential("myGoApplication"); err == nil {
		fmt.Println(cred.TargetName, string(cred.CredentialBlob))
	}
}

func ExampleGenericCredential_Delete() {
	cred, _ := GetGenericCredential("myGoApplication")
	if err := cred.Delete(); err == nil {
		fmt.Println("Deleted")
	}
}

func ExampleGenericCredential_Write() {
	cred := NewGenericCredential("myGoApplication")
	cred.CredentialBlob = []byte("my secret")
	if err := cred.Write(); err == nil {
		fmt.Println("Created")
	}
}
