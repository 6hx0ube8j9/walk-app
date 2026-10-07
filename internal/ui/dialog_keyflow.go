package ui

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

const (
	kfWhGetMessage = 3      // WH_GETMESSAGE: Message queue filter
	kfEsWantReturn = 0x1000 // ES_WANTRETURN: Multi-line edit native newline
)

// Isolated DLL procedures to prevent symbol collision across package files.
var (
	kfModUser32               = syscall.NewLazyDLL("user32.dll")
	kfProcSetWindowsHookExW   = kfModUser32.NewProc("SetWindowsHookExW")
	kfProcUnhookWindowsHookEx = kfModUser32.NewProc("UnhookWindowsHookEx")
	kfProcCallNextHookEx      = kfModUser32.NewProc("CallNextHookEx")
)

var (
	kfOnce       sync.Once
	kfCallback   uintptr
	kfStackMu    sync.Mutex
	kfStack      []*keyFlowContext
	activeHookId uintptr
)

type keyFlowContext struct {
	dlg        *walk.Dialog
	acceptHWND win.HWND
	cancelHWND win.HWND
	inputHWNDs []win.HWND
	isTextEdit []bool
}

func ensureKeyFlowCallback() {
	kfOnce.Do(func() {
		kfCallback = syscall.NewCallback(keyFlowMessageProc)
	})
}

// keyFlowMessageProc filters keystrokes before IsDialogMessage processing.
func keyFlowMessageProc(nCode int32, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 {
		kfStackMu.Lock()
		var ctx *keyFlowContext
		if len(kfStack) > 0 {
			ctx = kfStack[len(kfStack)-1] // Always route to active top-level dialog
		}
		kfStackMu.Unlock()

		if ctx != nil {
			pMsg := (*win.MSG)(unsafe.Pointer(lParam))
			if pMsg.Message == win.WM_KEYDOWN {
				isCtrl := win.GetKeyState(win.VK_CONTROL) < 0
				hFocus := win.GetFocus()

				switch pMsg.WParam {
				case win.VK_ESCAPE:
					// Esc anywhere: cancel dialog
					ctx.dlg.Cancel()
					pMsg.Message = win.WM_NULL
					return 0

				case win.VK_RETURN:
					switch {
					case hFocus == ctx.acceptHWND:
						// Enter on Save button: trigger Accept
						ctx.dlg.Accept()
						pMsg.Message = win.WM_NULL
						return 0

					case hFocus == ctx.cancelHWND:
						// Enter on Cancel button: trigger Cancel
						ctx.dlg.Cancel()
						pMsg.Message = win.WM_NULL
						return 0

					default:
						// Enter inside input controls
						for i, hwnd := range ctx.inputHWNDs {
							if hFocus == hwnd {
								if ctx.isTextEdit[i] {
									// TextEdit: Ctrl+Enter saves, plain Enter passes through for newline
									if isCtrl {
										ctx.dlg.Accept()
										pMsg.Message = win.WM_NULL
										return 0
									}
									break
								} else {
									// LineEdit: Ctrl+Enter saves, plain Enter moves to next input or Save
									if isCtrl {
										ctx.dlg.Accept()
										pMsg.Message = win.WM_NULL
										return 0
									}
									if i+1 < len(ctx.inputHWNDs) {
										win.SetFocus(ctx.inputHWNDs[i+1])
									} else {
										win.SetFocus(ctx.acceptHWND)
									}
									pMsg.Message = win.WM_NULL
									return 0
								}
							}
						}
					}

				case 'S':
					// Ctrl+S anywhere: trigger Accept
					if isCtrl {
						ctx.dlg.Accept()
						pMsg.Message = win.WM_NULL
						return 0
					}
				}
			}
		}
	}

	var hHook uintptr
	kfStackMu.Lock()
	hHook = activeHookId
	kfStackMu.Unlock()

	ret, _, _ := kfProcCallNextHookEx.Call(hHook, uintptr(nCode), wParam, lParam)
	return ret
}

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

// SetupDialogKeyFlow attaches the WH_GETMESSAGE hook and routes all keyboard flow.
func SetupDialogKeyFlow(dlg *walk.Dialog, acceptPB, cancelPB *walk.PushButton) func() {
	ensureKeyFlowCallback()

	inputs := CollectInputs(dlg)
	var inputHWNDs []win.HWND
	var isTextEdit []bool

	for _, in := range inputs {
		inputHWNDs = append(inputHWNDs, in.Handle())
		_, ok := in.(*walk.TextEdit)
		isTextEdit = append(isTextEdit, ok)

		// Inject ES_WANTRETURN so TextEdit accepts plain Enter natively
		if ok {
			hwnd := in.Handle()
			style := win.GetWindowLong(hwnd, win.GWL_STYLE)
			if style&kfEsWantReturn == 0 {
				win.SetWindowLong(hwnd, win.GWL_STYLE, style|kfEsWantReturn)
				win.SetWindowPos(hwnd, 0, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_FRAMECHANGED)
			}
		}
	}

	dlg.Activating().Attach(func() {
		FocusFirstInput(inputs)
	})

	ctx := &keyFlowContext{
		dlg:        dlg,
		acceptHWND: acceptPB.Handle(),
		cancelHWND: cancelPB.Handle(),
		inputHWNDs: inputHWNDs,
		isTextEdit: isTextEdit,
	}

	runtime.LockOSThread()

	kfStackMu.Lock()
	if len(kfStack) == 0 {
		tid := win.GetCurrentThreadId()
		hHook, _, _ := kfProcSetWindowsHookExW.Call(uintptr(kfWhGetMessage), kfCallback, 0, uintptr(tid))
		activeHookId = hHook
	}
	kfStack = append(kfStack, ctx)
	kfStackMu.Unlock()

	return func() {
		kfStackMu.Lock()
		for i := len(kfStack) - 1; i >= 0; i-- {
			if kfStack[i] == ctx {
				kfStack = append(kfStack[:i], kfStack[i+1:]...)
				break
			}
		}
		if len(kfStack) == 0 && activeHookId != 0 {
			kfProcUnhookWindowsHookEx.Call(activeHookId)
			activeHookId = 0
		}
		kfStackMu.Unlock()

		runtime.UnlockOSThread()
	}
}
