package wincred

import (
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func fixtureCredential() (cred *Credential) {
	cred = new(Credential)
	cred.TargetName = "Foo"
	cred.Comment = "Bar"
	cred.LastWritten = time.Now()
	cred.TargetAlias = "MyAlias"
	cred.UserName = "Nobody"
	cred.Persist = PersistLocalMachine
	return
}

func TestUtf16PtrToString(t *testing.T) {
	input := "Foo Bar"
	utf16Ptr, err := syscall.UTF16PtrFromString(input)
	output := utf16PtrToString(utf16Ptr)
	assert.Nil(t, err)
	assert.Equal(t, input, output)
}

func TestUtf16PtrToString_Nil(t *testing.T) {
	output := utf16PtrToString(nil)
	assert.Equal(t, "", output)
}

func BenchmarkUtf16PtrToString(b *testing.B) {
	input := "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam" +
		"nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam"
	utf16Ptr, _ := syscall.UTF16PtrFromString(input)
	for i := 0; i < b.N; i++ {
		utf16PtrToString(utf16Ptr)
	}
}

func TestUtf16ToByte(t *testing.T) {
	input := []uint16{1, 2, 3, 4, 258}
	output := utf16ToByte(input)
	assert.Equal(t, 10, len(output))
	assert.Equal(t, byte(0x01), output[0])
	assert.Equal(t, byte(0x00), output[1])
	assert.Equal(t, byte(0x02), output[2])
	assert.Equal(t, byte(0x00), output[3])
	assert.Equal(t, byte(0x03), output[4])
	assert.Equal(t, byte(0x00), output[5])
	assert.Equal(t, byte(0x04), output[6])
	assert.Equal(t, byte(0x00), output[7])
	assert.Equal(t, byte(0x02), output[8]) // 2 +
	assert.Equal(t, byte(0x01), output[9]) // 1 * 256 = 258
}

func TestUtf16ToByte_Empty(t *testing.T) {
	input := []uint16{}
	output := utf16ToByte(input)
	assert.Equal(t, 0, len(output))
}

func BenchmarkUtf16ToByte(b *testing.B) {
	input := []uint16{1, 2, 3, 4, 258}
	for i := 0; i < b.N; i++ {
		utf16ToByte(input)
	}
}

func TestGoBytes(t *testing.T) {
	input := []byte{1, 2, 3, 4, 5}
	output := goBytes(uintptr(unsafe.Pointer(&input[0])), uint32(len(input)))
	assert.Equal(t, len(input), len(output))
	assert.Equal(t, input[0], output[0])
	assert.Equal(t, input[1], output[1])
	assert.Equal(t, input[2], output[2])
	assert.Equal(t, input[3], output[3])
	assert.Equal(t, input[4], output[4])
	input[0] = 99
	assert.NotEqual(t, input[0], output[0]) // is it a copy?
}

func TestGoBytes_Null(t *testing.T) {
	assert.NotPanics(t, func() {
		output := goBytes(0, 123)
		assert.Equal(t, []byte{}, output)
	})
}

func BenchmarkGoBytes(b *testing.B) {
	input := []byte{1, 2, 3, 4, 5}
	for i := 0; i < b.N; i++ {
		goBytes(uintptr(unsafe.Pointer(&input[0])), uint32(len(input)))
	}
}

func TestConversion(t *testing.T) {
	cred := fixtureCredential()
	native := nativeFromCredential(cred)
	res := nativeToCredential(native)
	assert.NotEqual(t, uintptr(0), native.TargetName)
	assert.Equal(t, cred.TargetName, res.TargetName)
	assert.Equal(t, cred.Comment, res.Comment)
	assert.Equal(t, cred.LastWritten, res.LastWritten)
	assert.Equal(t, cred.TargetAlias, res.TargetAlias)
	assert.Equal(t, cred.UserName, res.UserName)
	cred.TargetName = "Another Foo"
	assert.NotEqual(t, cred.TargetName, res.TargetName)
}

func TestConversion_Nil(t *testing.T) {
	assert.NotPanics(t, func() {
		res := nativeToCredential(nil)
		assert.Nil(t, res)
	})
	assert.NotPanics(t, func() {
		res := nativeFromCredential(nil)
		assert.Nil(t, res)
	})
}

func TestConversion_CredentialBlob(t *testing.T) {
	cred := new(Credential)
	cred.CredentialBlob = []byte{1, 2, 3}
	native := nativeFromCredential(cred)
	res := nativeToCredential(native)
	assert.Equal(t, uint32(3), native.CredentialBlobSize)
	assert.NotEqual(t, uintptr(0), native.CredentialBlob)
	assert.Equal(t, cred.CredentialBlob, res.CredentialBlob)
}

func TestConversion_CredentialBlob_Empty(t *testing.T) {
	cred := new(Credential)
	cred.CredentialBlob = []byte{} // empty blob
	native := nativeFromCredential(cred)
	res := nativeToCredential(native)
	assert.Equal(t, uintptr(0), native.CredentialBlob)
	assert.Equal(t, uint32(0), native.CredentialBlobSize)
	assert.Equal(t, []byte{}, res.CredentialBlob)
}

func TestConversion_CredentialBlob_Nil(t *testing.T) {
	cred := new(Credential)
	cred.CredentialBlob = nil // nil blob
	native := nativeFromCredential(cred)
	res := nativeToCredential(native)
	assert.Equal(t, uintptr(0), native.CredentialBlob)
	assert.Equal(t, uint32(0), native.CredentialBlobSize)
	assert.Equal(t, []byte{}, res.CredentialBlob)
}

func TestConversion_Attributes(t *testing.T) {
	cred := new(Credential)
	cred.Attributes = []CredentialAttribute{
		{Keyword: "Foo", Value: []byte{1, 2, 3}},
		{Keyword: "Bar", Value: []byte{}},
	}
	native := nativeFromCredential(cred)
	res := nativeToCredential(native)
	assert.NotEqual(t, uintptr(0), native.Attributes)
	assert.Equal(t, uint32(2), native.AttributeCount)
	assert.Equal(t, cred.Attributes, res.Attributes)
}

func TestConversion_Attributes_Empty(t *testing.T) {
	cred := new(Credential)
	cred.Attributes = []CredentialAttribute{}
	native := nativeFromCredential(cred)
	res := nativeToCredential(native)
	assert.Equal(t, uintptr(0), native.Attributes)
	assert.Equal(t, uint32(0), native.AttributeCount)
	assert.Equal(t, []CredentialAttribute{}, res.Attributes)
}

func TestConversion_Attributes_Nil(t *testing.T) {
	cred := new(Credential)
	cred.Attributes = nil
	native := nativeFromCredential(cred)
	res := nativeToCredential(native)
	assert.Equal(t, uintptr(0), native.Attributes)
	assert.Equal(t, uint32(0), native.AttributeCount)
	assert.Equal(t, []CredentialAttribute{}, res.Attributes)
}

func BenchmarkConversionFrom(b *testing.B) {
	cred := fixtureCredential()
	for i := 0; i < b.N; i++ {
		nativeFromCredential(cred)
	}
}

func BenchmarkConversionTo(b *testing.B) {
	cred := fixtureCredential()
	n := nativeFromCredential(cred)
	for i := 0; i < b.N; i++ {
		nativeToCredential(n)
	}
}
