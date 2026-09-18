package core

import (
	"context"
	"walk-app/internal/types"
)

type Service struct {
	cmdCh    <-chan types.UICommand
	stateCh  chan types.UIState // 改为双向 channel，允许在丢旧帧时读取
	effectCh chan<- types.UIEffect

	// 领域业务状态
	isNetworkReady bool
}

func NewService(cmdCh <-chan types.UICommand, stateCh chan types.UIState, effectCh chan<- types.UIEffect) *Service {
	return &Service{
		cmdCh:          cmdCh,
		stateCh:        stateCh,
		effectCh:       effectCh,
		isNetworkReady: false,
	}
}

// Run 启动核心业务事件循环
func (s *Service) Run(ctx context.Context) {
	// 推送初始状态
	s.emitState()

	for {
		select {
		case <-ctx.Done():
			return
		case cmd, ok := <-s.cmdCh:
			if !ok {
				return
			}
			s.handleCommand(cmd)
		}
	}
}

func (s *Service) handleCommand(cmd types.UICommand) {
	switch cmd.Action {
	case "sync_data":
		if !s.isNetworkReady {
			s.effectCh <- types.UIEffect{
				Type:    "ShowError",
				Title:   "业务执行失败",
				Message: "网络未连接，无法从云端获取同步任务",
			}
			return
		}
		s.emitState()

	case "export_report":
		s.effectCh <- types.UIEffect{
			Type:    "ShowError",
			Title:   "业务执行失败",
			Message: "本地磁盘只读或存储空间不足",
		}

	case "toggle_network":
		s.isNetworkReady = !s.isNetworkReady
		s.emitState()

	case "exit_app":
		// 业务层做完收尾后下发退出指令
		s.effectCh <- types.UIEffect{Type: "ExitApp"}
	}
}

func (s *Service) emitState() {
	status := "网络状态: 未就绪 (点击同步会报错)"
	if s.isNetworkReady {
		status = "网络状态: 正常连接 (点击同步将成功)"
	}

	state := types.UIState{
		IsNetworkReady: s.isNetworkReady,
		StatusText:     status,
	}

	// 丢旧帧保最新：容量为 1 的管道满时，挤掉旧值换入最新值
	select {
	case s.stateCh <- state:
	default:
		select {
		case <-s.stateCh: // 读出旧帧腾出位置
		default:
		}
		s.stateCh <- state
	}
}
