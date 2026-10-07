package ui

import (
	"github.com/tailscale/walk"
)

func SetupDialogKeyFlowNative(dlg *walk.Dialog, acceptPB, cancelPB *walk.PushButton) {
	inputs := CollectInputs(dlg)

	// 1. 注册全局快捷键（通过 Walk 原生 Action，优雅处理 Esc / Ctrl+Enter / Ctrl+S）
	escAction := walk.NewAction()
	escAction.SetShortcut(walk.Shortcut{Key: walk.KeyEscape})
	escAction.Triggered().Attach(func() {
		dlg.Cancel()
	})
	dlg.Actions().Add(escAction)

	submitAction := walk.NewAction()
	submitAction.SetShortcut(walk.Shortcut{Modifiers: walk.ModCtrl, Key: walk.KeyReturn})
	submitAction.Triggered().Attach(func() {
		dlg.Accept()
	})
	dlg.Actions().Add(submitAction)

	saveAction := walk.NewAction()
	saveAction.SetShortcut(walk.Shortcut{Modifiers: walk.ModCtrl, Key: walk.KeyS})
	saveAction.Triggered().Attach(func() {
		dlg.Accept()
	})
	dlg.Actions().Add(saveAction)

	// 2. 为输入控件绑定原生的按键流转逻辑
	for i, in := range inputs {
		idx := i
		switch w := in.(type) {
		case *walk.LineEdit:
			w.KeyDown().Attach(func(key walk.Key) {
				if key == walk.KeyReturn {
					// 单行输入框敲 Enter：跳到下一个控件，如果是末尾则跳转到保存按钮
					if idx+1 < len(inputs) {
						inputs[idx+1].SetFocus()
					} else if acceptPB != nil {
						acceptPB.SetFocus()
					}
				}
			})
		case *walk.TextEdit:
			// TextEdit 设置了 VScroll 或原生多行时，Enter 本身就会正常插入换行，无需额外干涉
		}
	}

	// 3. 窗口激活时首焦定位
	dlg.Activating().Attach(func() {
		FocusFirstInput(inputs)
	})
}
