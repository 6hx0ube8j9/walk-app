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
	// 1. 单实例互斥检查
	lock, ok := AcquireSingleInstance(MutexName, AppName)
	if !ok {
		return
	}
	defer lock.Release()

	// 2. 初始化 Walk 应用
	app, err := walk.InitApp()
	if err != nil {
		log.Fatalf("初始化 App 失败: %v", err)
	}

	// 3. 声明并创建窗口
	var mw *walk.MainWindow
	err = MainWindow{
		AssignTo: &mw,
		Title:    AppName,
		MinSize:  Size{Width: 360, Height: 240},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "程序已进入后台保护运行。\n点击右上角 X 会直接隐藏到托盘。"},
		},
	}.Create()
	if err != nil {
		log.Fatalf("创建窗口失败: %v", err)
	}

	// 启动时隐藏主面板
	mw.Hide()

	// 4. 挂载托盘与消息托管器
	tray, err := SetupTrayManager(app, mw, AppName)
	if err != nil {
		log.Fatalf("初始化托盘失败: %v", err)
	}
	defer tray.Exit()

	// 5. 进入主事件循环
	app.Run()
}
