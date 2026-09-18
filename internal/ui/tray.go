package ui

import (
	"syscall"
	"unsafe"

	"myapp/internal/core"

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
	app        *walk.Application
	mw         *walk.MainWindow
	ni         *walk.NotifyIcon
	oldWndProc uintptr
}

func SetupTrayManager(app *walk.Application, mw *walk.MainWindow, svc *core.Service, toolTip string) (*TrayManager, error) {
	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return nil, err
	}

	tm := &TrayManager{app: app, mw: mw, ni: ni}
	ni.SetToolTip(toolTip)
	ni.SetIcon(walk.IconInformation())

	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			tm.ToggleWindow()
		}
	})

	showAction := walk.NewAction()
	showAction.SetText("显示/隐藏主界面")
	showAction.Triggered().Attach(tm.ToggleWindow)
	ni.ContextMenu().Actions().Add(showAction)

	// 托盘右键直接触发业务强类型方法
	syncAction := walk.NewAction()
	syncAction.SetText("同步数据 (托盘触发)")
	syncAction.Triggered().Attach(func() { _ = svc.SyncData() })
	ni.ContextMenu().Actions().Add(syncAction)

	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(tm.Exit)
	ni.ContextMenu().Actions().Add(exitAction)

	// 底层截胡：点击 X 只做隐藏
	newWndProc := syscall.NewCallback(func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case win.WM_CLOSE:
			win.ShowWindow(hwnd, win.SW_HIDE)
			return 0
		case win.WM_QUERYENDSESSION:
			return 1
		case win.WM_ENDSESSION:
			if wParam != 0 {
				tm.Exit()
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

	tm.oldWndProc = win.SetWindowLongPtr(mw.Handle(), win.GWLP_WNDPROC, newWndProc)

	if err := ni.SetVisible(true); err != nil {
		ni.Dispose()
		return nil, err
	}
	return tm, nil
}

func (tm *TrayManager) ToggleWindow() {
	if tm.mw.Visible() {
		tm.mw.Hide()
	} else {
		tm.mw.Show()
		win.ShowWindow(tm.mw.Handle(), win.SW_RESTORE)
		win.SetForegroundWindow(tm.mw.Handle())
	}
}

func (tm *TrayManager) Exit() {
	if tm.ni != nil {
		tm.ni.SetVisible(false)
		tm.ni.Dispose()
	}
	if tm.oldWndProc != 0 && tm.mw != nil {
		win.SetWindowLongPtr(tm.mw.Handle(), win.GWLP_WNDPROC, tm.oldWndProc)
	}
	if tm.mw != nil {
		tm.mw.Close()
	}
	tm.app.Exit(0)
}
