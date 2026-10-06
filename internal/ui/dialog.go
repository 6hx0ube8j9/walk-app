package ui

import (
	"runtime"
	"syscall"

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
	whCBT        = 5
	hcbtActivate = 5
)

// showCenteredMsgBox 核心：使用 CBT 钩子将原生模态弹窗居中至指定的 target 窗口或面板控件
func showCenteredMsgBox(target walk.Widget, title, message string, style walk.MsgBoxStyle) int {
	// 锁定当前 OS 线程，确保 Hook 与 MessageBox 处于同一系统线程
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var hHook uintptr
	var targetHWND win.HWND

	if target != nil {
		targetHWND = target.Handle()
	}

	hookCallback := syscall.NewCallback(func(nCode int32, wParam uintptr, lParam uintptr) uintptr {
		if nCode == hcbtActivate && targetHWND != 0 {
			msgBoxHwnd := win.HWND(wParam)

			var targetRect, msgBoxRect win.RECT
			win.GetWindowRect(targetHWND, &targetRect)
			win.GetWindowRect(msgBoxHwnd, &msgBoxRect)

			targetW := targetRect.Right - targetRect.Left
			targetH := targetRect.Bottom - targetRect.Top
			dlgW := msgBoxRect.Right - msgBoxRect.Left
			dlgH := msgBoxRect.Bottom - msgBoxRect.Top

			// 精确计算相对 target 的居中坐标
			x := targetRect.Left + (targetW-dlgW)/2
			y := targetRect.Top + (targetH-dlgH)/2

			win.SetWindowPos(msgBoxHwnd, 0, x, y, 0, 0, win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOACTIVATE)

			// 调整完成后立即注销钩子
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

	var ownerForm walk.Form
	if target != nil {
		ownerForm = target.Form()
	}

	// 调用原生 walk.MsgBox，具备 Windows 内核对 Tab/Enter/Esc 的原生响应支持
	result := walk.MsgBox(ownerForm, title, message, style)

	// 兜底注销
	if hHook != 0 {
		procUnhookWindowsHookEx.Call(hHook)
	}

	return result
}

// ShowConfirmDialog 确认提示弹窗（支持传入 MainWindow 或其子控件 TableView）
func ShowConfirmDialog(target walk.Widget, title, message string) bool {
	res := showCenteredMsgBox(target, title, message, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
	return res == win.IDYES
}

// ShowErrorDialog 错误提示弹窗
func ShowErrorDialog(target walk.Widget, title, message string) {
	showCenteredMsgBox(target, title, message, walk.MsgBoxOK|walk.MsgBoxIconError)
}

// 兼容别名
func RunConfirmDialog(target walk.Widget, title, message string) bool {
	return ShowConfirmDialog(target, title, message)
}

func RunErrorDialog(target walk.Widget, title, message string) {
	ShowErrorDialog(target, title, message)
}
