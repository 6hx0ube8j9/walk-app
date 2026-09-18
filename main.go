package main

import (
	"runtime"
	"syscall"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-H windowsgui" -o app_b.exe main.go

func init() {
	// 关键：将当前协程锁定在主 OS 线程，防止 Win32 消息循环因线程切换失效
	runtime.LockOSThread()
}

// 错误辅助函数：在 GUI 模式下遇到致命错误弹出窗口提示，代替静默 return
func showError(msg string) {
	captionPtr, _ := syscall.UTF16PtrFromString("启动异常")
	msgPtr, _ := syscall.UTF16PtrFromString(msg)
	win.MessageBox(0, msgPtr, captionPtr, win.MB_ICONERROR|win.MB_OK)
}

func main() {
	var mw *walk.MainWindow

	err := MainWindow{
		AssignTo: &mw,
		Title:    "Walk 单进程",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "点击右上角 X 会自动隐藏到托盘，绝不会退出"},
		},
	}.Create()

	if err != nil {
		showError("MainWindow 创建失败: " + err.Error())
		return
	}

	var isExiting bool

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !isExiting {
			*canceled = true
			mw.Hide()
		}
	})

	// 启动时隐藏主窗口
	mw.Hide()

	ni, err := walk.NewNotifyIcon()
	if err != nil {
		showError("NotifyIcon 创建失败: " + err.Error())
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("Tailscale Walk 托盘常驻")
	ni.SetIcon(walk.IconInformation())

	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if mw.Visible() {
				mw.Hide()
			} else {
				mw.Show()
				win.ShowWindow(mw.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(mw.Handle())
			}
		}
	})

	// 直接从已有 ContextMenu 中添加 Action
	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		isExiting = true
		mw.Close()
		win.PostQuitMessage(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	if err := ni.SetVisible(true); err != nil {
		showError("托盘图标设置可见失败: " + err.Error())
		return
	}

	// Win32 标准消息循环
	var msg win.MSG
	for {
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == 0 || ret == -1 {
			break
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
}
