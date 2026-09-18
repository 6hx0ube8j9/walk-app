package main

import (
	"log"
	"runtime"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

func init() {
	// 关键：将当前 goroutine 永久锁死在主 OS 线程，防止 Win32 消息循环失效
	runtime.LockOSThread()
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
		log.Fatalf("MainWindow 创建失败: %v", err)
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

	// 初始化托盘
	ni, err := walk.NewNotifyIcon()
	if err != nil {
		log.Fatalf("NotifyIcon 创建失败: %v", err)
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

	// 关键防护：确保 ContextMenu 已初始化，防止 nil pointer panic
	menu := ni.ContextMenu()
	if menu == nil {
		menu, err = walk.NewMenu()
		if err != nil {
			log.Fatalf("ContextMenu 创建失败: %v", err)
		}
		ni.SetContextMenu(menu)
	}

	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		isExiting = true
		mw.Close()
		win.PostQuitMessage(0)
	})
	menu.Actions().Add(exitAction)

	if err := ni.SetVisible(true); err != nil {
		log.Fatalf("托盘图标显示失败: %v", err)
	}

	// Win32 标准消息循环
	var msg win.MSG
	for {
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == 0 || ret == -1 {
			break // 0 表示接收到 WM_QUIT，-1 表示异常
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
}
