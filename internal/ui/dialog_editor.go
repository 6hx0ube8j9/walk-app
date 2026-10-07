package ui

import (
	"fmt"
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

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

// RunEditor 标准通用对话框编辑器
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
						dlg.Accept()
					},
				},
				PushButton{
					AssignTo: &cancelPB,
					Text:     cfg.CancelBtnText,
					MinSize:  Size{Width: 80, Height: 26},
					OnClicked: func() {
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
		return EditorResult{Accepted: false, Error: err}
	}

	// 1. 设置 Esc 键原生绑定取消按钮
	_ = dlg.SetCancelButton(cancelPB)

	// 2. 焦点在“保存”按钮按 Enter 保存；焦点在“取消”按钮按 Enter 取消
	acceptPB.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			dlg.Accept()
		}
	})
	cancelPB.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			dlg.Cancel()
		}
	})

	// 3. 多行文本框 TextEdit：注入 ES_WANTRETURN (0x1000) 确保 Enter 换行，Ctrl+Enter / Ctrl+S 快捷保存
	textEdits := findTextEdits(dlg)
	for _, te := range textEdits {
		curTE := te
		hwnd := curTE.Handle()
		style := win.GetWindowLong(hwnd, win.GWL_STYLE)
		if style&0x1000 == 0 {
			win.SetWindowLong(hwnd, win.GWL_STYLE, style|0x1000)
			win.SetWindowPos(hwnd, 0, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_FRAMECHANGED)
		}

		curTE.KeyDown().Attach(func(key walk.Key) {
			if (key == walk.KeyReturn || key == walk.Key('S')) && walk.ModifiersDown() == walk.ModControl {
				dlg.Accept()
			}
		})
	}

	// 4. 单行输入框 LineEdit：敲 Enter 直接保存提交
	lineEdits := findLineEdits(dlg)
	for _, le := range lineEdits {
		le.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn && walk.ModifiersDown() == 0 {
				dlg.Accept()
			}
		})
	}

	if cfg.OnReady != nil {
		cfg.OnReady(dlg)
	}

	defer dlg.Dispose()

	dlg.Starting().Attach(func() {
		lockWindowSize(dlg.Handle())
		centerDialog(dlg, owner, hActive)
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
					return
				}
				processErr = err
			}
			isAccepted = true
		}

		if !*canceled {
			restoreFocus(owner, hActive)
		}
	})

	dlg.Run()

	return EditorResult{Accepted: isAccepted, Error: processErr}
}

// OpenTestEditorDialog 供托盘测试菜单调用的样板窗口
func OpenTestEditorDialog(owner walk.Form) EditorResult {
	var nameLE *walk.LineEdit
	var contentTE *walk.TextEdit

	cfg := EditorConfig{
		Title:     "配置内容编辑测试",
		Width:     450,
		MinHeight: 320,
		Widgets: []Widget{
			Label{Text: "配置项名称（单行框，敲 Enter 触发保存）："},
			LineEdit{
				AssignTo: &nameLE,
				Text:     "测试规则项目_01",
			},
			Label{Text: "规则内容（多行框，敲 Enter 换行，Ctrl+Enter 保存）："},
			TextEdit{
				AssignTo: &contentTE,
				Text:     "rules:\r\n  - DOMAIN-SUFFIX,google.com,Proxy\r\n  - GEOIP,CN,DIRECT",
				VScroll:  true,
			},
		},
		OnAccept: func() (bool, error) {
			fmt.Printf("[测试] 保存成功！名称: %s, 内容行数: %d\n", nameLE.Text(), len(contentTE.Text()))
			return true, nil
		},
	}

	return RunEditor(owner, cfg)
}

// -----------------------------------------------------------------------------
// 内置辅助函数：控件扫描与窗口位置管理
// -----------------------------------------------------------------------------

func findTextEdits(container walk.Container) []*walk.TextEdit {
	if container == nil || container.Children() == nil {
		return nil
	}
	var list []*walk.TextEdit
	for i := 0; i < container.Children().Len(); i++ {
		child := container.Children().At(i)
		if te, ok := child.(*walk.TextEdit); ok {
			list = append(list, te)
		} else if c, ok := child.(walk.Container); ok {
			list = append(list, findTextEdits(c)...)
		}
	}
	return list
}

func findLineEdits(container walk.Container) []*walk.LineEdit {
	if container == nil || container.Children() == nil {
		return nil
	}
	var list []*walk.LineEdit
	for i := 0; i < container.Children().Len(); i++ {
		child := container.Children().At(i)
		if le, ok := child.(*walk.LineEdit); ok {
			list = append(list, le)
		} else if c, ok := child.(walk.Container); ok {
			list = append(list, findLineEdits(c)...)
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
