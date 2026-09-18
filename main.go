package main

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-H windowsgui -s -w" -o app_b.exe main.go

const (
	// 全局唯一的互斥锁名称与主窗口标题
	appMutexName  = "Global\\MyProductionTrayApp_SingleInstance_Mutex"
	appWindowName = "生产级后台常驻程序"
)

var (
	moduser32             = syscall.NewLazyDLL("user32.dll")
	modkernel32           = syscall.NewLazyDLL("kernel32.dll")
	procRegisterWindowMsg = moduser32.NewProc("RegisterWindowMessageW")
	procCreateMutex       = modkernel32.NewProc("CreateMutexW")
	msgTaskbarCreated     uint32
)

func init() {
	// 1. 锁死主 OS 线程，保障 Win32 消息循环稳定
	runtime.LockOSThread()

	// 2. 注册 Explorer 重启消息广播
	taskbarStr, _ := syscall.UTF16PtrFromString("TaskbarCreated")
	ret, _, _ := procRegisterWindowMsg.Call(uintptr(unsafe.Pointer(taskbarStr)))
	msgTaskbarCreated = uint32(ret)
}

// 检查并确保全局单实例运行
func checkSingleInstance() (uintptr, bool) {
	const ERROR_ALREADY_EXISTS = 183
	mutexNamePtr, _ := syscall.UTF16PtrFromString(appMutexName)

	hMutex, _, _ := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(mutexNamePtr)))
	if syscall.GetLastError() == syscall.Errno(ERROR_ALREADY_EXISTS) {
		// 已经存在运行实例，寻找原窗口并激活唤醒
		titlePtr, _ := syscall.UTF16PtrFromString(appWindowName)
		hwnd := win.FindWindow(nil, titlePtr)
		if hwnd != 0 {
			win.ShowWindow(hwnd, win.SW_RESTORE)
			win.SetForegroundWindow(hwnd)
		}
		return 0, false
	}
	return hMutex, true
}

func main() {
	// 单实例校验
	hMutex, isOnlyInstance := checkSingleInstance()
	if !isOnlyInstance {
		return
	}
	defer win.CloseHandle(win.HANDLE(hMutex))

	app, err := walk.InitApp()
	if err != nil {
		return
	}

	var mw *walk.MainWindow

	err = MainWindow{
		AssignTo: &mw,
		Title:    appWindowName,
		MinSize:  Size{Width: 360, Height: 240},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "程序已进入后台保护运行。\n点击右上角 X 会直接隐藏到托盘。"},
		},
	}.Create()

	if err != nil {
		return
	}

	// 声明托盘指针，方便消息处理中捕获
	var ni *walk.NotifyIcon

	// 唤醒/隐藏窗口通用函数
	toggleWindow := func() {
		if mw.Visible() {
			mw.Hide()
		} else {
			mw.Show()
			win.ShowWindow(mw.Handle(), win.SW_RESTORE)
			win.SetForegroundWindow(mw.Handle())
		}
	}

	// Win32 底层消息拦截器
	var oldWndProc uintptr
	newWndProc := syscall.NewCallback(func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case win.WM_CLOSE:
			// 拦截普通点击 X 的关闭行为，改为平滑隐藏
			win.ShowWindow(hwnd, win.SW_HIDE)
			return 0

		case win.WM_QUERYENDSESSION:
			// Windows 关机或注销前询问：放行并允许关机
			return 1

		case win.WM_ENDSESSION:
			// Windows 确认执行关机：清理托盘并退出
			if wParam != 0 {
				if ni != nil {
					ni.SetVisible(false)
				}
				app.Exit(0)
				return 0
			}

		default:
			// 资源管理器崩溃重启自愈：重新注册显示托盘图标
			if msgTaskbarCreated != 0 && msg == msgTaskbarCreated {
				if ni != nil {
					ni.SetVisible(false)
					ni.SetVisible(true)
				}
			}
		}

		return win.CallWindowProc(oldWndProc, hwnd, msg, wParam, lParam)
	})
	oldWndProc = win.SetWindowLongPtr(mw.Handle(), win.GWLP_WNDPROC, newWndProc)

	// 初始状态隐藏窗口
	mw.Hide()

	// 初始化托盘
	ni, err = walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip(appWindowName)
	ni.SetIcon(walk.IconInformation())

	// 单击托盘图标切换显隐
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			toggleWindow()
		}
	})

	// 右键菜单
	showAction := walk.NewAction()
	showAction.SetText("显示/隐藏主界面")
	showAction.Triggered().Attach(toggleWindow)
	ni.ContextMenu().Actions().Add(showAction)

	separator := walk.NewSeparatorAction()
	ni.ContextMenu().Actions().Add(separator)

	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		// 1. 显式注销托盘图标，杜绝幽灵残影
		ni.SetVisible(false)

		// 2. 卸载消息过程钩子
		win.SetWindowLongPtr(mw.Handle(), win.GWLP_WNDPROC, oldWndProc)

		// 3. 安全退出应用
		mw.Close()
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	if err := ni.SetVisible(true); err != nil {
		return
	}

	app.Run()
}
