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
	kfWhGetMessage = 3      // WH_GETMESSAGE: Message queue hook
	kfEsWantReturn = 0x1000 // ES_WANTRETURN: Multi-line edit native newline
)

// Isolated DLL procedures to prevent symbol collision across package files.
var (
	kfModUser32               = syscall.NewLazyDLL("user32.dll")
	kfModKernel32             = syscall.NewLazyDLL("kernel32.dll")
	kfProcSetWindowsHookExW   = kfModUser32.NewProc("SetWindowsHookExW")
	kfProcUnhookWindowsHookEx = kfModUser32.NewProc("UnhookWindowsHookEx")
	kfProcCallNextHookEx      = kfModUser32.NewProc("CallNextHookEx")
	kfProcGetCurrentThreadId  = kfModKernel32.NewProc("GetCurrentThreadId")
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
					// Esc: cancel dialog
					ctx.dlg.Cancel()
					pMsg.Message = win.WM_NULL
					return 0

				case win.VK_RETURN:
					switch {
					case hFocus == ctx.acceptHWND:
						// Enter on Save button
						ctx.dlg.Accept()
						pMsg.Message = win.WM_NULL
						return 0

					case hFocus == ctx.cancelHWND:
						// Enter on Cancel button
						ctx.dlg.Cancel()
						pMsg.Message = win.WM_NULL
						return 0

					default:
						// Enter in input controls
						for i, hwnd := range ctx.inputHWNDs {
							if hFocus == hwnd {
								if ctx.isTextEdit[i] {
									// Multi-line edit: Ctrl+Enter saves, plain Enter inserts newline natively
									if isCtrl {
										ctx.dlg.Accept()
										pMsg.Message = win.WM_NULL
										return 0
									}
									break
								} else {
									// Single-line edit: Ctrl+Enter saves, plain Enter shifts focus to next control
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
					// Ctrl+S: shortcut to save anywhere in input controls
					if isCtrl {
						for _, hwnd := range ctx.inputHWNDs {
							if hFocus == hwnd {
								ctx.dlg.Accept()
								pMsg.Message = win.WM_NULL
								return 0
							}
						}
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

// FocusFirstInput focuses the first available input and positions the caret at the end of the text.
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

// SetupDialogKeyFlow attaches the keyboard routing engine and initializes input controls.
// Returns a cleanup closure that must be deferred before dialog disposal.
func SetupDialogKeyFlow(dlg *walk.Dialog, acceptPB, cancelPB *walk.PushButton) func() {
	ensureKeyFlowCallback()

	inputs := CollectInputs(dlg)
	var inputHWNDs []win.HWND
	var isTextEdit []bool

	for _, in := range inputs {
		inputHWNDs = append(inputHWNDs, in.Handle())
		_, ok := in.(*walk.TextEdit)
		isTextEdit = append(isTextEdit, ok)

		// Inject ES_WANTRETURN style so TextEdit accepts Enter natively
		if ok {
			hwnd := in.Handle()
			style := win.GetWindowLong(hwnd, win.GWL_STYLE)
			if style&kfEsWantReturn == 0 {
				win.SetWindowLong(hwnd, win.GWL_STYLE, style|kfEsWantReturn)
				win.SetWindowPos(hwnd, 0, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_FRAMECHANGED)
			}
		}
	}

	// Focus first input with caret positioned at the end upon dialog activation
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
		tid, _, _ := kfProcGetCurrentThreadId.Call()
		hHook, _, _ := kfProcSetWindowsHookExW.Call(uintptr(kfWhGetMessage), kfCallback, 0, tid)
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
