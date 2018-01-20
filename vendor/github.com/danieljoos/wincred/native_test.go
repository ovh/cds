package wincred

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockProc struct {
	mock.Mock
	orig   proc
	target *proc
}

func (t *mockProc) Setup(target *proc) {
	t.target = target
	t.orig = *t.target
	*(t.target) = t
}

func (t *mockProc) TearDown() {
	*(t.target) = t.orig
}

func (t *mockProc) Call(a ...uintptr) (r1, r2 uintptr, lastErr error) {
	args := t.Called(a)
	return uintptr(args.Int(0)), uintptr(args.Int(1)), args.Error(2)
}

func TestNativeCredRead_MockFailure(t *testing.T) {
	// The test error
	testError := errors.New("test error")
	// Mock `CreadRead`: returns failure state and the error
	mockCredRead := new(mockProc)
	mockCredRead.On("Call", mock.AnythingOfType("[]uintptr")).Return(0, 0, testError)
	mockCredRead.Setup(&procCredRead)
	defer mockCredRead.TearDown()
	// Mock `CredFree`: Must not be called
	mockCredFree := new(mockProc)
	mockCredFree.On("Call", mock.AnythingOfType("[]uintptr")).Return(0, 0, nil)
	mockCredFree.Setup(&procCredFree)
	defer mockCredFree.TearDown()

	// Test it:
	var res *Credential
	var err error
	assert.NotPanics(t, func() { res, err = nativeCredRead("foo", naCRED_TYPE_GENERIC) })
	assert.Nil(t, res)
	assert.NotNil(t, err)
	assert.Equal(t, "test error", err.Error())
	mockCredRead.AssertNumberOfCalls(t, "Call", 1)
	mockCredFree.AssertNumberOfCalls(t, "Call", 0)
}

func TestNativeCredRead_Mock(t *testing.T) {
	// prepare some test data
	cred := new(Credential)
	cred.TargetName = "Foo"
	cred.Comment = "Bar"
	cred.CredentialBlob = []byte{1, 2, 3}
	credNative := nativeFromCredential(cred)
	// Mock `CreadRead`: returns success and sets the pointer to the prepared nativeCred struct
	mockCredRead := new(mockProc)
	mockCredRead.
		On("Call", mock.AnythingOfType("[]uintptr")).
		Return(1, 0, nil).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]uintptr)
			assert.Equal(t, 4, len(arg))
			*(*uintptr)(unsafe.Pointer(arg[3])) = uintptr(unsafe.Pointer(credNative))
		})
	mockCredRead.Setup(&procCredRead)
	defer mockCredRead.TearDown()
	// Mock `CredFree`: Must be called as well with the correct pointer
	mockCredFree := new(mockProc)
	mockCredFree.
		On("Call", mock.AnythingOfType("[]uintptr")).
		Return(0, 0, nil).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]uintptr)
			assert.Equal(t, 1, len(arg))
			assert.Equal(t, uintptr(unsafe.Pointer(credNative)), arg[0])
		})
	mockCredFree.Setup(&procCredFree)
	defer mockCredFree.TearDown()

	// Test it:
	var res *Credential
	var err error
	assert.NotPanics(t, func() { res, err = nativeCredRead("Foo", naCRED_TYPE_GENERIC) })
	mockCredRead.AssertNumberOfCalls(t, "Call", 1)
	mockCredFree.AssertNumberOfCalls(t, "Call", 1)
	assert.NotNil(t, res)
	assert.Nil(t, err)
	assert.Equal(t, "Foo", res.TargetName)
	assert.Equal(t, "Bar", res.Comment)
	assert.Equal(t, []byte{1, 2, 3}, res.CredentialBlob)
	assert.NotEqual(t, &cred, &res)
}

func TestNativeCredWrite_MockFailure(t *testing.T) {
	// Mock `CreadWrite`: returns failure state and the error
	mockCredWrite := new(mockProc)
	mockCredWrite.On("Call", mock.AnythingOfType("[]uintptr")).Return(0, 0, errors.New("test error"))
	mockCredWrite.Setup(&procCredWrite)
	defer mockCredWrite.TearDown()

	// Test it:
	var err error
	assert.NotPanics(t, func() { err = nativeCredWrite(new(Credential), naCRED_TYPE_GENERIC) })
	assert.NotNil(t, err)
	assert.Equal(t, "test error", err.Error())
	mockCredWrite.AssertNumberOfCalls(t, "Call", 1)
}

func TestNativeCredWrite_Mock(t *testing.T) {
	// Mock `CreadWrite`: returns success state
	mockCredWrite := new(mockProc)
	mockCredWrite.On("Call", mock.AnythingOfType("[]uintptr")).Return(1, 0, nil)
	mockCredWrite.Setup(&procCredWrite)
	defer mockCredWrite.TearDown()

	// Test it:
	var err error
	assert.NotPanics(t, func() { err = nativeCredWrite(new(Credential), naCRED_TYPE_GENERIC) })
	assert.Nil(t, err)
	mockCredWrite.AssertNumberOfCalls(t, "Call", 1)
}

func TestNativeCredDelete_MockFailure(t *testing.T) {
	// Mock `CreadDelete`: returns failure state and an error
	mockCredDelete := new(mockProc)
	mockCredDelete.On("Call", mock.AnythingOfType("[]uintptr")).Return(0, 0, errors.New("test error"))
	mockCredDelete.Setup(&procCredDelete)
	defer mockCredDelete.TearDown()

	// Test it:
	var err error
	assert.NotPanics(t, func() { err = nativeCredDelete(new(Credential), naCRED_TYPE_GENERIC) })
	assert.NotNil(t, err)
	assert.Equal(t, "test error", err.Error())
	mockCredDelete.AssertNumberOfCalls(t, "Call", 1)
}

func TestNativeCredDelete_Mock(t *testing.T) {
	// Mock `CreadDelete`: returns success state
	mockCredDelete := new(mockProc)
	mockCredDelete.On("Call", mock.AnythingOfType("[]uintptr")).Return(1, 0, nil)
	mockCredDelete.Setup(&procCredDelete)
	defer mockCredDelete.TearDown()

	// Test it:
	var err error
	assert.NotPanics(t, func() { err = nativeCredDelete(new(Credential), naCRED_TYPE_GENERIC) })
	assert.Nil(t, err)
	mockCredDelete.AssertNumberOfCalls(t, "Call", 1)
}

func TestNativeCredEnumerate_MockFailure(t *testing.T) {
	// The test error
	testError := errors.New("test error")
	// Mock `CreadEnumerate`: returns failure state and the error
	mockCredEnumerate := new(mockProc)
	mockCredEnumerate.On("Call", mock.AnythingOfType("[]uintptr")).Return(0, 0, testError)
	mockCredEnumerate.Setup(&procCredEnumerate)
	defer mockCredEnumerate.TearDown()
	// Mock `CredFree`: Must not be called
	mockCredFree := new(mockProc)
	mockCredFree.On("Call", mock.AnythingOfType("[]uintptr")).Return(0, 0, nil)
	mockCredFree.Setup(&procCredFree)
	defer mockCredFree.TearDown()

	// Test it:
	var res []*Credential
	var err error
	assert.NotPanics(t, func() { res, err = nativeCredEnumerate("", true) })
	assert.Nil(t, res)
	assert.NotNil(t, err)
	assert.Equal(t, "test error", err.Error())
	mockCredEnumerate.AssertNumberOfCalls(t, "Call", 1)
	mockCredFree.AssertNumberOfCalls(t, "Call", 0)
}

func TestNativeCredEnumerate_Mock(t *testing.T) {
	// prepare some test data
	creds := []*Credential{new(Credential), new(Credential)}
	creds[0].TargetName = "Foo"
	creds[1].TargetName = "Bar"
	credsNative := [](*nativeCREDENTIAL){
		nativeFromCredential(creds[0]),
		nativeFromCredential(creds[1]),
	}

	// Mock `CreadEnumerate`: returns success and sets the pointer to the prepared nativeCreds array
	mockCredEnumerate := new(mockProc)
	mockCredEnumerate.
		On("Call", mock.AnythingOfType("[]uintptr")).
		Return(1, 0, nil).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]uintptr)
			assert.Equal(t, 4, len(arg))
			*(*int)(unsafe.Pointer(arg[2])) = len(credsNative)
			*(*uintptr)(unsafe.Pointer(arg[3])) = uintptr(unsafe.Pointer(&credsNative[0]))
		})
	mockCredEnumerate.Setup(&procCredEnumerate)
	defer mockCredEnumerate.TearDown()
	// Mock `CredFree`: Must be called as well with the correct pointer
	mockCredFree := new(mockProc)
	mockCredFree.
		On("Call", mock.AnythingOfType("[]uintptr")).
		Return(0, 0, nil).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]uintptr)
			assert.Equal(t, 1, len(arg))
			assert.Equal(t, uintptr(unsafe.Pointer(&credsNative[0])), arg[0])
		})
	mockCredFree.Setup(&procCredFree)
	defer mockCredFree.TearDown()

	// Test it:
	var res []*Credential
	var err error
	assert.NotPanics(t, func() { res, err = nativeCredEnumerate("", true) })
	mockCredEnumerate.AssertNumberOfCalls(t, "Call", 1)
	mockCredFree.AssertNumberOfCalls(t, "Call", 1)
	assert.NotNil(t, res)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "Foo", res[0].TargetName)
	assert.Equal(t, "Bar", res[1].TargetName)
}
