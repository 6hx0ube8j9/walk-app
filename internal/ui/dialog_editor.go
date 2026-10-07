package ui

import (
	"log"

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
	}.Create(owner)

	// 调用方式完全不变
	cleanupKeyFlow := SetupDialogKeyFlow(dlg, acceptPB, cancelPB)
	defer func() {
		cleanupKeyFlow()
		dlg.Dispose()
	}()

	if cfg.OnReady != nil {
		cfg.OnReady(dlg)
	}

	dlg.Starting().Attach(func() {
		lockWindowSize(dlg.Handle())
		centerDialog(dlg, owner) // 匹配 dialog_util.go 的 2 个参数
	})

	dlg.SizeChanged().Attach(func() {
		centerDialog(dlg, owner) // 匹配 dialog_util.go 的 2 个参数
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

func restoreFocus(parent walk.Form, hActive win.HWND) {
	if parent != nil && parent.Visible() && !win.IsIconic(parent.Handle()) {
		win.SetForegroundWindow(parent.Handle())
		win.SetFocus(parent.Handle())
	} else if hActive != 0 && !win.IsIconic(hActive) {
		win.SetForegroundWindow(hActive)
		win.SetFocus(hActive)
	}
}
