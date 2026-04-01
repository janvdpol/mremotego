//go:build windows
// +build windows

package launcher

import (
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"
)

// hideConsoleWindow sets the command attributes to hide console windows on Windows
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// writeWindowsCredential stores a generic credential in Windows Credential Manager
// using CredWriteW directly, avoiding cmdkey's argument-parsing issues with
// usernames or passwords that contain spaces or special characters.
func writeWindowsCredential(target, username, password string) error {
	credWriteW := syscall.NewLazyDLL("advapi32.dll").NewProc("CredWriteW")

	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("invalid target name: %w", err)
	}
	userPtr, err := syscall.UTF16PtrFromString(username)
	if err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}

	// Encode password as UTF-16LE bytes without null terminator, as expected by
	// Windows Credential Manager for RDP credentials consumed by mstsc.
	passUTF16, err := syscall.UTF16FromString(password)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	passUTF16 = passUTF16[:len(passUTF16)-1] // strip null terminator
	passBlobSize := uint32(len(passUTF16) * 2)

	// CREDENTIALW mirrors the Win32 CREDENTIALW struct layout.
	type credentialW struct {
		Flags              uint32
		Type               uint32
		TargetName         *uint16
		Comment            *uint16
		LastWritten        [2]uint32
		CredentialBlobSize uint32
		CredentialBlob     uintptr
		Persist            uint32
		AttributeCount     uint32
		Attributes         uintptr
		TargetAlias        *uint16
		UserName           *uint16
	}

	var blobPtr uintptr
	if passBlobSize > 0 {
		blobPtr = uintptr(unsafe.Pointer(&passUTF16[0]))
	}

	cred := credentialW{
		Type:               1, // CRED_TYPE_GENERIC
		TargetName:         targetPtr,
		UserName:           userPtr,
		CredentialBlobSize: passBlobSize,
		CredentialBlob:     blobPtr,
		Persist:            2, // CRED_PERSIST_LOCAL_MACHINE
	}

	ret, _, callErr := credWriteW.Call(uintptr(unsafe.Pointer(&cred)), 0)
	runtime.KeepAlive(passUTF16) // prevent GC until after the syscall
	if ret == 0 {
		return fmt.Errorf("CredWriteW failed: %w", callErr)
	}
	return nil
}

// deleteWindowsCredential removes a generic credential from Windows Credential Manager.
// Returns nil if the credential does not exist.
func deleteWindowsCredential(target string) error {
	credDeleteW := syscall.NewLazyDLL("advapi32.dll").NewProc("CredDeleteW")

	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("invalid target name: %w", err)
	}

	const credTypeGeneric = 1
	const errorNotFound = syscall.Errno(1168) // ERROR_NOT_FOUND
	ret, _, callErr := credDeleteW.Call(uintptr(unsafe.Pointer(targetPtr)), credTypeGeneric, 0)
	if ret == 0 && callErr != errorNotFound {
		return fmt.Errorf("CredDeleteW failed: %w", callErr)
	}
	return nil
}
