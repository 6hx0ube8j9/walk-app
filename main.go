package main

import (
	"log"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

func main() {
	// 初始化应用
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

	// 启动时隐藏主窗口
	mw.Hide()

	// 创建系统托盘图标（Tailscale fork 版本无需传参）
	ni, err = walk.NewNotifyIcon()
	if err != nil {
		log.Fatal(err)
	}
	defer ni.Dispose()

	if err := ni.SetToolTip("点击打开简易程序"); err != nil {
		log.Fatal(err)
	}

	// 使用窗口图标赋给托盘
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
