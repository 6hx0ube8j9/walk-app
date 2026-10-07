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
						log.Println("[原生事件] >>> 触发了【保存】按钮的 OnClicked <<<")
						dlg.Accept()
					},
				},
				PushButton{
					AssignTo: &cancelPB,
					Text:     cfg.CancelBtnText,
					MinSize:  Size{Width: 80, Height: 26},
					OnClicked: func() {
						log.Println("[原生事件] >>> 触发了【取消】按钮的 OnClicked <<<")
						dlg.Cancel()
					},
				},
			},
		},
	)

	// 完全遵循另一位 AI 的建议：
	// 1. 设置 DefaultButton 绑定回车确认
	// 2. 设置 CancelButton 绑定 Esc 取消
	// 3. 不挂载任何 Hook
	err := Dialog{
		AssignTo:      &dlg,
		Title:         cfg.Title,
		DefaultButton: &acceptPB, // 对方方案核心：回车交给默认按钮
		CancelButton:  &cancelPB, // 对方方案核心：Esc 交给取消按钮
		MinSize:       Size{Width: cfg.Width, Height: cfg.MinHeight},
		Layout:        VBox{Margins: Margins{Left: 18, Top: 15, Right: 18, Bottom: 15}, Spacing: 12},
		Children:      layoutChildren,
	}.Create(owner)

	if err != nil {
		log.Printf("[Editor] 弹窗创建失败: %v", err)
		return EditorResult{Accepted: false, Error: err}
	}

	defer dlg.Dispose()

	if cfg.OnReady != nil {
		cfg.OnReady(dlg)
	}

	dlg.Starting().Attach(func() {
		lockWindowSize(dlg.Handle())
		centerDialog(dlg, owner)
	})

	dlg.SizeChanged().Attach(func() {
		centerDialog(dlg, owner)
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

	log.Printf("[Editor] 正在展示原生模式编辑窗口: %s", cfg.Title)
	dlg.Run()

	return EditorResult{Accepted: isAccepted, Error: processErr}
}

func OpenTestEditorDialog(owner walk.Form) EditorResult {
	var nameLE *walk.LineEdit
	var contentTE *walk.TextEdit

	cfg := EditorConfig{
		Title:     "对方AI方案测试(无Hook)",
		Width:     450,
		MinHeight: 320,
		Widgets: []Widget{
			Label{Text: "配置项名称（对方方案：监听 OnKeyDown 试图拦截 Enter 跳转）："},
			LineEdit{
				AssignTo: &nameLE,
				Text:     "测试规则项目_01",
				// 对方方案核心：在控件上监听键盘事件
				OnKeyDown: func(key walk.Key) {
					log.Printf("[对方方案] LineEdit 收到按键: %v", key)
					if key == walk.KeyReturn {
						log.Println("[对方方案] 成功拦截到 Enter，正在转移焦点到多行框！")
						contentTE.SetFocus()
					}
				},
			},
			Label{Text: "规则内容（多行框）："},
			TextEdit{
				AssignTo: &contentTE,
				Text:     "rules:\r\n  - DOMAIN-SUFFIX,google.com,Proxy\r\n  - GEOIP,CN,DIRECT",
				VScroll:  true,
			},
		},
		OnAccept: func() (bool, error) {
			log.Printf("[Editor-Result] 保存成功 -> 名称: %s", nameLE.Text())
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
