# logm

面向 Go 1.27 的结构化日志 facade，直接复用标准库 `log/slog` 的 Handler
实现。日志序列化、group、error、source 和并发语义均由标准库维护。

## 快速开始

```go
package main

import (
    "log/slog"
    "github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
)

func main() {
    logm.MustInit(logm.PresetProd())
    defer logm.Close()
    slog.Info("started", "port", 8080)
}
```

`logm.New` 返回拥有 sink 生命周期的 `*logm.Logger`，使用完毕应调用
`Close`；它嵌入 `*slog.Logger`，因此完整兼容标准日志 API。

对于 stdout 需要承载 JSON、CSV 或其他机器可读结果的 CLI 命令，使用
`logm.PresetCLI()`，日志会写入 stderr，且保留自动选择的开发/生产格式。

## 配置

```go
package main

import (
    "bytes"
    "context"
    "log/slog"
    "github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
)

func main() {
    var out bytes.Buffer
    logger, err := logm.New(logm.Config{
        Level: slog.LevelDebug,
        Outputs: []logm.Output{{Writer: &out, Format: logm.FormatJSON}},
        AddSource: true,
        TimeFormat: "rfc3339ms",
        Middleware: []logm.Middleware{
            func(ctx context.Context, r slog.Record) (slog.Record, bool) {
                r.AddAttrs(slog.String("service", "api"))
                return r, true
            },
        },
    })
    if err != nil { panic(err) }
    defer logger.Close()
    logger.Info("request", slog.Group("http", slog.Int("status", 200)))
}
```

`Outputs` 可配置多个不同格式的目标；也可把标准 `slog.Handler` 放入
`Config.Handlers`，logm 会用 `slog.NewMultiHandler` 组合它们。可选格式：

- `FormatText`：标准 `slog.TextHandler`
- `FormatJSON`：标准 `slog.JSONHandler`
- `FormatPretty`：标准 TextHandler 加终端级别着色

`ReplaceAttr` 遵循标准 slog 语义，可用于脱敏、时间格式和 source 路径裁剪。
需要由 logger 负责关闭的文件或网络 sink 请设置 `Output.Own: true`。

## 异步与多路 sink

```go
import (
    "github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
    "github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

sink := writer.NewAsync(writer.File("app.log"), writer.AsyncConfig{
    Capacity: 4096,
    Overflow: writer.OverflowBlock,
})
cfg := logm.Config{Outputs: []logm.Output{{Writer: sink, Format: logm.FormatJSON}}}
```

异步 sink 的 `Sync`/`Close` 会 drain 队列并传播底层写错误；`OverflowFail`、
`OverflowDropNewest` 和 `OverflowDropOldest` 可在高负载时选择背压策略。
`writer.Multi` 会尝试写入所有目标并通过 `errors.Join` 汇总错误。

## 环境变量

`LoadConfigFromEnv` 严格解析以下变量：`LOGM_ENV`、`LOGM_LEVEL`、
`LOGM_FORMAT`、`LOGM_OUTPUT`、`LOGM_SOURCE`、`LOGM_TIME_FORMAT`、
`LOGM_TIMEZONE`。非法值直接返回错误，不会静默回退。

## Context 与 request ID

```go
ctx := logm.WithNewRequestID(context.Background()) // UUIDv7
logm.FromContext(ctx).Info("processing", "id", logm.RequestID(ctx))
```
