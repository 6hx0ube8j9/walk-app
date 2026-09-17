package main

import (
	"log"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

func main() {
	// Tailscale fork 使用 walk.InitApp() 初始化
	app, err := walk.InitApp()
	if err != nil {
		log.Fatal(err)
	}
	defer app.Exit(0)

	var mw *walk.MainWindow
	var ni *walk.NotifyIcon

	// 创建主窗口
	if err := (MainWindow{
		AssignTo: &mw,
		Title:    "简易窗口",
		MinSize:  Size{Width: 300, Height: 150},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "确定",
						OnClicked: func() {
							mw.Hide()
						},
					},
					PushButton{
						Text: "取消",
						OnClicked: func() {
							mw.Hide()
						},
					},
				},
			},
		},
	}).Create(); err != nil {
		log.Fatal(err)
	}

	// 启动时隐藏窗口
	mw.Hide()

	// 创建系统托盘图标（Tailscale fork 的参数为 MainWindow 传入）
	ni, err = walk.NewNotifyIcon(mw)
	if err != nil {
		log.Fatal(err)
	}
	defer ni.Dispose()

	if err := ni.SetToolTip("点击打开简易程序"); err != nil {
		log.Fatal(err)
	}

	// 使用应用自带的默认图标
	if err := ni.SetIcon(mw.Icon()); err != nil {
		log.Fatal(err)
	}

	// 点击托盘左键显示窗口
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			mw.Show()
			mw.SetFocus()
		}
	})

	// 右键菜单：退出
	exitAction := walk.NewAction()
	exitAction.SetText("退出")
	exitAction.Triggered().Attach(func() {
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	if err := ni.SetVisible(true); err != nil {
		log.Fatal(err)
	}

	// 运行应用主循环
	app.Run()
}
