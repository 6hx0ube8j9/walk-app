package ui

import (
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

const spiGetWorkArea = 0x0030

func ShowErrorDialog(owner walk.Form, title, message string) {
	var dlg *walk.Dialog
	var acceptPB *walk.PushButton

	var parent walk.Form
	if owner != nil && owner.Visible() {
		parent = owner
	}

	err := Dialog{
		AssignTo:      &dlg,
		Title:         title,
		MinSize:       Size{Width: 320, Height: 150},
		Layout:        VBox{Margins: Margins{Top: 15, Bottom: 15, Left: 15, Right: 15}, Spacing: 10},
		DefaultButton: &acceptPB,
		Children: []Widget{
			Label{Text: message},
			VSpacer{},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo:  &acceptPB,
						Text:      "确定",
						MinSize:   Size{Width: 70, Height: 26},
						OnClicked: func() { dlg.Accept() },
					},
				},
			},
		},
	}.Create(parent)

	if err != nil {
		return
	}

	dlg.Starting().Attach(func() {
		if parent == nil {
			var rect win.RECT
			win.GetWindowRect(dlg.Handle(), &rect)
			dlgW := rect.Right - rect.Left
			dlgH := rect.Bottom - rect.Top

			var workArea win.RECT
			var screenW, screenH int32
			if win.SystemParametersInfo(spiGetWorkArea, 0, unsafe.Pointer(&workArea), 0) {
				screenW = workArea.Right - workArea.Left
				screenH = workArea.Bottom - workArea.Top
			} else {
				screenW = win.GetSystemMetrics(win.SM_CXSCREEN)
				screenH = win.GetSystemMetrics(win.SM_CYSCREEN)
			}

			x := workArea.Left + (screenW-dlgW)/2
			y := workArea.Top + (screenH-dlgH)/2
			win.SetWindowPos(dlg.Handle(), win.HWND_TOP, x, y, 0, 0, win.SWP_NOSIZE)
			win.SetForegroundWindow(dlg.Handle())
		}
	})

	dlg.Run()
}
