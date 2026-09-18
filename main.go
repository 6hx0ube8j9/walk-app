package main

import (
	"context"
	"log"
	"runtime"

	"walk-app/internal/core"
	"walk-app/internal/platform"
	"walk-app/internal/types"
	"walk-app/internal/ui"

	"github.com/tailscale/walk"
)

const (
	AppName   = "生产级常驻程序"
	MutexName = "Global\\WalkApp_SingleInstance_Mutex"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	// 1. 进程防多开
	lock, ok := platform.AcquireSingleInstance(MutexName, AppName)
	if !ok {
		return
	}
	defer lock.Release()

	// 2. 初始化底层 GUI 运行环境
	walkApp, err := walk.InitApp()
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	// 3. 构建 MVI 通信管道
	cmdCh := make(chan types.UICommand, 32)
	stateCh := make(chan types.UIState, 1)
	effectCh := make(chan types.UIEffect, 16)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. 启动核心大脑 Goroutine (0 GUI 依赖)
	svc := core.NewService(cmdCh, stateCh, effectCh)
	go svc.Run(ctx)

	// 5. 构建并挂载 UI 引擎（整合 Window + Tray + Hook）
	_, err := ui.NewUIEngine(walkApp, cmdCh, stateCh, effectCh, AppName)
	if err != nil {
		log.Fatalf("初始化 UI 失败: %v", err)
	}

	// 6. 运行主消息循环
	walkApp.Run()
}
