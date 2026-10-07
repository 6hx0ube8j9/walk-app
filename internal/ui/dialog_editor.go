package ui

import (
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

const (
	whGetMessage = 3
)

var (
	modUser32               = syscall.NewLazyDLL("user32.dll")
	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	procSetWindowsHookExW   = modUser32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = modUser32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = modUser32.NewProc("CallNextHookEx")
	procGetCurrentThreadId  = modKernel32.NewProc("GetCurrentThreadId")

	globalGetMsgCallback uintptr
	activeEditorMu       sync.Mutex
	activeEditorCtx      *editorHookContext
)

type editorHookContext struct {
	hHook      uintptr
	dlg        *walk.Dialog
	acceptHWND win.HWND
	cancelHWND win.HWND
	inputHWNDs []win.HWND
	isTextEdit []bool
}

func init() {
	// 全局终生仅创建 1 次回调，彻底杜绝 Go 槽位耗尽
	globalGetMsgCallback = syscall.NewCallback(editorGetMsgProc)
}

// editorGetMsgProc 线程级消息预处理器：在 IsDialogMessage 介入前精准分流
func editorGetMsgProc(nCode int32, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 && activeEditorCtx != nil {
		ctx := activeEditorCtx
		pMsg := (*win.MSG)(unsafe.Pointer(lParam))

		if pMsg.Message == win.WM_KEYDOWN {
			isCtrl := win.GetKeyState(win.VK_CONTROL) < 0

			switch pMsg.WParam {
			case win.VK_ESCAPE:
				log.Println("[Editor] 按 Esc 键触发取消")
				ctx.dlg.Cancel()
				pMsg.Message = win.WM_NULL // 吞噬按键，防止系统重复触发
				return 0

			case win.VK_RETURN:
				hFocus := win.GetFocus()
				switch {
				case hFocus == ctx.acceptHWND:
					log.Println("[Editor] 焦点在“保存”按钮，按 Enter 触发保存")
					ctx.dlg.Accept()
					pMsg.Message = win.WM_NULL
					return 0

				case hFocus == ctx.cancelHWND:
					log.Println("[Editor] 焦点在“取消”按钮，按 Enter 触发取消")
					ctx.dlg.Cancel()
					pMsg.Message = win.WM_NULL
					return 0

				default:
					// 判定是否在输入框内
					for i, hwnd := range ctx.inputHWNDs {
						if hFocus == hwnd {
							if ctx.isTextEdit[i] {
								// 多行输入框 (TextEdit)
								if isCtrl {
									log.Println("[Editor] 多行文本框按 Ctrl+Enter -> 快捷保存")
									ctx.dlg.Accept()
									pMsg.Message = win.WM_NULL
									return 0
								}
								// 普通 Enter：不拦截，放行让 TextEdit 正常换行！
								break
							} else {
								// 单行输入框 (LineEdit)
								if isCtrl {
									log.Printf("[Editor] 输入框 %d 按 Ctrl+Enter -> 快捷保存", i+1)
									ctx.dlg.Accept()
									pMsg.Message = win.WM_NULL
									return 0
								}
								// 普通 Enter：顺畅流转到下一个输入框
								if i+1 < len(ctx.inputHWNDs) {
									log.Printf("[Editor] 输入框 %d 按 Enter -> 光标流转至输入框 %d", i+1, i+2)
									win.SetFocus(ctx.inputHWNDs[i+1])
								} else {
									log.Printf("[Editor] 末尾输入框按 Enter -> 聚焦保存按钮")
									win.SetFocus(ctx.acceptHWND)
								}
								pMsg.Message = win.WM_NULL
								return 0
							}
						}
					}
				}

			case 'S':
				if isCtrl {
					hFocus := win.GetFocus()
					for _, hwnd := range ctx.inputHWNDs {
						if hFocus == hwnd {
							log.Println("[Editor] 输入框内按 Ctrl+S -> 快捷保存")
							ctx.dlg.Accept()
							pMsg.Message = win.WM_NULL
							return 0
						}
					}
				}
			}
		}
	}

	var hHook uintptr
	if activeEditorCtx != nil {
		hHook = activeEditorCtx.hHook
	}
	ret, _, _ := procCallNextHookEx.Call(hHook, uintptr(nCode), wParam, lParam)
	return ret
}

type EditorConfig struct {
	Title         string
	Width         int
	MinHeight     int
	Widgets       []Widget
	OnAccept      func() (bool, error)
	AcceptBtnText string
	CancelBtnText string
	OnReady       func(dlg *walk.Dialog)
}

type EditorResult struct {
	Accepted bool
	Error    error
}

func RunEditor(owner walk.Form, cfg EditorConfig) EditorResult {
	if cfg.AcceptBtnText == "" {
		cfg.AcceptBtnText = "保存(&S)"
	}
	if cfg.CancelBtnText == "" {
		cfg.CancelBtnText = "取消(&C)"
	}

	hActive := win.GetForegroundWindow()

	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	var isAccepted bool
	var processErr error

	layoutChildren := append(cfg.Widgets,
		VSpacer{},
		Composite{
			Layout: HBox{MarginsZero: true, Spacing: 10},
			Children: []Widget{
				HSpacer{},
				PushButton{
					AssignTo: &acceptPB,
					Text:     cfg.AcceptBtnText,
					MinSize:  Size{Width: 80, Height: 26},
					OnClicked: func() {
						log.Println("[Editor] 鼠标点击了“保存”按钮")
						dlg.Accept()
					},
				},
				PushButton{
					AssignTo: &cancelPB,
					Text:     cfg.CancelBtnText,
					MinSize:  Size{Width: 80, Height: 26},
					OnClicked: func() {
						log.Println("[Editor] 鼠标点击了“取消”按钮")
						dlg.Cancel()
					},
				},
			},
		},
	)

	err := Dialog{
		AssignTo: &dlg,
		Title:    cfg.Title,
		MinSize:  Size{Width: cfg.Width, Height: cfg.MinHeight},
		Layout:   VBox{Margins: Margins{Left: 18, Top: 15, Right: 18, Bottom: 15}, Spacing: 12},
		Children: layoutChildren,
		// 严禁在此处绑定 DefaultButton / CancelButton，交由 WH_GETMESSAGE 完全接管
	}.Create(owner)

	if err != nil {
		log.Printf("[Editor] 弹窗创建失败: %v", err)
		return EditorResult{Accepted: false, Error: err}
	}

	// 收集所有输入控件
	inputs := findInputWidgets(dlg)
	var inputHWNDs []win.HWND
	var isTextEdit []bool
	for _, in := range inputs {
		inputHWNDs = append(inputHWNDs, in.Handle())
		_, ok := in.(*walk.TextEdit)
		isTextEdit = append(isTextEdit, ok)

		// 为 TextEdit 打上 ES_WANTRETURN，确保原生消息接收回车
		if ok {
			hwnd := in.Handle()
			style := win.GetWindowLong(hwnd, win.GWL_STYLE)
			if style&0x1000 == 0 {
				win.SetWindowLong(hwnd, win.GWL_STYLE, style|0x1000)
				win.SetWindowPos(hwnd, 0, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_FRAMECHANGED)
			}
		}
	}

	// =========================================================================
	// 挂载当前 UI 线程专用的 WH_GETMESSAGE 钩子
	// =========================================================================
	activeEditorMu.Lock()
	runtime.LockOSThread()

	ctx := &editorHookContext{
		dlg:        dlg,
		acceptHWND: acceptPB.Handle(),
		cancelHWND: cancelPB.Handle(),
		inputHWNDs: inputHWNDs,
		isTextEdit: isTextEdit,
	}
	activeEditorCtx = ctx

	tid, _, _ := procGetCurrentThreadId.Call()
	hHook, _, _ := procSetWindowsHookExW.Call(uintptr(whGetMessage), globalGetMsgCallback, 0, tid)
	ctx.hHook = hHook

	defer func() {
		if ctx.hHook != 0 {
			procUnhookWindowsHookEx.Call(ctx.hHook)
			ctx.hHook = 0
		}
		activeEditorCtx = nil
		runtime.UnlockOSThread()
		activeEditorMu.Unlock()
		dlg.Dispose()
	}()

	if cfg.OnReady != nil {
		cfg.OnReady(dlg)
	}

	dlg.Starting().Attach(func() {
		lockWindowSize(dlg.Handle())
		centerDialog(dlg, owner, hActive)
	})

	// 激活时：聚焦第 1 个输入框，光标定位于文字末尾
	dlg.Activating().Attach(func() {
		if len(inputs) > 0 {
			firstInput := inputs[0]
			firstInput.SetFocus()

			if le, ok := firstInput.(*walk.LineEdit); ok {
				textLen := len([]rune(le.Text()))
				le.SetTextSelection(textLen, textLen)
				log.Printf("[Editor] 弹窗展示，默认聚焦第 1 个输入框，光标定位于文字末尾 (下标: %d)", textLen)
			}
		}
	})

	dlg.SizeChanged().Attach(func() {
		centerDialog(dlg, owner, hActive)
	})

	dlg.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if reason == walk.CloseReasonUnknown && dlg.Result() == walk.DlgCmdOK {
			if cfg.OnAccept != nil {
				ok, err := cfg.OnAccept()
				if !ok {
					*canceled = true
					log.Println("[Editor] 业务校验未通过，拦截关闭")
					return
				}
				processErr = err
			}
			isAccepted = true
			log.Println("[Editor] 编辑完成并确认保存")
		} else {
			log.Println("[Editor] 放弃保存，退出编辑窗")
		}

		if !*canceled {
			restoreFocus(owner, hActive)
		}
	})

	log.Printf("[Editor] 正在展示编辑窗口: %s", cfg.Title)
	dlg.Run()

	return EditorResult{Accepted: isAccepted, Error: processErr}
}

func OpenTestEditorDialog(owner walk.Form) EditorResult {
	var nameLE *walk.LineEdit
	var contentTE *walk.TextEdit

	cfg := EditorConfig{
		Title:     "配置内容编辑测试",
		Width:     450,
		MinHeight: 320,
		Widgets: []Widget{
			Label{Text: "配置项名称（单行框，按 Enter 跳转到下方内容框）："},
			LineEdit{
				AssignTo: &nameLE,
				Text:     "测试规则项目_01",
			},
			Label{Text: "规则内容（多行框，按 Enter 换行，Ctrl+Enter 保存）："},
			TextEdit{
				AssignTo: &contentTE,
				Text:     "rules:\r\n  - DOMAIN-SUFFIX,google.com,Proxy\r\n  - GEOIP,CN,DIRECT",
				VScroll:  true,
			},
		},
		OnAccept: func() (bool, error) {
			log.Printf("[Editor-Result] 校验通过 -> 名称: %s, 内容行数: %d", nameLE.Text(), len(contentTE.Text()))
			return true, nil
		},
	}

	return RunEditor(owner, cfg)
}

func findInputWidgets(container walk.Container) []walk.Widget {
	if container == nil || container.Children() == nil {
		return nil
	}
	var list []walk.Widget
	for i := 0; i < container.Children().Len(); i++ {
		child := container.Children().At(i)
		switch w := child.(type) {
		case *walk.LineEdit, *walk.TextEdit:
			list = append(list, w)
		case walk.Container:
			list = append(list, findInputWidgets(w)...)
		}
	}
	return list
}

func lockWindowSize(hwnd win.HWND) {
	style := win.GetWindowLong(hwnd, win.GWL_STYLE)
	style &^= win.WS_THICKFRAME | win.WS_MAXIMIZEBOX
	win.SetWindowLong(hwnd, win.GWL_STYLE, style)
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_FRAMECHANGED)
}

func centerDialog(dlg *walk.Dialog, owner walk.Form, hActive win.HWND) {
	if dlg == nil {
		return
	}

	var dRect win.RECT
	win.GetWindowRect(dlg.Handle(), &dRect)
	dlgW := dRect.Right - dRect.Left
	dlgH := dRect.Bottom - dRect.Top

	var dcRect win.RECT
	win.GetClientRect(dlg.Handle(), &dcRect)
	dPtLT := win.POINT{X: 0, Y: 0}
	win.ClientToScreen(dlg.Handle(), &dPtLT)

	dcOffsetCX := (dPtLT.X - dRect.Left) + dcRect.Right/2
	dcOffsetCY := (dPtLT.Y - dRect.Top) + dcRect.Bottom/2

	var workArea win.RECT
	win.SystemParametersInfo(0x0030, 0, unsafe.Pointer(&workArea), 0)

	var x, y int32
	shouldFollowOwner := owner != nil && owner.Visible() && !win.IsIconic(owner.Handle())

	if shouldFollowOwner && hActive != 0 && hActive != owner.Handle() {
		shouldFollowOwner = false
	}

	if shouldFollowOwner {
		var pClientRect win.RECT
		win.GetClientRect(owner.Handle(), &pClientRect)

		ptLT := win.POINT{X: 0, Y: 0}
		win.ClientToScreen(owner.Handle(), &ptLT)

		pCX := ptLT.X + pClientRect.Right/2
		pCY := ptLT.Y + pClientRect.Bottom/2

		x = pCX - dcOffsetCX
		y = pCY - dcOffsetCY
	} else {
		x = workArea.Left + (workArea.Right-workArea.Left-dlgW)/2
		y = workArea.Top + (workArea.Bottom-workArea.Top-dlgH)/2
	}

	if x < workArea.Left {
		x = workArea.Left
	} else if x+dlgW > workArea.Right {
		x = workArea.Right - dlgW
	}

	if y < workArea.Top {
		y = workArea.Top
	} else if y+dlgH > workArea.Bottom {
		y = workArea.Bottom - dlgH
	}

	win.SetWindowPos(dlg.Handle(), win.HWND_TOP, x, y, 0, 0, win.SWP_NOSIZE)
}

func restoreFocus(parent walk.Form, hActive win.HWND) {
	if parent != nil && parent.Visible() && !win.IsIconic(parent.Handle()) {
		win.SetForegroundWindow(parent.Handle())
		win.SetFocus(parent.Handle())
	} else if hActive != 0 && !win.IsIconic(hActive) {
		win.SetForegroundWindow(hActive)
		win.SetFocus(hActive)
	}
}
