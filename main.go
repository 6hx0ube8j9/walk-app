package main

import (
	"os"
	"os/exec"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

//go:generate go build -ldflags="-H windowsgui" -o app.exe main.go

func main() {
	// 区分当前是“托盘后台”还是“弹出的面板”
	if len(os.Args) > 1 && os.Args[1] == "--panel" {
		runPanel() // 运行面板进程
	} else {
		runTray()  // 运行托盘常驻进程
	}
}

// ================= 1. 独立面板进程 =================
func runPanel() {
	app, err := walk.InitApp()
	if err != nil {
		return
	}
	defer app.Exit(0)

	var mw *walk.MainWindow

	// 创建面板窗口：自带最大化、最小化、关闭(X)按钮
	err = (MainWindow{
		AssignTo: &mw,
		Title:    "独立的配置面板",
		MinSize:  Size{Width: 350, Height: 250},
		Layout:   VBox{},
		Children: []Widget{
			PushButton{
				Text: "模拟面板崩溃/直接退出",
				OnClicked: func() {
					os.Exit(1) // 模拟面板异常退出，托盘绝对不会死！
				},
			},
		},
	}).Create()
	if err != nil {
		return
	}

	// 这里的 X 关闭是真正的窗口销毁和进程退出
	mw.Show()
	app.Run()
}

// ================= 2. 后台托盘常驻进程 =================
func runTray() {
	app, err := walk.InitApp()
	if err != nil {
		return
	}
	defer app.Exit(0)

	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("双进程架构：托盘常驻")
	ni.SetIcon(walk.IconInformation())

	// 左键点击托盘：拉起一个全新的面板子进程
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			// 用自身 exe 加上 --panel 参数启动独立子进程
			cmd := exec.Command(os.Args[0], "--panel")
			_ = cmd.Start() 
		}
	})

	// 右键菜单：退出整个程序（连同托盘一起关掉）
	exitAction := walk.NewAction()
	exitAction.SetText("退出主程序")
	exitAction.Triggered().Attach(func() {
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)
	ni.SetVisible(true)

	app.Run()
}
