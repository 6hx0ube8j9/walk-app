package main

import (
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
	defer app.Exit(0)

	var mw *walk.MainWindow
	err = MainWindow{
		AssignTo: &mw,
		Title:    "流派B - 单进程显隐面板",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "点击右上角 X 会自动隐藏到托盘，而不是退出程序"},
		},
	}.Create()

	if err != nil {
		return
	}

	// ---------------- 终极防御机制 ----------------
	// 抛弃 walk 极度不可靠的 CloseReason，用自己的标志位判断是“最小化”还是“真退出”
	var isExiting bool

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !isExiting {
			*canceled = true // 拦截销毁

			// 核心：绝对不能在这里直接写 mw.Hide()，否则必引发静默崩溃（闪退）！
			// 必须用 Synchronize 异步执行隐藏
			mw.Synchronize(func() {
				mw.Hide()
			})
		}
	})
	// ----------------------------------------------

	// 启动时默认隐藏窗口，只留托盘
	mw.Hide()

	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose() // 正常流程销毁

	ni.SetToolTip("流派B - 单进程常驻")
	ni.SetIcon(walk.IconInformation())

	// 左键点击托盘：切换面板显隐状态
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if mw.Visible() {
				mw.Hide()
			} else {
				mw.Show()
				// 突破 Windows 防打扰机制，强行置顶前台
				win.ShowWindow(mw.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(mw.Handle())
			}
		}
	})

	// 右键菜单：真正退出程序
	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		isExiting = true // 标记为真退出，放行 Closing 拦截
		ni.Dispose()     // 手动提前销毁托盘，防止 Windows 任务栏留下幽灵图标
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)
	ni.SetVisible(true)

	app.Run()
}
