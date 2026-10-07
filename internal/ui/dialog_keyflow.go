package ui

import (
	"github.com/tailscale/walk"
)

// CollectInputs 递归遍历容器查找所有 LineEdit 和 TextEdit 控件
func CollectInputs(container walk.Container) []walk.Widget {
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
			list = append(list, CollectInputs(w)...)
		}
	}
	return list
}

// FocusFirstInput 聚焦第 1 个可用输入框并将光标定位于文字末尾
func FocusFirstInput(inputs []walk.Widget) {
	for _, in := range inputs {
		if in.Visible() && in.Enabled() {
			in.SetFocus()
			if le, ok := in.(*walk.LineEdit); ok {
				textLen := len([]rune(le.Text()))
				le.SetTextSelection(textLen, textLen)
			}
			return
		}
	}
}

// SetupDialogKeyFlow 使用 100% 纯 Walk 原生事件管理键盘流
func SetupDialogKeyFlow(dlg *walk.Dialog, acceptPB, cancelPB *walk.PushButton) func() {
	inputs := CollectInputs(dlg)

	// 1. 窗体全局快捷键（通过 ShortcutActions 注册 Esc / Ctrl+Enter / Ctrl+S）
	escAction := walk.NewAction()
	escAction.SetShortcut(walk.Shortcut{Key: walk.KeyEscape})
	escAction.Triggered().Attach(func() {
		dlg.Cancel()
	})
	dlg.ShortcutActions().Add(escAction)

	submitAction := walk.NewAction()
	submitAction.SetShortcut(walk.Shortcut{Modifiers: walk.ModControl, Key: walk.KeyReturn})
	submitAction.Triggered().Attach(func() {
		dlg.Accept()
	})
	dlg.ShortcutActions().Add(submitAction)

	saveAction := walk.NewAction()
	saveAction.SetShortcut(walk.Shortcut{Modifiers: walk.ModControl, Key: walk.KeyS})
	saveAction.Triggered().Attach(func() {
		dlg.Accept()
	})
	dlg.ShortcutActions().Add(saveAction)

	// 2. 动态调度 DefaultButton：通过 FocusedChanged 解决 Tab 切换到取消后 Enter 失效的问题
	if cancelPB != nil {
		cancelPB.FocusedChanged().Attach(func() {
			if cancelPB.Focused() {
				dlg.SetDefaultButton(cancelPB)
			}
		})
	}

	if acceptPB != nil {
		acceptPB.FocusedChanged().Attach(func() {
			if acceptPB.Focused() {
				dlg.SetDefaultButton(acceptPB)
			}
		})
	}

	for i, in := range inputs {
		idx := i
		// 当焦点进入任何输入框时，清空默认按钮，防止多行文本框的 Enter 被按钮劫持
		in.FocusedChanged().Attach(func() {
			if in.Focused() {
				dlg.SetDefaultButton(nil)
			}
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

	// 3. 激活时初始化首焦，默认清空 DefaultButton
	dlg.Activating().Attach(func() {
		dlg.SetDefaultButton(nil)
		FocusFirstInput(inputs)
	})

	return func() {}
}
