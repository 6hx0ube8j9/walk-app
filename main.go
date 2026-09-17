package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

func main() {
	var mw *walk.MainWindow
	var ni *walk.NotifyIcon

	// 初始化 Walk GUI
	walk.Initialize()
	defer walk.Shutdown()

	// 创建隐藏的主窗口（作为托盘宿主和控制中心）
	if err := (MainWindow{
		AssignTo: &mw,
		Title:    "简易窗口",
		MinSize:  Size{Width: 300, Size: Size{Height: 150}, Height: 150},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "确定",
						OnClicked: func() {
							mw.Hide() // 点击确定后隐藏窗口
						},
					},
					PushButton{
						Text: "取消",
						OnClicked: func() {
							mw.Hide() // 点击取消后隐藏窗口
						},
					},
				},
			},
		},
	}).Create(); err != nil {
		panic(err)
	}

	// 隐藏主窗口，不让它在启动时直接显示
	mw.Hide()

	// 创建系统托盘图标
	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		panic(err)
	}
	defer ni.Dispose()

	// 设置托盘图标提示文字
	if err := ni.SetToolTip("点击打开简易程序"); err != nil {
		panic(err)
	}

	// 这里使用内置的标准图标（也可以换成你自己的 .ico 文件）
	if err := ni.SetIcon(walk.IconStandardInformation()); err != nil {
		panic(err)
	}

	// 核心：点击托盘图标时打开 GUI 窗口
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			mw.Visible()
			mw.Show()
			mw.SetFocus()
		}
	})

	// 添加右键菜单：退出程序
	exitAction := walk.NewAction()
	exitAction.SetText("退出")
	exitAction.Triggered().Attach(func() {
		walk.App().Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	// 让托盘图标可见
	if err := ni.SetVisible(true); err != nil {
		panic(err)
	}

	// 运行主消息循环
	mw.Run()
}
