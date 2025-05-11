package errors

import "errors"

var (
	Handled = errors.New("handled") // 已处理, 不再需要 rootCmd 输出
	Aborted = errors.New("aborted")
)
