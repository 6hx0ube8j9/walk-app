package main

import (
	"context"
	"flag"
	"log"
	"os"
	"runtime"
	"syscall"
	"unsafe"

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

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procAllocConsole    = kernel32.NewProc("AllocConsole")
	procAttachConsole   = kernel32.NewProc("AttachConsole")
	procSetConsoleTitle = kernel32.NewProc("SetConsoleTitleW")
)

const attachParentProcess = ^uintptr(0) // (DWORD)-1

// setupDebugConsole 激活控制台并重定向输出流，兼容 -H=windowsgui 模式
func setupDebugConsole(title string) {
	// 1. 优先附着到调用它的终端（CMD / PowerShell）
	r, _, _ := procAttachConsole.Call(attachParentProcess)
	if r == 0 {
		// 2. 若不是从终端启动（例如直接双击 EXE），分配独立的控制台窗口
		procAllocConsole.Call()
	}

	if title != "" {
		t, _ := syscall.UTF16PtrFromString(title)
		procSetConsoleTitle.Call(uintptr(unsafe.Pointer(t)))
	}

	// 3. 强制重定向 os.Stdout、os.Stderr 和 log 至当前控制台输出设备
	conout, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0644)
	if err == nil {
		os.Stdout = conout
		os.Stderr = conout
		log.SetOutput(conout)
		log.SetFlags(log.Ltime | log.Lmicroseconds)
	}
}

func init() {
	runtime.LockOSThread()
}

func main() {
	// 命令行参数支持：默认开启控制台监听，可通过 -nolog 关闭
	noLog := flag.Bool("nolog", false, "关闭控制台调试日志")
	flag.Parse()

	if !*noLog {
		setupDebugConsole(AppName + " [实时日志控制台]")
	}

	log.Println(">>> 进程启动，正在检查单例锁...")

	// 1. 进程防多开
	lock, ok := platform.AcquireSingleInstance(MutexName, AppName)
	if !ok {
		log.Println("[WARN] 已存在运行中的实例，退出当前进程")
		return
	}
	defer lock.Release()
	log.Println("[OK] 单例锁获取成功")

	// 2. 初始化底层 GUI 运行环境
	walkApp, err := walk.InitApp()
	if err != nil {
		log.Fatalf("[FATAL] 初始化应用失败: %v", err)
	}
	log.Println("[OK] Walk GUI 引擎初始化完毕")

	// 3. 构建 MVI 通信管道与实时监听透传代理
	// 底层真实管道
	rawCmdCh := make(chan types.UICommand, 32)
	rawStateCh := make(chan types.UIState, 1)
	rawEffectCh := make(chan types.UIEffect, 16)

	// 对外暴露管道（带日志拦截）
	uiCmdCh := make(chan types.UICommand, 32)
	uiStateCh := make(chan types.UIState, 1)
	uiEffectCh := make(chan types.UIEffect, 16)

	// 监听并打印 UI -> Core 的指令 (UICommand)
	go func() {
		for cmd := range uiCmdCh {
			log.Printf("[UI -> CORE] 指令: Action=%-1
