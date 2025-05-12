package cmd

import "errors"

var (
	ErrConfigCreated = errors.New("config created")
	ErrHandled       = errors.New("handled") // 已处理, 不再需要 rootCmd 输出
	ErrAborted       = errors.New("aborted")
)
