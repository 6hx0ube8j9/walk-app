package types

// UICommand 交互意图（UI -> Core）
type UICommand struct {
	Action string
}

// UIState 视图状态（Core -> UI，单向被动渲染）
type UIState struct {
	IsNetworkReady bool
	StatusText     string
}

// UIEffect 瞬时副作用（Core -> UI，单次消费，如弹窗、气泡、退出信号）
type UIEffect struct {
	Type    string // "ShowError" 或 "ExitApp"
	Title   string
	Message string
}
