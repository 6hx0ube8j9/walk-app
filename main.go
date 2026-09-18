package main

import (
	"fmt"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-s -w" -o app_b.exe main.go

func main() {

	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("程序发生致命崩溃 (Panic):\n%v\n\n堆栈信息:\n%s", r, string(debug.Stack()))
			win.MessageBox(0, syscall.StringToUTF16Ptr(errStr), syscall.StringToUTF16Ptr("致命错误"), win.MB_ICONERROR|win.MB_TOPMOST)
		}
	}()
	// ==================================================

	app, err := walk.InitApp()
	if err != nil {
		return
	}
	defer app.Exit(0)

	var mw *walk.MainWindow
	err = MainWindow{
		AssignTo: &mw,
		Title:    "流派B - 终极捕获版",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "如果这次再关闭，一定会弹窗告诉你具体的报错代码行！"},
		},
	}.Create()

	if err != nil {
		return
	}

	var isExiting bool

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !isExiting {
			*canceled = true

			go func() {
				time.Sleep(50 * time.Millisecond)
				mw.Synchronize(func() {
					mw.Hide()
				})
			}()
		}
	})
	// ------------------------------------------------

	mw.Hide()

	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("Mihomo Tray 稳定版")
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
		isExiting = true 
		ni.Dispose()
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)
	ni.SetVisible(true)

	app.Run()
}
