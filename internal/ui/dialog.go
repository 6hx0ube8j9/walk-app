package ui

import (
	"github.com/tailscale/walk"
)

// resolveOwner 确保只在父窗口有效且可见时才作为 Owner 传入，避免弹窗被隐藏的主窗口挂起
func resolveOwner(owner walk.Form) walk.Form {
	if owner != nil && owner.Visible() {
		return owner
	}
	return nil
}

// ShowErrorDialog 弹出错误提示框（自带错误图标与 Windows 错误提示音）
func ShowErrorDialog(owner walk.Form, title, message string) {
	walk.MsgBox(
		resolveOwner(owner),
		title,
		message,
		walk.MsgBoxOK|walk.MsgBoxIconError,
	)
}

// ShowConfirmDialog 弹出“是/否”确认提示框（自带询问图标与提示音，返回是否点击“是”）
func ShowConfirmDialog(owner walk.Form, title, message string) bool {
	cmd := walk.MsgBox(
		resolveOwner(owner),
		title,
		message,
		walk.MsgBoxYesNo|walk.MsgBoxIconQuestion,
	)
	return cmd == walk.DlgCmdYes
}
