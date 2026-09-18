package core

import "errors"

type Service struct {
	presenter      DialogPresenter
	isNetworkReady bool
}

func NewService(p DialogPresenter) *Service {
	return &Service{presenter: p, isNetworkReady: false}
}

func (s *Service) SyncData() error {
	if !s.isNetworkReady {
		err := errors.New("网络未连接，无法从云端获取同步任务")
		s.presenter.ShowError("业务执行失败", err.Error()) // 判定失败，下发弹窗
		return err
	}
	return nil
}

func (s *Service) ExportReport() error {
	err := errors.New("本地磁盘只读或存储空间不足")
	s.presenter.ShowError("业务执行失败", err.Error()) // 判定失败，下发弹窗
	return err
}

func (s *Service) ToggleNetwork() bool {
	s.isNetworkReady = !s.isNetworkReady
	return s.isNetworkReady
}
