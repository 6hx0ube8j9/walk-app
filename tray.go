package main

import (
	"syscall"
	"unsafe"

	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

var (
	modUser32             = syscall.NewLazyDLL("user32.dll")
	procRegisterWindowMsg = modUser32.NewProc("RegisterWindowMessageW")
	msgTaskbarCreated     uint32
)

func init() {
	taskbarStr, _ := syscall.UTF16PtrFromString("TaskbarCreated")
	ret, _, _ := procRegisterWindowMsg.Call(uintptr(unsafe.Pointer(taskbarStr)))
	msgTaskbarCreated = uint32(ret)
}

type TrayManager struct {
	app        *App
	ni         *walk.NotifyIcon
	oldWndProc uintptr
}

func SetupTrayManager(app *App, toolTip string) (*TrayManager, error) {
	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return nil, err
	}

	tm := &TrayManager{
		app: app,
		ni:  ni,
	}

	ni.SetToolTip(toolTip)
	ni.SetIcon(walk.IconInformation())

	// 单击托盘切换面板显隐
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			tm.ToggleWindow()
		}
	})

	// 托盘右键菜单
	showAction := walk.NewAction()
	showAction.SetText("显示/隐藏主界面")
	showAction.Triggered().Attach(tm.ToggleWindow)
	ni.ContextMenu().Actions().Add(showAction)

	// 托盘菜单派发业务意图
	syncAction := walk.NewAction()
	syncAction.SetText("同步数据 (托盘触发)")
	syncAction.Triggered().Attach(func() {
		app.HandleOperation("sync_data")
	})
	ni.ContextMenu().Actions().Add(syncAction)

	exportAction := walk.NewAction()
	exportAction.SetText("导出报表 (托盘触发)")
	exportAction.Triggered().Attach(func() {
		app.HandleOperation("export_report")
	})
	ni.ContextMenu().Actions().Add(exportAction)

	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(tm.Exit)
	ni.ContextMenu().Actions().Add(exitAction)

	// Win32 底层截胡消息循环：拦截点击 X
	newWndProc := syscall.NewCallback(func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case win.WM_CLOSE:
			win.ShowWindow(hwnd, win.SW_HIDE) // 拦截关闭，直接平滑隐藏
			return 0

		case win.WM_QUERYENDSESSION:
			return 1 // 允许 Windows 正常关机

		case win.WM_ENDSESSION:
			if wParam != 0 {
				tm.Exit()
				return 0
			}

		default:
			// 资源管理器崩溃重启后自愈重绘托盘
			if msgTaskbarCreated != 0 && msg == msgTaskbarCreated {
				tm.ni.SetVisible(false)
				tm.ni.SetVisible(true)
			}
		}
		return win.CallWindowProc(tm.oldWndProc, hwnd, msg, wParam, lParam)
	})

	tm.oldWndProc = win.SetWindowLongPtr(app.MW.Handle(), win.GWLP_WNDPROC, newWndProc)

	if err := ni.SetVisible(true); err != nil {
		ni.Dispose()
		return nil, err
	}

	return tm, nil
}

func (tm *TrayManager) ToggleWindow() {
	if tm.app.MW.Visible() {
		tm.app.MW.Hide()
	} else {
		tm.app.MW.Show()
		win.ShowWindow(tm.app.MW.Handle(), win.SW_RESTORE)
		win.SetForegroundWindow(tm.app.MW.Handle())
	}
}

func (tm *TrayManager) Exit() {
	if tm.ni != nil {
		tm.ni.SetVisible(false)
		tm.ni.Dispose()
	}
	if tm.oldWndProc != 0 && tm.app.MW != nil {
		win.SetWindowLongPtr(tm.app.MW.Handle(), win.GWLP_WNDPROC, tm.oldWndProc)
	}
	if tm.app.MW != nil {
		tm.app.MW.Close()
	}
	tm.app.WalkApp.Exit(0)
}
