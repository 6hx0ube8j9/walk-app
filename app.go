package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/tailscale/walk"
)

type ErrorPresenter interface {
	ShowError(owner walk.Form, title, message string)
}

type DefaultErrorPresenter struct{}

func (p *DefaultErrorPresenter) ShowError(owner walk.Form, title, message string) {
	ShowErrorDialog(owner, title, message)
}

type App struct {
	WalkApp   *walk.Application
	MW        *walk.MainWindow
	Tray      *TrayManager
	Presenter ErrorPresenter

	// 模拟应用运行状态
	IsNetworkReady bool
}

func NewApp() (*App, error) {
	walkApp, err := walk.InitApp()
	if err != nil {
		return nil, err
	}

	return &App{
		WalkApp:        walkApp,
		Presenter:      &DefaultErrorPresenter{},
		IsNetworkReady: false, // 模拟未就绪状态
	}, nil
}

// HandleOperation 统一业务调度入口：UI 层只管触发，App 负责判定结果
func (a *App) HandleOperation(action string) {
	// 执行业务判定
	err := a.executeBusinessLogic(action)
	if err != nil {
		// 判定为错误，统一下发弹窗
		a.ShowError("业务执行失败", fmt.Sprintf("指令 [%s] 无法完成: %v", action, err))
		return
	}

	log.Printf("[SUCCESS] 指令 [%s] 执行成功\n", action)
}

// executeBusinessLogic 纯业务判定逻辑（可脱离 UI 单独跑测试）
func (a *App) executeBusinessLogic(action string) error {
	switch action {
	case "sync_data":
		if !a.IsNetworkReady {
			return errors.New("网络未连接，无法同步远端数据")
		}
	case "export_report":
		return errors.New("存储空间不足，导出被拒绝")
	default:
		return errors.New("未知操作指令")
	}
	return nil
}

func (a *App) ShowError(title, message string) {
	log.Printf("[ERROR DIALOG] %s: %s\n", title, message)

	if a.MW == nil {
		a.Presenter.ShowError(nil, title, message)
		return
	}

	a.MW.Synchronize(func() {
		a.Presenter.ShowError(a.MW, title, message)
	})
}
