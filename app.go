package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/tailscale/walk"
)

type ErrorDialogPresenter interface {
	ShowError(owner walk.Form, title, message string)
}

type DefaultDialogPresenter struct{}

func (p *DefaultDialogPresenter) ShowError(owner walk.Form, title, message string) {
	ShowErrorDialog(owner, title, message)
}

type App struct {
	WalkApp   *walk.Application
	MW        *walk.MainWindow
	Tray      *TrayManager
	Presenter ErrorDialogPresenter

	// 状态机字段（模拟业务状态）
	IsNetworkReady bool
}

func NewApp() (*App, error) {
	walkApp, err := walk.InitApp()
	if err != nil {
		return nil, err
	}

	return &App{
		WalkApp:        walkApp,
		Presenter:      &DefaultDialogPresenter{},
		IsNetworkReady: false, // 初始网络设为未连接，模拟触发错误
	}, nil
}

func (a *App) HandleOperation(action string) {
	err := a.executeBusinessLogic(action)
	if err != nil {
		a.ShowError("业务执行失败", fmt.Sprintf("指令 [%s] 异常: %v", action, err))
		return
	}

	log.Printf("[SUCCESS] 操作 [%s] 执行成功\n", action)
}

func (a *App) executeBusinessLogic(action string) error {
	switch action {
	case "sync_data":
		if !a.IsNetworkReady {
			return errors.New("网络未连接，无法从云端获取同步任务")
		}
	case "export_report":
		return errors.New("本地磁盘只读或存储空间不足")
	case "toggle_network":
		a.IsNetworkReady = !a.IsNetworkReady
		log.Printf("网络状态已切换为: %v\n", a.IsNetworkReady)
		return nil
	default:
		return errors.New("不支持的操作类型")
	}
	return nil
}

func (a *App) ShowError(title, message string) {
	log.Printf("[ERROR] %s: %s\n", title, message)

	if a.MW == nil {
		a.Presenter.ShowError(nil, title, message)
		return
	}

	a.MW.Synchronize(func() {
		a.Presenter.ShowError(a.MW, title, message)
	})
}

func (a *App) Run() {
	a.WalkApp.Run()
}
