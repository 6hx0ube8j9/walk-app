package ui

import (
	"github.com/tailscale/walk"
)

func SetupDialogKeyFlow(dlg *walk.Dialog, acceptPB, cancelPB *walk.PushButton) func() {
	inputs := CollectInputs(dlg)

	// 1. 全局快捷键注册（Esc 退出、Ctrl+Enter 提交、Ctrl+S 提交）
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

	// 2. 核心补丁：通过 FocusIn 动态调度 DefaultButton，解决 Tab 切换后 Enter 失效的问题
	if cancelPB != nil {
		cancelPB.FocusIn().Attach(func() {
			// Tab 到取消按钮时，将其设为默认按钮，此时按 Enter 触发取消
			dlg.SetDefaultButton(cancelPB)
		})
	}

	if acceptPB != nil {
		acceptPB.FocusIn().Attach(func() {
			// Tab 到保存按钮时，将其设为默认按钮，此时按 Enter 触发保存
			dlg.SetDefaultButton(acceptPB)
		})
	}

	for i, in := range inputs {
		idx := i
		// 当焦点回到任何输入框时，清空默认按钮，防止多行文本框的 Enter 被按钮劫持
		in.FocusIn().Attach(func() {
			dlg.SetDefaultButton(nil)
		})

		if le, ok := in.(*walk.LineEdit); ok {
			le.KeyDown().Attach(func(key walk.Key) {
				if key == walk.KeyReturn {
					if idx+1 < len(inputs) {
						inputs[idx+1].SetFocus()
					} else if acceptPB != nil {
						acceptPB.SetFocus()
					}
				}
			})
		}
	}

	// 3. 激活时初始化首焦
	dlg.Activating().Attach(func() {
		dlg.SetDefaultButton(nil) // 启动时默认置空，让首个输入框接管
		FocusFirstInput(inputs)
	})

	return func() {}
}
