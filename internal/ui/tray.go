package ui

import (
	"syscall"
	"unsafe"
	"walk-app/internal/types"

	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

var (
	procRegisterWindowMsg = syscall.NewLazyDLL("user32.dll").NewProc("RegisterWindowMessageW")
	msgTaskbarCreated     uint32
)

func init() {
	taskbarStr, _ := syscall.UTF16PtrFromString("TaskbarCreated")
	ret, _, _ := procRegisterWindowMsg.Call(uintptr(unsafe.Pointer(taskbarStr)))
	msgTaskbarCreated = uint32(ret)
}

type TrayManager struct {
	ni         *walk.NotifyIcon
	oldWndProc uintptr
}

func SetupTray(engine *UIEngine, toolTip string) (*TrayManager, error) {
	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return nil, err
	}

	tm := &TrayManager{ni: ni}
	ni.SetToolTip(toolTip)
	ni.SetIcon(walk.IconInformation())

	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			engine.ToggleWindow()
		}
	})

	showAction := walk.NewAction()
	showAction.SetText("显示/隐藏主界面")
	showAction.Triggered().Attach(engine.ToggleWindow)
	ni.ContextMenu().Actions().Add(showAction)

	syncAction := walk.NewAction()
	syncAction.SetText("同步数据")
	syncAction.Triggered().Attach(func() {
		if engine.cmdCh != nil {
			engine.cmdCh <- types.UICommand{Action: "sync_data"}
		}
	})
	ni.ContextMenu().Actions().Add(syncAction)

	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	testErrorAction := walk.NewAction()
	testErrorAction.SetText("【测试】错误弹窗")
	testErrorAction.Triggered().Attach(func() {
		var parent walk.Form
		if engine.view != nil && engine.view.Window != nil {
			parent = engine.view.Window
		}
		ShowErrorDialog(parent, "系统错误", "这是一条测试错误提示信息！")
	})
	ni.ContextMenu().Actions().Add(testErrorAction)

	testConfirmAction := walk.NewAction()
	testConfirmAction.SetText("【测试】确认弹窗")
	testConfirmAction.Triggered().Attach(func() {
		var parent walk.Form
		if engine.view != nil && engine.view.Window != nil {
			parent = engine.view.Window
		}

		if ShowConfirmDialog(parent, "操作确认", "这是一个带提示音的测试弹窗，是否确认继续？") {
			if engine.cmdCh != nil {
				engine.cmdCh <- types.UICommand{Action: "test_confirm_yes"}
			}
		}
	})
	ni.ContextMenu().Actions().Add(testConfirmAction)

	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	exitAction := walk.NewAction()
	exitAction.SetText("彻底退出")
	exitAction.Triggered().Attach(func() {
		engine.Exit()
	})
	ni.ContextMenu().Actions().Add(exitAction)

	newWndProc := syscall.NewCallback(func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case win.WM_CLOSE:
			win.ShowWindow(hwnd, win.SW_HIDE)
			return 0
		case win.WM_QUERYENDSESSION:
			return 1
		case win.WM_ENDSESSION:
			if wParam != 0 {
				engine.Exit()
				return 0
			}
		default:
			if msgTaskbarCreated != 0 && msg == msgTaskbarCreated {
				tm.ni.SetVisible(false)
				tm.ni.SetVisible(true)
			}
		}
		return win.CallWindowProc(tm.oldWndProc, hwnd, msg, wParam, lParam)
	})

	if engine.view != nil && engine.view.Window != nil {
		tm.oldWndProc = win.SetWindowLongPtr(engine.view.Window.Handle(), win.GWLP_WNDPROC, newWndProc)
	}

	if err := ni.SetVisible(true); err != nil {
		ni.Dispose()
		return nil, err
	}
	return tm, nil
}

func (tm *TrayManager) Dispose(hwnd win.HWND) {
	if tm.ni != nil {
		tm.ni.SetVisible(false)
		tm.ni.Dispose()
	}
	if tm.oldWndProc != 0 && hwnd != 0 {
		win.SetWindowLongPtr(hwnd, win.GWLP_WNDPROC, tm.oldWndProc)
	}
}
