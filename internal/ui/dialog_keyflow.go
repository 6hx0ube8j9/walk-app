package ui

import (
	"github.com/tailscale/walk"
)

// CollectInputs traverses the container recursively to discover all LineEdit and TextEdit controls.
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

// FocusFirstInput focuses the first available editable input and positions the caret at the end.
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

// SetupDialogKeyFlow sets up keyboard flow using 100% native Walk actions and events.
// Zero hooks, zero DLLs, zero global variables.
func SetupDialogKeyFlow(dlg *walk.Dialog, acceptPB, cancelPB *walk.PushButton) func() {
	inputs := CollectInputs(dlg)

	// 1. Window-level shortcuts via native Walk Actions (Esc / Ctrl+Enter / Ctrl+S)
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

	// 2. LineEdit Enter key routing
	for i, in := range inputs {
		idx := i
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
		// TextEdit natively inserts newline because Dialog has no DefaultButton to hijack Enter
	}

	// 3. Focus first editable control upon dialog activation
	dlg.Activating().Attach(func() {
		FocusFirstInput(inputs)
	})

	// No hook cleanup required; returned closure maintains API compatibility
	return func() {}
}
