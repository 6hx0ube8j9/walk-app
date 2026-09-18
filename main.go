package main

import (
	"syscall"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-H windowsgui" -o app_b.exe main.go

func main() {
	app, err := walk.InitApp()
	if err != nil {
		return
	}

	var mw *walk.MainWindow

	err = MainWindow{
		AssignTo: &mw,
		Title:    "单进程后台常驻",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "点击右上角 X 会直接隐藏到托盘，绝不退出"},
		},
	}.Create()

	if err != nil {
		return
	}

	// 核心修复：通过底层 Win32 Subclass 强行拦截 WM_CLOSE 消息
	var oldWndProc uintptr
	newWndProc := syscall.NewCallback(func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
		if msg == win.WM_CLOSE {
			win.ShowWindow(hwnd, win.SW_HIDE) // 拦截关闭，直接隐藏窗口
			return 0                          // 吃掉消息，阻止 Win32 继续分发销毁窗体
		}
		return win.CallWindowProc(oldWndProc, hwnd, msg, wParam, lParam)
	})
	oldWndProc = win.SetWindowLongPtr(mw.Handle(), win.GWLP_WNDPROC, newWndProc)

	// 初始隐藏
	mw.Hide()

	// 托盘初始化
	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("后台常驻程序")
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

	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		// 真正退出时，先解除钩子再退出
		win.SetWindowLongPtr(mw.Handle(), win.GWLP_WNDPROC, oldWndProc)
		mw.Close()
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	ni.SetVisible(true)

	app.Run()
}
