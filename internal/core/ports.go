package core

// DialogPresenter 规定业务层对外下发弹窗的契约，UI 层负责实现
type DialogPresenter interface {
	ShowError(title, message string)
}
