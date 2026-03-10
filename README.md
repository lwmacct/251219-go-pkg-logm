# logm

基于 Go 1.26+ `log/slog` 的结构化日志库。

[![License](https://img.shields.io/github/license/lwmacct/251219-go-pkg-logm)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/lwmacct/251219-go-pkg-logm.svg)](https://pkg.go.dev/github.com/lwmacct/251219-go-pkg-logm)
[![Go CI](https://github.com/lwmacct/251219-go-pkg-logm/actions/workflows/go-ci.yml/badge.svg)](https://github.com/lwmacct/251219-go-pkg-logm/actions/workflows/go-ci.yml)
[![codecov](https://codecov.io/gh/lwmacct/251219-go-pkg-logm/branch/main/graph/badge.svg)](https://codecov.io/gh/lwmacct/251219-go-pkg-logm)
[![Go Report Card](https://goreportcard.com/badge/github.com/lwmacct/251219-go-pkg-logm)](https://goreportcard.com/report/github.com/lwmacct/251219-go-pkg-logm)
[![GitHub Tag](https://img.shields.io/github/v/tag/lwmacct/251219-go-pkg-logm?sort=semver)](https://github.com/lwmacct/251219-go-pkg-logm/tags)

## 特性

- **标准库优先**：初始化后直接使用 `slog.Info()` / `slog.Error()`
- **显式配置**：使用 `logm.Config`，不再依赖大量链式 option
- **预设开箱即用**：`PresetDefault` / `PresetDev` / `PresetProd` / `PresetAuto` / `PresetFromEnv`
- **输出可组合**：支持 `writer.Multi(...)`、`writer.Async(...)`、文件轮转
- **多 Handler**：支持 Go 1.26 `slog.NewMultiHandler(...)`
- **动态级别**：支持 `slog.LevelVar`

## 安装

```bash
go get github.com/lwmacct/251219-go-pkg-logm
```

## 快速开始

最简单的方式是直接使用预设配置：

```go
package main

import (
	"log/slog"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
)

func main() {
	logm.MustInit(logm.PresetAuto())
	defer logm.Close()

	slog.Info("started", "port", 8080)
}
```

## 配置模型

当前版本使用显式 `Config`：

```go
type Config struct {
    Level        string
    LevelVar     *slog.LevelVar
    Formatter    logm.Formatter
    Output       logm.Writer
    SlogHandlers []slog.Handler
    AddSource    bool
    TimeFormat   string
    Timezone     string
    Location     *time.Location
    Interceptors []logm.Interceptor
}
```

如果你想从默认值开始调整：

```go
cfg := logm.PresetDefault()
cfg.Level = "DEBUG"
```

## 预设配置

- `logm.PresetDefault()`：默认基础配置
- `logm.PresetDev()`：开发环境，彩色输出、DEBUG、带 source
- `logm.PresetProd()`：生产环境，JSON 输出、INFO、UTC 时间
- `logm.PresetAuto()`：根据环境自动选择 dev / prod
- `logm.PresetFromEnv()`：从环境变量生成配置

`PresetFromEnv()` 支持：

- `LOGM_ENV`
- `LOGM_LEVEL`
- `LOGM_FORMAT`
- `LOGM_OUTPUT`
- `LOGM_SOURCE`
- `LOGM_TIME_FORMAT`

其中 `LOGM_OUTPUT` 支持逗号分隔，例如：

```bash
LOGM_OUTPUT=stdout,/var/log/app.json.log
```

## 自定义配置

```go
package main

import (
	"log/slog"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

func main() {
	cfg := logm.Config{
		Level:     "DEBUG",
		Formatter: formatter.ColorText(),
		Output: writer.Multi(
			writer.Stdout(),
			writer.File("/var/log/app.log", writer.WithRotation(100, 7)),
		),
		AddSource:  true,
		TimeFormat: "time",
		Timezone:   "Asia/Shanghai",
	}

	logm.MustInit(cfg)
	defer logm.Close()

	slog.Info("request done", "status", 200)
}
```

## 多路输出

同一种格式输出到多个目标，推荐直接组合 `writer.Multi(...)`：

```go
cfg := logm.Config{
	Output: writer.Multi(
		writer.Stdout(),
		writer.File("/var/log/app.log"),
	),
}
```

如果你更喜欢字符串形式，也可以：

```go
cfg := logm.PresetProd()
cfg.Output = logm.ParseOutput("stdout,/var/log/app.log")
```

## Go 1.26 MultiHandler

如果需要“不同格式”同时输出，使用 `Config.SlogHandlers`：

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

	logger := logm.New(logm.Config{
		Output: writer.Stdout(),
		SlogHandlers: []slog.Handler{
			slog.NewJSONHandler(file, nil),
		},
	})

	logger.Info("hello", "user", "alice")
}
```

上面这条日志会：

- 走 `logm` 的文本输出到终端
- 同时走标准库 `JSONHandler` 输出到文件

## 动态级别

```go
levelVar := &slog.LevelVar{}
levelVar.Set(slog.LevelInfo)

logger := logm.New(logm.Config{
	LevelVar: levelVar,
})

logger.Debug("hidden")

levelVar.Set(slog.LevelDebug)
logger.Debug("visible")
```

全局默认 logger 也可以直接用：

```go
logm.SetLevel("DEBUG")
```

## Context 集成

```go
ctx := context.Background()
ctx = logm.WithRequestID(ctx, "req-12345")

log := logm.FromContext(ctx)
log.Info("processing request")
```

## 输出子包

`pkg/logm/writer` 提供常用输出实现：

- `writer.Stdout()`
- `writer.Stderr()`
- `writer.File(path, opts...)`
- `writer.Async(w, bufferSize)`
- `writer.Multi(w1, w2, ...)`

## 格式化子包

`pkg/logm/formatter` 提供常用格式化器：

- `formatter.Text()`
- `formatter.JSON()`
- `formatter.ColorText()`
- `formatter.ColorJSON()`

## 文档

```bash
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer
```

在线文档：

- https://pkg.go.dev/github.com/lwmacct/251219-go-pkg-logm/pkg/logm
