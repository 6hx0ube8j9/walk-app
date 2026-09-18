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

	// 单击托盘：纯 UI 视窗内部切换显隐，不走管道
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			engine.ToggleWindow()
		}
	})

	showAction := walk.NewAction()
	showAction.SetText("显示/隐藏主界面")
	showAction.Triggered().Attach(engine.ToggleWindow)
	ni.ContextMenu().Actions().Add(showAction)

	// 托盘右键抛出业务意图
	syncAction := walk.NewAction()
	syncAction.SetText("同步数据 (托盘触发)")
	syncAction.Triggered().Attach(func() {
		engine.cmdCh <- types.UICommand{Action: "sync_data"}
	})
	ni.ContextMenu().Actions().Add(syncAction)

	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	// 退出程序是业务意图，走管道由 Core 处理收尾
	exitAction := walk.NewAction()
	exitAction.SetText("彻底退出")
	exitAction.Triggered().Attach(func() {
		engine.cmdCh <- types.UICommand{Action: "exit_app"}
	})
	ni.ContextMenu().Actions().Add(exitAction)

	// 底层截胡：点击 X 属于纯视窗行为，UI 内部消化隐藏
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

	tm.oldWndProc = win.SetWindowLongPtr(engine.view.Window.Handle(), win.GWLP_WNDPROC, newWndProc)

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
