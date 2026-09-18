package ui

import (
	"walk-app/internal/core"

	"github.com/tailscale/walk"
)

type WalkPresenter struct {
	mw *walk.MainWindow
}

func NewWalkPresenter() *WalkPresenter {
	return &WalkPresenter{}
}

func (p *WalkPresenter) SetWindow(mw *walk.MainWindow) {
	p.mw = mw
}

// ShowError 落地实现 core.DialogPresenter，保证在 Win32 UI 线程弹出
func (p *WalkPresenter) ShowError(title, message string) {
	if p.mw == nil {
		ShowErrorDialog(nil, title, message)
		return
	}
	p.mw.Synchronize(func() {
		ShowErrorDialog(p.mw, title, message)
	})
}

var _ core.DialogPresenter = (*WalkPresenter)(nil)
