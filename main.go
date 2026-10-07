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
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procAllocConsole       = kernel32.NewProc("AllocConsole")
	procAttachConsole      = kernel32.NewProc("AttachConsole")
	procSetConsoleTitle    = kernel32.NewProc("SetConsoleTitleW")
	procSetConsoleCP       = kernel32.NewProc("SetConsoleCP")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
)

const attachParentProcess = ^uintptr(0)

func setupDebugConsole(title string) {
	r, _, _ := procAttachConsole.Call(attachParentProcess)
	if r == 0 {
		procAllocConsole.Call()
	}

	procSetConsoleCP.Call(65001)
	procSetConsoleOutputCP.Call(65001)

	if title != "" {
		t, _ := syscall.UTF16PtrFromString(title)
		procSetConsoleTitle.Call(uintptr(unsafe.Pointer(t)))
	}

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
	noLog := flag.Bool("nolog", false, "关闭控制台调试日志")
	flag.Parse()

	if !*noLog {
		setupDebugConsole(AppName + " [实时日志控制台]")
	}

	log.Println(">>> 进程启动，检查单例锁...")

	lock, ok := platform.AcquireSingleInstance(MutexName, AppName)
	if !ok {
		log.Println("[WARN] 已存在运行实例，退出")
		return
	}
	defer lock.Release()
	log.Println("[OK] 单例锁获取成功")

	walkApp, err := walk.InitApp()
	if err != nil {
		log.Fatalf("[FATAL] 初始化应用失败: %v", err)
	}
	log.Println("[OK] GUI 引擎初始化完成")

	rawCmdCh := make(chan types.UICommand, 32)
	rawStateCh := make(chan types.UIState, 1)
	rawEffectCh := make(chan types.UIEffect, 16)

	uiCmdCh := make(chan types.UICommand, 32)
	uiStateCh := make(chan types.UIState, 1)
	uiEffectCh := make(chan types.UIEffect, 16)

	go func() {
		for cmd := range uiCmdCh {
			log.Printf("[UI->CORE] 指令: %+v", cmd)
			rawCmdCh <- cmd
		}
	}()

	go func() {
		for state := range rawStateCh {
			log.Printf("[CORE->UI] 状态更新: %+v", state)
			select {
			case uiStateCh <- state:
			default:
				select {
				case <-uiStateCh:
				default:
				}
				uiStateCh <- state
			}
		}
	}()

	go func() {
		for effect := range rawEffectCh {
			log.Printf("[CORE->UI] 副作用: %+v", effect)
			uiEffectCh <- effect
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := core.NewService(rawCmdCh, rawStateCh, rawEffectCh)
	go svc.Run(ctx)
	log.Println("[OK] 核心服务启动完成")

	_, err = ui.NewUIEngine(walkApp, uiCmdCh, uiStateCh, uiEffectCh, AppName)
	if err != nil {
		log.Fatalf("[FATAL] 初始化 UI 失败: %v", err)
	}
	log.Println("[OK] UI 挂载成功，进入消息循环")

	walkApp.Run()
	log.Println(">>> 进程主循环退出")
}
