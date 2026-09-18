package main

import (
	"log"
	"runtime"

	"walk-app/internal/core"
	"walk-app/internal/platform"
	"walk-app/internal/ui"

	"github.com/tailscale/walk"
)

//go:generate go build -ldflags="-H windowsgui -s -w" -o app.exe .

const (
	AppName   = "生产级常驻程序"
	MutexName = "Global\\MyApp_SingleInstance_Mutex"
)

func init() {
	// 锁定系统主线程，保障 Windows 消息循环安全
	runtime.LockOSThread()
}

func main() {
	// 1. 防多开检查（若已存在实例，则唤醒置顶并退出当前进程）
	lock, ok := platform.AcquireSingleInstance(MutexName, AppName)
	if !ok {
		return
	}
	defer lock.Release()

	// 2. 初始化底层 GUI 运行时
	walkApp, err := walk.InitApp()
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	// 3. 构建弹窗桥接器与业务大脑（依赖注入）
	presenter := ui.NewWalkPresenter()
	svc := core.NewService(presenter)

	// 4. 构建主窗口并绑定桥接器
	mw, err := ui.CreateMainWindow(svc, AppName)
	if err != nil {
		log.Fatalf("创建主窗口失败: %v", err)
	}
	presenter.SetWindow(mw)
	mw.Hide() // 初始状态隐藏，常驻托盘

	// 5. 挂载托盘与 Win32 底层消息拦截（X 隐藏、关机放行、Explorer 崩溃自愈）
	tray, err := ui.SetupTrayManager(walkApp, mw, svc, AppName)
	if err != nil {
		log.Fatalf("初始化托盘失败: %v", err)
	}
	defer tray.Exit()

	// 6. 启动 Windows 消息主循环
	walkApp.Run()
}
