package ui

import (
	"log"
	"sync"
	"syscall"
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

const (
	wmGetDlgCode      = 0x0087
	dlgcDefPushButton = 0x0010
	dlgcButton        = 0x2000
)

var (
	globalBtnSubclassCallback uintptr
	btnOldProcMap             sync.Map // win.HWND -> uintptr
	btnActionMap              sync.Map // win.HWND -> func()
)

func init() {
	// 全局终生仅创建 1 次回调，彻底杜绝 Go 槽位泄漏
	globalBtnSubclassCallback = syscall.NewCallback(buttonSubclassProc)
}

// buttonSubclassProc 拦截按钮原生消息，精准接管回车键，杜绝焦点重入死锁
func buttonSubclassProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	oldProcVal, ok := btnOldProcMap.Load(hwnd)
	if !ok {
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}
	oldProc := oldProcVal.(uintptr)

	switch msg {
	case wmGetDlgCode:
		// 告诉 Windows 对话框管理器：本按钮接管回车键
		return dlgcDefPushButton | dlgcButton

	case win.WM_KEYDOWN:
		if wParam == win.VK_RETURN {
			if actionVal, ok := btnActionMap.Load(hwnd); ok {
				action := actionVal.(func())
				action()
				return 0
			}
		}
	}

	return win.CallWindowProc(oldProc, hwnd, msg, wParam, lParam)
}

func setupButtonSubclass(hwnd win.HWND, action func()) {
	btnActionMap.Store(hwnd, action)
	old := win.SetWindowLongPtr(hwnd, win.GWLP_WNDPROC, globalBtnSubclassCallback)
	btnOldProcMap.Store(hwnd, old)
}

func restoreButtonSubclass(hwnd win.HWND) {
	if old, ok := btnOldProcMap.LoadAndDelete(hwnd); ok {
		win.SetWindowLongPtr(hwnd, win.GWLP_WNDPROC, old.(uintptr))
	}
	btnActionMap.Delete(hwnd)
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
						log.Println("[Editor] 触发“保存”动作")
						dlg.Accept()
					},
				},
				PushButton{
					AssignTo: &cancelPB,
					Text:     cfg.CancelBtnText,
					MinSize:  Size{Width: 80, Height: 26},
					OnClicked: func() {
						log.Println("[Editor] 触发“取消”动作")
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
	}.Create(owner)

	if err != nil {
		log.Printf("[Editor] 弹窗创建失败: %v", err)
		return EditorResult{Accepted: false, Error: err}
	}

	// 1. 原生 Esc 绑定全局取消
	_ = dlg.SetCancelButton(cancelPB)

	// 2. 注入底层子类化：彻底接管回车键，且绝不产生焦点重入死锁
	setupButtonSubclass(acceptPB.Handle(), func() {
		log.Println("[Editor] 焦点在“保存”按钮，按 Enter 触发保存")
		dlg.Accept()
	})
	setupButtonSubclass(cancelPB.Handle(), func() {
		log.Println("[Editor] 焦点在“取消”按钮，按 Enter 触发取消")
		dlg.Cancel()
	})

	defer func() {
		restoreButtonSubclass(acceptPB.Handle())
		restoreButtonSubclass(cancelPB.Handle())
		dlg.Dispose()
	}()

	// 3. 输入框回车跳转与快捷保存（纯静态绑定，无任何焦点联动副作用）
	inputs := findInputWidgets(dlg)
	for i, input := range inputs {
		idx := i
		switch w := input.(type) {
		case *walk.LineEdit:
			w.KeyDown().Attach(func(key walk.Key) {
				if key == walk.KeyReturn {
					if walk.ModifiersDown() == walk.ModControl {
						log.Printf("[Editor] 输入框 %d 按下 Ctrl+Enter -> 快捷保存", idx+1)
						dlg.Accept()
						return
					}
					if walk.ModifiersDown() == 0 {
						if idx+1 < len(inputs) {
							log.Printf("[Editor] 输入框 %d 按 Enter -> 光标流转至输入框 %d", idx+1, idx+2)
							inputs[idx+1].SetFocus()
						} else {
							log.Printf("[Editor] 末尾输入框按 Enter -> 聚焦保存按钮")
							acceptPB.SetFocus()
						}
					}
				}
			})
		case *walk.TextEdit:
			w.KeyDown().Attach(func(key walk.Key) {
				if (key == walk.KeyReturn || key == walk.Key('S')) && walk.ModifiersDown() == walk.ModControl {
					log.Println("[Editor] 多行文本框按下 Ctrl+Enter / Ctrl+S -> 快捷保存")
					dlg.Accept()
				}
			})
		}
	}

	if cfg.OnReady != nil {
		cfg.OnReady(dlg)
	}

	dlg.Starting().Attach(func() {
		lockWindowSize(dlg.Handle())
		centerDialog(dlg, owner, hActive)
	})

	// 4. 打开弹窗：聚焦第 1 个输入框，光标精准停在末尾（不全选）
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
