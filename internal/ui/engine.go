package ui

import (
	"walk-app/internal/types"

	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

type UIEngine struct {
	walkApp  *walk.Application
	view     *MainWindowView
	tray     *TrayManager
	cmdCh    chan<- types.UICommand
	stateCh  <-chan types.UIState
	effectCh <-chan types.UIEffect
}

func NewUIEngine(
	walkApp *walk.Application,
	cmdCh chan<- types.UICommand,
	stateCh <-chan types.UIState,
	effectCh <-chan types.UIEffect,
	title string,
) (*UIEngine, error) {
	engine := &UIEngine{
		walkApp:  walkApp,
		cmdCh:    cmdCh,
		stateCh:  stateCh,
		effectCh: effectCh,
	}

	view, err := CreateMainWindow(cmdCh, title)
	if err != nil {
		return nil, err
	}
	engine.view = view
	view.Window.Hide()

	tray, err := SetupTray(engine, title)
	if err != nil {
		return nil, err
	}
	engine.tray = tray

	// 启动后台监听管道
	go engine.listenState()
	go engine.listenEffect()

	return engine, nil
}

// listenState 单向数据流：收到新 State 后派发到 UI 线程被动渲染
func (e *UIEngine) listenState() {
	for state := range e.stateCh {
		s := state
		e.view.Window.Synchronize(func() {
			_ = e.view.StatusLabel.SetText(s.StatusText)
		})
	}
}

// listenEffect 监听瞬时副作用（如弹窗、退出）
func (e *UIEngine) listenEffect() {
	for eff := range e.effectCh {
		f := eff
		e.view.Window.Synchronize(func() {
			switch f.Type {
			case "ShowError":
				ShowErrorDialog(e.view.Window, f.Title, f.Message)
			case "ExitApp":
				e.Exit()
			}
		})
	}
}

func (e *UIEngine) ToggleWindow() {
	if e.view.Window.Visible() {
		e.view.Window.Hide()
	} else {
		e.view.Window.Show()
		win.ShowWindow(e.view.Window.Handle(), win.SW_RESTORE)
		win.SetForegroundWindow(e.view.Window.Handle())
	}
}

func (e *UIEngine) Exit() {
	if e.tray != nil && e.view != nil {
		e.tray.Dispose(e.view.Window.Handle())
	}
	if e.view != nil {
		e.view.Window.Close()
	}
	e.walkApp.Exit(0)
}
