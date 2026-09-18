package main

import (
	"log"
	"runtime"
)

//go:generate go build -ldflags="-H windowsgui -s -w" -o app_b.exe .

const (
	AppName   = "生产级后台常驻程序"
	MutexName = "Global\\MyProductionTrayApp_SingleInstance_Mutex"
)

func init() {
	// 锁死主 OS 线程，保障 Win32 消息泵运转正常
	runtime.LockOSThread()
}

func main() {
	// 1. 单实例检查（防重复启动）
	lock, ok := AcquireSingleInstance(MutexName, AppName)
	if !ok {
		return
	}
	defer lock.Release()

	// 2. 初始化核心上下文
	app, err := NewApp()
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	// 3. 构建并挂载主窗口 GUI
	mw, err := CreateMainWindow(app, AppName)
	if err != nil {
		log.Fatalf("创建主窗口失败: %v", err)
	}
	app.MW = mw
	mw.Hide() // 初始静默启动到托盘

	// 4. 构建并挂载系统托盘
	tray, err := SetupTrayManager(app, AppName)
	if err != nil {
		log.Fatalf("初始化托盘失败: %v", err)
	}
	app.Tray = tray
	defer tray.Exit()

	// 5. 启动主消息循环
	app.Run()
}
