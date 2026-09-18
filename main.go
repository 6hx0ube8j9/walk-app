package main

import (
	"log"
	"runtime"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

//go:generate go build -ldflags="-H windowsgui -s -w" -o app_b.exe .

const (
	AppName   = "生产级后台常驻程序"
	MutexName = "Global\\MyProductionTrayApp_SingleInstance_Mutex"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	lock, ok := AcquireSingleInstance(MutexName, AppName)
	if !ok {
		return
	}
	defer lock.Release()

	app, err := walk.InitApp()
	if err != nil {
		log.Fatalf("初始化 App 失败: %v", err)
	}

	var mw *walk.MainWindow
	err = MainWindow{
		AssignTo: &mw,
		Title:    AppName,
		MinSize:  Size{Width: 360, Height: 240},
		Layout:   VBox{Margins: Margins{Top: 20, Bottom: 20, Left: 20, Right: 20}, Spacing: 12},
		Children: []Widget{
			Label{Text: "程序已进入后台保护运行。\n点击右上角 X 会直接隐藏到托盘。"},
			VSpacer{Size: 10},
			PushButton{
				Text: "测试错误弹窗",
				OnClicked: func() {
					ShowErrorDialog(mw, "界面错误", "这是一条由主界面按钮触发的异常提示！")
				},
			},
		},
	}.Create()
	if err != nil {
		log.Fatalf("创建窗口失败: %v", err)
	}

	mw.Hide()

	tray, err := SetupTrayManager(app, mw, AppName)
	if err != nil {
		log.Fatalf("初始化托盘失败: %v", err)
	}
	defer tray.Exit()

	app.Run()
}
