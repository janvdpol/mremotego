//go:build windows
// +build windows

package launcher

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// hideConsoleWindow sets the command attributes to hide console windows on Windows
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

const credTag = "Managed by MRemoteGo"

// credentialW mirrors the Win32 CREDENTIALW struct layout.
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

// writeWindowsCredential stores a generic credential in Windows Credential Manager
// using CredWriteW directly, avoiding cmdkey's argument-parsing issues with
// usernames or passwords that contain spaces or special characters.
// The Comment field is set to credTag so orphaned entries can be found on startup.
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
	commentPtr, err := syscall.UTF16PtrFromString(credTag)
	if err != nil {
		return fmt.Errorf("invalid comment: %w", err)
	}

	// Encode password as UTF-16LE bytes without null terminator, as expected by
	// Windows Credential Manager for RDP credentials consumed by mstsc.
	passUTF16, err := syscall.UTF16FromString(password)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	passUTF16 = passUTF16[:len(passUTF16)-1] // strip null terminator
	passBlobSize := uint32(len(passUTF16) * 2)

	var blobPtr uintptr
	if passBlobSize > 0 {
		blobPtr = uintptr(unsafe.Pointer(&passUTF16[0]))
	}

	cred := credentialW{
		Type:               2, // CRED_TYPE_DOMAIN_PASSWORD — required for mstsc to find credentials
		TargetName:         targetPtr,
		Comment:            commentPtr,
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

// cleanupOrphanedCredentials removes any Credential Manager entries that were
// written by mremotego (identified by Comment == credTag) but never cleaned up,
// e.g. because the process crashed.
func cleanupOrphanedCredentials() {
	advapi32 := syscall.NewLazyDLL("advapi32.dll")
	credEnumerateW := advapi32.NewProc("CredEnumerateW")
	credFree := advapi32.NewProc("CredFree")

	var count uint32
	var credArrayPtr uintptr
	ret, _, _ := credEnumerateW.Call(0, 0, uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&credArrayPtr)))
	if ret == 0 {
		return // no credentials or access error
	}
	defer credFree.Call(credArrayPtr)

	ptrSize := unsafe.Sizeof(uintptr(0))
	for i := uint32(0); i < count; i++ {
		credPtr := *(*uintptr)(unsafe.Pointer(credArrayPtr + uintptr(i)*ptrSize))
		if credPtr == 0 {
			continue
		}
		cred := (*credentialW)(unsafe.Pointer(credPtr))
		if cred.Comment == nil {
			continue
		}
		comment := syscall.UTF16ToString(unsafe.Slice(cred.Comment, 256))
		if comment != credTag {
			continue
		}
		if cred.TargetName != nil {
			target := syscall.UTF16ToString(unsafe.Slice(cred.TargetName, 1024))
			_ = deleteWindowsCredential(target)
		}
	}
}

// deleteWindowsCredential removes a generic credential from Windows Credential Manager.
// Returns nil if the credential does not exist.
func deleteWindowsCredential(target string) error {
	credDeleteW := syscall.NewLazyDLL("advapi32.dll").NewProc("CredDeleteW")

	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("invalid target name: %w", err)
	}

	const credTypeDomainPassword = 2
	const errorNotFound = syscall.Errno(1168) // ERROR_NOT_FOUND
	ret, _, callErr := credDeleteW.Call(uintptr(unsafe.Pointer(targetPtr)), credTypeDomainPassword, 0)
	if ret == 0 && callErr != errorNotFound {
		return fmt.Errorf("CredDeleteW failed: %w", callErr)
	}
	return nil
}

// enumWindowsCB is created exactly once (syscall.NewCallback has a per-process limit of 1024).
var (
	enumWindowsOnce  sync.Once
	enumWindowsCB    uintptr
	enumWindowsMu    sync.Mutex
	enumWindowsPID   int
	enumWindowsFound bool
)

// waitForMstscConnected polls until mstsc for the given PID shows a
// TscShellContainerClass window (active RDP session), mstsc exits (exited
// closed), or timeout expires — whichever comes first.
func waitForMstscConnected(pid int, timeout time.Duration, exited <-chan struct{}) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-exited:
			return
		case t := <-ticker.C:
			if !t.Before(deadline) {
				return
			}
			if hasTscShellWindow(pid) {
				return
			}
		}
	}
}

// hasTscShellWindow returns true if there is a top-level window with class
// "TscShellContainerClass" belonging to the given process, which mstsc creates
// once the RDP session shell is ready.
func hasTscShellWindow(pid int) bool {
	enumWindowsOnce.Do(func() {
		user32 := syscall.NewLazyDLL("user32.dll")
		getWindowThreadProcessId := user32.NewProc("GetWindowThreadProcessId")
		getClassNameW := user32.NewProc("GetClassNameW")
		enumWindowsCB = syscall.NewCallback(func(hwnd syscall.Handle, _ uintptr) uintptr {
			var winPID uint32
			getWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&winPID)))
			if int(winPID) != enumWindowsPID {
				return 1 // continue
			}
			buf := make([]uint16, 64)
			getClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
			if syscall.UTF16ToString(buf) == "TscShellContainerClass" {
				enumWindowsFound = true
				return 0 // stop enumeration
			}
			return 1 // continue
		})
	})

	// EnumWindows calls the callback synchronously in the same goroutine, so
	// protecting enumWindowsPID/enumWindowsFound with a mutex is sufficient.
	enumWindowsMu.Lock()
	defer enumWindowsMu.Unlock()
	enumWindowsPID = pid
	enumWindowsFound = false
	user32 := syscall.NewLazyDLL("user32.dll")
	enumWindows := user32.NewProc("EnumWindows")
	enumWindows.Call(enumWindowsCB, 0)
	return enumWindowsFound
}
