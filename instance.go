package main

import (
	"syscall"
	"unsafe"

	"github.com/tailscale/win"
)

var (
	modKernel32     = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = modKernel32.NewProc("CreateMutexW")
)

type InstanceLock struct {
	handle win.HANDLE
}

func AcquireSingleInstance(mutexName, windowTitle string) (*InstanceLock, bool) {
	const ERROR_ALREADY_EXISTS = 183
	namePtr, _ := syscall.UTF16PtrFromString(mutexName)

	hMutex, _, _ := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(namePtr)))
	if syscall.GetLastError() == syscall.Errno(ERROR_ALREADY_EXISTS) {
		if hMutex != 0 {
			win.CloseHandle(win.HANDLE(hMutex))
		}
		// 激活已有实例的窗口
		titlePtr, _ := syscall.UTF16PtrFromString(windowTitle)
		if hwnd := win.FindWindow(nil, titlePtr); hwnd != 0 {
			win.ShowWindow(hwnd, win.SW_RESTORE)
			win.SetForegroundWindow(hwnd)
		}
		return nil, false
	}

	return &InstanceLock{handle: win.HANDLE(hMutex)}, true
}

func (l *InstanceLock) Release() {
	if l != nil && l.handle != 0 {
		win.CloseHandle(l.handle)
		l.handle = 0
	}
}
