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

// resolveTarget 解析入参，提取 HWND 与 Form，兼容 walk.Form、walk.Widget 及 nil
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

// showCenteredMsgBox 通过 WH_CBT 钩子在窗口激活瞬间计算居中坐标
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

			var x, y int32
			if targetHWND != 0 && win.IsWindow(targetHWND) && win.IsWindowVisible(targetHWND) {
				var targetRect win.RECT
				win.GetWindowRect(targetHWND, &targetRect)
				targetW := targetRect.Right - targetRect.Left
				targetH := targetRect.Bottom - targetRect.Top
				x = targetRect.Left + (targetW-dlgW)/2
				y = targetRect.Top + (targetH-dlgH)/2
			} else {
				var workArea win.RECT
				if win.SystemParametersInfo(spiGetWorkArea, 0, unsafe.Pointer(&workArea), 0) {
					screenW := workArea.Right - workArea.Left
					screenH := workArea.Bottom - workArea.Top
					x = workArea.Left + (screenW-dlgW)/2
					y = workArea.Top + (screenH-dlgH)/2
				} else {
					screenW := win.GetSystemMetrics(win.SM_CXSCREEN)
					screenH := win.GetSystemMetrics(win.SM_CYSCREEN)
					x = (screenW - dlgW) / 2
					y = (screenH - dlgH) / 2
				}
			}

			win.SetWindowPos(msgBoxHwnd, 0, x, y, 0, 0, win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE)

			if hHook != 0 {
				procUnhookWindowsHookEx.Call(hHook)
				hHook = 0
			}
		}

		ret, _, _ := procCallNextHookEx.Call(hHook, uintptr(nCode), wParam, lParam)
		return ret
	})

	tid, _, _ := procGetCurrentThreadId.Call()
	hHook, _, _ = procSetWindowsHookExW.Call(uintptr(whCBT), hookCallback, 0, tid)

	result := walk.MsgBox(ownerForm, title, message, style)

	if hHook != 0 {
		procUnhookWindowsHookEx.Call(hHook)
	}

	return result
}

// ShowConfirmDialog 确认提示弹窗
func ShowConfirmDialog(target interface{}, title, message string) bool {
	res := showCenteredMsgBox(target, title, message, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
	return res == win.IDYES
}

// ShowErrorDialog 错误提示弹窗
func ShowErrorDialog(target interface{}, title, message string) {
	showCenteredMsgBox(target, title, message, walk.MsgBoxOK|walk.MsgBoxIconError)
}

// RunConfirmDialog 别名兼容
func RunConfirmDialog(target interface{}, title, message string) bool {
	return ShowConfirmDialog(target, title, message)
}

// RunErrorDialog 别名兼容
func RunErrorDialog(target interface{}, title, message string) {
	ShowErrorDialog(target, title, message)
}
