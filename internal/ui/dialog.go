package ui

import (
	"reflect"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

var (
	modUser32               = syscall.NewLazyDLL("user32.dll")
	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	procSetWindowsHookExW   = modUser32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = modUser32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = modUser32.NewProc("CallNextHookEx")
	procGetCurrentThreadId  = modKernel32.NewProc("GetCurrentThreadId")
)

const (
	whCBT          = 5
	hcbtActivate   = 5
	spiGetWorkArea = 0x0030
)

// resolveTarget 解析目标，提取 HWND 和所属 Form，兼容 walk.Form、walk.Widget 与 nil
func resolveTarget(target interface{}) (win.HWND, walk.Form) {
	if target == nil {
		return 0, nil
	}
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return 0, nil
	}

	switch t := target.(type) {
	case walk.Form:
		return t.Handle(), t
	case walk.Widget:
		return t.Handle(), t.Form()
	case interface{ Handle() win.HWND }:
		return t.Handle(), nil
	}
	return 0, nil
}

// showCenteredMsgBox 使用 CBT 钩子将原生模态弹窗居中到指定的窗口、面板控件或屏幕
func showCenteredMsgBox(target interface{}, title, message string, style walk.MsgBoxStyle) int {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	targetHWND, ownerForm := resolveTarget(target)
	var hHook uintptr

	hookCallback := syscall.NewCallback(func(nCode int32, wParam uintptr, lParam uintptr) uintptr {
		if nCode == hcbtActivate {
			msgBoxHwnd := win.HWND(wParam)

			var msgBoxRect win.RECT
			win.GetWindowRect(msgBoxHwnd, &msgBoxRect)
			dlgW := msgBoxRect.Right - msgBoxRect.Left
			dlgH := msgBoxRect.Bottom - msgBoxRect.Top

			var x, y int
