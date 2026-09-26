package ui

import (
	"walk-app/internal/types"

	"github.com/tailscale/walk"
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

	// 1. 创建主窗口 (对应 window.go)
	view, err := CreateMainWindow(cmdCh, title)
	if err != nil {
		return nil, err
	}
	engine.view = view
	view.Window.Hide()

	bounds := view.Window.Bounds()
	view.Window.SetBounds(bounds)
	// 2. 初始化托盘 (对应 tray.go)
	tray, err := SetupTray(engine, title)
	if err != nil {
		return nil, err
	}
	engine.tray = tray

	// 3. 启动后台异步监听
	go engine.listenState()
	go engine.listenEffect()

	return engine, nil
}

func (e *UIEngine) listenState() {
	for state := range e.stateCh {
		s := state
		if e.view != nil && e.view.Window != nil && e.view.StatusLabel != nil {
			e.view.Window.Synchronize(func() {
				_ = e.view.StatusLabel.SetText(s.StatusText)
			})
		}
	}
}

func (e *UIEngine) listenEffect() {
	for eff := range e.effectCh {
		f := eff
		if e.view != nil && e.view.Window != nil {
			e.view.Window.Synchronize(func() {
				switch f.Type {
				case "ExitApp":
					e.Exit()
				}
			})
		}
	}
}

func (e *UIEngine) ToggleWindow() {
	if e.view == nil {
		return
	}
	if e.view.Window.Visible() {
		e.view.Window.Hide()
	} else {
		e.view.Wake()  
	}
}

func (e *UIEngine) Exit() {
	if e.tray != nil && e.view != nil && e.view.Window != nil {
		e.tray.Dispose(e.view.Window.Handle())
	}
	if e.view != nil && e.view.Window != nil {
		e.view.Window.Close()
	}
	if e.walkApp != nil {
		e.walkApp.Exit(0)
	}
}
