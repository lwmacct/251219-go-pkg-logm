# logm

基于 Go 1.26+ `log/slog` 的结构化日志库。

[![License](https://img.shields.io/github/license/lwmacct/251219-go-pkg-logm)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/lwmacct/251219-go-pkg-logm.svg)](https://pkg.go.dev/github.com/lwmacct/251219-go-pkg-logm)
[![Go CI](https://github.com/lwmacct/251219-go-pkg-logm/actions/workflows/go-ci.yml/badge.svg)](https://github.com/lwmacct/251219-go-pkg-logm/actions/workflows/go-ci.yml)
[![codecov](https://codecov.io/gh/lwmacct/251219-go-pkg-logm/branch/main/graph/badge.svg)](https://codecov.io/gh/lwmacct/251219-go-pkg-logm)
[![Go Report Card](https://goreportcard.com/badge/github.com/lwmacct/251219-go-pkg-logm)](https://goreportcard.com/report/github.com/lwmacct/251219-go-pkg-logm)
[![GitHub Tag](https://img.shields.io/github/v/tag/lwmacct/251219-go-pkg-logm?sort=semver)](https://github.com/lwmacct/251219-go-pkg-logm/tags)

## 为什么选择 logm

- **零侵入设计**：初始化后可直接使用标准库 `slog.Info()` 等函数，业务代码无需依赖 logm
- **可插拔架构**：Handler + Formatter + Writer 分离，按需组合
- **生产就绪**：支持日志轮转、异步写入、动态级别调整

## 安装

```bash
go get github.com/lwmacct/251219-go-pkg-logm
```

## 快速开始

```go
package main

import (
    "log/slog"
    "github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
)

func main() {
    // 一次性初始化（失败时 panic）
    logm.MustInit(logm.PresetAuto()...)
    defer logm.Close()

    // 之后直接使用标准库 slog
    slog.Info("started", "port", 8080)
}
```

## Go 1.26 多路输出

Go 1.26 新增了标准库 `slog.NewMultiHandler(...)`。
`logm` 现在提供 `WithSlogHandler` / `WithSlogHandlers`，可以把 `logm` 自己的 Handler
和标准库、第三方 `slog.Handler` 一起组合。

```go
package main

import (
    "log/slog"
    "os"

    "github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
    "github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

func main() {
    file, err := os.Create("app.json.log")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    logger := logm.New(
        logm.WithWriter(writer.Stdout()),
        logm.WithSlogHandler(slog.NewJSONHandler(file, nil)),
    )

    logger.Info("hello", "user", "alice")
}
```

上面这条日志会同时输出到：

- 终端（通过 `logm` 默认文本格式）
- 文件（通过标准库 `slog.JSONHandler` 输出 JSON）

## 文档

完整 API 文档和使用示例：

```bash
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer
```

或在线查看：[pkg.go.dev](https://pkg.go.dev/github.com/lwmacct/251219-go-pkg-logm/pkg/logm)
