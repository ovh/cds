package permbits

import (
	"os"
	"runtime"
	"syscall"
	"testing"
)

func TestFile(t *testing.T) {
	filename := ".permbits_test_temporary_file"

	// Clean up from previous test runs
	os.RemoveAll(filename)

	// Create the test file
	syscall.Umask(0)
	file, err := os.Create(filename)
	if err != nil {
		// We can't perform these tests, but this is not a failure
		return
	}
	fi, err := file.Stat()
	if err != nil {
		return
	}

	fm := fi.Mode()
	perms := FileMode(fm)
	if perms != 0666 {
		t.Errorf("PermissionBits created with wrong value. Should be 666, got %o", perms)
	}

	perms.SetUserExecute(true)
	perms.SetGroupExecute(true)
	perms.SetOtherExecute(true)

	UpdateFileMode(&fm, perms)

	os.Chmod(filename, fm)

	perms, err = Stat(filename)
	if err != nil {
		t.Error(err)
	}
	if perms != 0777 {
		t.Errorf("Failed to chmod file. Should be 777, got %o", perms)
	}

	// Only check sticky bits on Linux where it is supported
	if runtime.GOOS == "linux" {
		perms.SetSetuid(true)
		perms.SetSetgid(true)
		perms.SetSticky(true)

		UpdateFileMode(&fm, perms)

		os.Chmod(filename, fm)

		perms, err = Stat(filename)
		if err != nil {
			t.Error(err)
		}
		if perms != 07777 {
			t.Errorf("Failed to chmod file. Should be 7777, got %o", perms)
		}

		// Test Chmod directly
		Chmod(filename, 06666)
		Chmod(filename, 07777)
		perms, err = Stat(filename)
		if err != nil {
			t.Error(err)
		}
		if perms != 07777 {
			t.Errorf("Failed to chmod file directly. Should be 7777, got %o", perms)
		}
	}

	// Clean up
	os.RemoveAll(".permbits_test_temporary_file")
}

func TestAllTrueAllFalse(t *testing.T) {
	var allTrue PermissionBits

	allTrue.SetSetuid(true)
	allTrue.SetSetgid(true)
	allTrue.SetSticky(true)
	allTrue.SetUserRead(true)
	allTrue.SetUserWrite(true)
	allTrue.SetUserExecute(true)
	allTrue.SetGroupRead(true)
	allTrue.SetGroupWrite(true)
	allTrue.SetGroupExecute(true)
	allTrue.SetOtherRead(true)
	allTrue.SetOtherWrite(true)
	allTrue.SetOtherExecute(true)

	if !allTrue.Setuid() {
		t.Error("allTrue: UserRead returns false")
	}
	if !allTrue.Setgid() {
		t.Error("allTrue: UserRead returns false")
	}
	if !allTrue.Sticky() {
		t.Error("allTrue: UserRead returns false")
	}
	if !allTrue.UserRead() {
		t.Error("allTrue: UserRead returns false")
	}
	if !allTrue.UserWrite() {
		t.Error("allTrue: UserWrite returns false")
	}
	if !allTrue.UserExecute() {
		t.Error("allTrue: UserExecute returns false")
	}
	if !allTrue.GroupRead() {
		t.Error("allTrue: GroupRead returns false")
	}
	if !allTrue.GroupWrite() {
		t.Error("allTrue: GroupWrite returns false")
	}
	if !allTrue.GroupExecute() {
		t.Error("allTrue: GroupExecute returns false")
	}
	if !allTrue.OtherRead() {
		t.Error("allTrue: OtherRead returns false")
	}
	if !allTrue.OtherWrite() {
		t.Error("allTrue: OtherWrite returns false")
	}
	if !allTrue.OtherExecute() {
		t.Error("allTrue: OtherExecute returns false")
	}

	if allTrue.String() != "rwxrwxrwx" {
		t.Error("allTrue: string incorrect")
	}

	allFalse := allTrue
	allFalse.SetSetuid(false)
	allFalse.SetSetgid(false)
	allFalse.SetSticky(false)
	allFalse.SetUserRead(false)
	allFalse.SetUserWrite(false)
	allFalse.SetUserExecute(false)
	allFalse.SetGroupRead(false)
	allFalse.SetGroupWrite(false)
	allFalse.SetGroupExecute(false)
	allFalse.SetOtherRead(false)
	allFalse.SetOtherWrite(false)
	allFalse.SetOtherExecute(false)

	if allFalse.Setuid() {
		t.Error("allFalse: UserRead returns true")
	}
	if allFalse.Setgid() {
		t.Error("allFalse: UserRead returns true")
	}
	if allFalse.Sticky() {
		t.Error("allFalse: UserRead returns true")
	}
	if allFalse.UserRead() {
		t.Error("allFalse: UserRead returns true")
	}
	if allFalse.UserWrite() {
		t.Error("allFalse: UserWrite returns true")
	}
	if allFalse.UserExecute() {
		t.Error("allFalse: UserExecute returns true")
	}
	if allFalse.GroupRead() {
		t.Error("allFalse: GroupRead returns true")
	}
	if allFalse.GroupWrite() {
		t.Error("allFalse: GroupWrite returns true")
	}
	if allFalse.GroupExecute() {
		t.Error("allFalse: GroupExecute returns true")
	}
	if allFalse.OtherRead() {
		t.Error("allFalse: OtherRead returns true")
	}
	if allFalse.OtherWrite() {
		t.Error("allFalse: OtherWrite returns true")
	}
	if allFalse.OtherExecute() {
		t.Error("allFalse: OtherExecute returns true")
	}
	if allFalse.String() != "---------" {
		t.Error("allFalse: string incorrect")
	}

}

// Confirming that running SetX twice in idempotent and doesn't break
func TestPermissionsDouble(t *testing.T) {
	var allTrue PermissionBits

	allTrue.SetSetuid(true)
	allTrue.SetSetgid(true)
	allTrue.SetSticky(true)
	allTrue.SetUserRead(true)
	allTrue.SetUserWrite(true)
	allTrue.SetUserExecute(true)
	allTrue.SetGroupRead(true)
	allTrue.SetGroupWrite(true)
	allTrue.SetGroupExecute(true)
	allTrue.SetOtherRead(true)
	allTrue.SetOtherWrite(true)
	allTrue.SetOtherExecute(true)
	allTrue.SetSetuid(true)
	allTrue.SetSetgid(true)
	allTrue.SetSticky(true)
	allTrue.SetUserRead(true)
	allTrue.SetUserWrite(true)
	allTrue.SetUserExecute(true)
	allTrue.SetGroupRead(true)
	allTrue.SetGroupWrite(true)
	allTrue.SetGroupExecute(true)
	allTrue.SetOtherRead(true)
	allTrue.SetOtherWrite(true)
	allTrue.SetOtherExecute(true)

	if !allTrue.Setuid() {
		t.Error("allTrue: UserRead returns false")
	}
	if !allTrue.Setgid() {
		t.Error("allTrue: UserRead returns false")
	}
	if !allTrue.Sticky() {
		t.Error("allTrue: UserRead returns false")
	}
	if !allTrue.UserRead() {
		t.Error("allTrue: UserRead returns false")
	}
	if !allTrue.UserWrite() {
		t.Error("allTrue: UserWrite returns false")
	}
	if !allTrue.UserExecute() {
		t.Error("allTrue: UserExecute returns false")
	}
	if !allTrue.GroupRead() {
		t.Error("allTrue: GroupRead returns false")
	}
	if !allTrue.GroupWrite() {
		t.Error("allTrue: GroupWrite returns false")
	}
	if !allTrue.GroupExecute() {
		t.Error("allTrue: GroupExecute returns false")
	}
	if !allTrue.OtherRead() {
		t.Error("allTrue: OtherRead returns false")
	}
	if !allTrue.OtherWrite() {
		t.Error("allTrue: OtherWrite returns false")
	}
	if !allTrue.OtherExecute() {
		t.Error("allTrue: OtherExecute returns false")
	}

	if allTrue.String() != "rwxrwxrwx" {
		t.Error("allTrue: string incorrect")
	}

	allFalse := allTrue
	allFalse.SetSetuid(false)
	allFalse.SetSetgid(false)
	allFalse.SetSticky(false)
	allFalse.SetUserRead(false)
	allFalse.SetUserWrite(false)
	allFalse.SetUserExecute(false)
	allFalse.SetGroupRead(false)
	allFalse.SetGroupWrite(false)
	allFalse.SetGroupExecute(false)
	allFalse.SetOtherRead(false)
	allFalse.SetOtherWrite(false)
	allFalse.SetOtherExecute(false)
	allFalse.SetSetuid(false)
	allFalse.SetSetgid(false)
	allFalse.SetSticky(false)
	allFalse.SetUserRead(false)
	allFalse.SetUserWrite(false)
	allFalse.SetUserExecute(false)
	allFalse.SetGroupRead(false)
	allFalse.SetGroupWrite(false)
	allFalse.SetGroupExecute(false)
	allFalse.SetOtherRead(false)
	allFalse.SetOtherWrite(false)
	allFalse.SetOtherExecute(false)

	if allFalse.Setuid() {
		t.Error("allFalse: UserRead returns true")
	}
	if allFalse.Setgid() {
		t.Error("allFalse: UserRead returns true")
	}
	if allFalse.Sticky() {
		t.Error("allFalse: UserRead returns true")
	}
	if allFalse.UserRead() {
		t.Error("allFalse: UserRead returns true")
	}
	if allFalse.UserWrite() {
		t.Error("allFalse: UserWrite returns true")
	}
	if allFalse.UserExecute() {
		t.Error("allFalse: UserExecute returns true")
	}
	if allFalse.GroupRead() {
		t.Error("allFalse: GroupRead returns true")
	}
	if allFalse.GroupWrite() {
		t.Error("allFalse: GroupWrite returns true")
	}
	if allFalse.GroupExecute() {
		t.Error("allFalse: GroupExecute returns true")
	}
	if allFalse.OtherRead() {
		t.Error("allFalse: OtherRead returns true")
	}
	if allFalse.OtherWrite() {
		t.Error("allFalse: OtherWrite returns true")
	}
	if allFalse.OtherExecute() {
		t.Error("allFalse: OtherExecute returns true")
	}
	if allFalse.String() != "---------" {
		t.Error("allFalse: string incorrect")
	}
}

func TestErrors(t *testing.T) {
	_, err := Stat("./file_does_not_exist")
	if err == nil {
		t.Error("Stat did not fail on missing file")
	}
	err = Chmod("./file_does_not_exist", 0777)
	if err == nil {
		t.Error("Chmod did not fail on missing file")
	}
}
