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

	// 3. 多行文本框 TextEdit：注入 ES_WANTRETURN 确保 Enter 原生换行，Ctrl+Enter / Ctrl+S 快捷保存
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
