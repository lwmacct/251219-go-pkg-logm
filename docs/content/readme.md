# Logger 包

基于 Go 1.21+ `log/slog` 的统一结构化日志系统。

## 特性

- 支持多种输出格式：JSON、Text、Colored（彩色终端）
- 灵活的日志级别控制（DEBUG、INFO、WARN、ERROR）
- 多种时间格式配置
- Context 集成，支持请求链路追踪
- 彩色 Handler 支持 JSON/map/struct 自动平铺
- WithGroup 分组支持（嵌套分组自动添加前缀）
- 配置验证（无效格式/级别会返回错误）
- 文件输出支持，带关闭机制

## 快速开始

### 基础用法

```go
package main

import (
    "github.com/lwmacct/250901-m-nbwb/internal/infrastructure/logger"
)

func main() {
    // 使用默认配置初始化
    logger.Init(nil)

    // 使用全局 logger
    logger.Info("server started", "port", 8080)
    logger.Debug("debug info", "key", "value")
    logger.Warn("warning message")
    logger.Error("error occurred", "error", err)
}
```

### 从环境变量初始化

```go
// 支持的环境变量：
// - LOG_LEVEL: DEBUG, INFO, WARN, ERROR
// - LOG_FORMAT: json, text, color
// - LOG_OUTPUT: stdout, stderr, 或文件路径
// - LOG_ADD_SOURCE: true, false
// - LOG_TIME_FORMAT: rfc3339, rfc3339ms, unix, unixms, datetime

logger.InitFromEnv()
```

### 自定义配置

```go
cfg := &logger.Config{
    Level:      "DEBUG",
    Format:     "json",        // json, text, color
    Output:     "stdout",      // stdout, stderr, /path/to/file.log
    AddSource:  true,          // 添加源码位置
    TimeFormat: "rfc3339ms",   // 时间格式
}
logger.Init(cfg)
```

## 输出格式

### JSON 格式

```json
{ "time": "2024-01-15T10:30:00.123+08:00", "level": "INFO", "msg": "request received", "method": "GET", "path": "/api" }
```

### Text 格式

```
time=2024-01-15T10:30:00.123+08:00 level=INFO msg="request received" method=GET path=/api
```

### Colored 格式（终端）

```json
{ "time": "2024-01-15 10:30:00.123", "level": "INFO", "msg": "request received", "method": "GET", "path": "/api" }
```

带有颜色高亮：

- DEBUG: 蓝色
- INFO: 绿色
- WARN: 黄色
- ERROR: 红色

## 高级功能

### 自动平铺（JSON/map/struct）

彩色 Handler 会自动将复杂类型平铺为 `key.subkey` 格式。

**JSON 字符串：**

```go
logger.Info("request", "body", `{"user":"alice","age":30}`)
// 输出: {"msg":"request","body.user":"alice","body.age":"30"}
```

**map[string]any：**

```go
logger.Info("request", "data", map[string]any{"user": "bob", "active": true})
// 输出: {"msg":"request","data.user":"bob","data.active":"true"}
```

**struct：**

```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}
logger.Info("user", "info", User{Name: "charlie", Age: 30})
// 输出: {"msg":"user","info.name":"charlie","info.age":"30"}
```

**嵌套和数组：**

```go
logger.Info("data", "payload", `{"user":{"name":"bob"},"tags":["go","rust"]}`)
// 输出: {"msg":"data","payload.user.name":"bob","payload.tags[0]":"go","payload.tags[1]":"rust"}
```

**slog.Group：**

```go
logger.Info("test", slog.Group("request", "method", "GET", "path", "/api"))
// 输出: {"msg":"test","request.method":"GET","request.path":"/api"}
```

### Context 集成

```go
// 将 logger 存入 context
ctx := logger.WithLogger(ctx, customLogger)

// 从 context 获取 logger
log := logger.FromContext(ctx)

// 便捷方法：添加 request_id
ctx = logger.WithRequestID(ctx, "req-123")
```

### 创建独立 Logger

```go
// 为特定模块创建独立配置的 logger
moduleLogger, err := logger.New(&logger.Config{
    Level:  "DEBUG",
    Format: "json",
    Output: "/var/log/module.log",
})

// 如需手动关闭文件
moduleLogger, closer, err := logger.NewWithCloser(cfg)
defer closer.Close()
```

### 带属性的 Logger

```go
// 添加固定属性
log := logger.WithAttrs("service", "api", "version", "1.0")
log.Info("started")  // 每条日志都会包含 service 和 version

// 分组
log := logger.WithGroup("request")
log.Info("received", "method", "GET")  // method 在 request 分组下
```

### 辅助函数

```go
// 格式化字节数
logger.FormatBytes(1536 * 1024)  // "1.5 MB"

// 记录错误并返回
return logger.LogError(ctx, "operation failed", err, "user_id", userID)

// 记录并包装错误
return logger.LogAndWrap("fetch failed", err, "url", url)
```

## 时间格式

| 格式        | 示例                            |
| ----------- | ------------------------------- |
| `rfc3339`   | `2024-01-15T10:30:00+08:00`     |
| `rfc3339ms` | `2024-01-15T10:30:00.123+08:00` |
| `datetime`  | `2024-01-15 10:30:00`           |
| `unix`      | `1705285800`                    |
| `unixms`    | `1705285800123`                 |
| `unixfloat` | `1705285800.123`                |

## 资源管理

输出到文件时，应在程序退出时关闭：

```go
func main() {
    logger.Init(&logger.Config{
        Output: "/var/log/app.log",
    })
    defer logger.Close()  // 确保文件正确关闭

    // ...
}
```

## 日志级别

支持大小写不敏感：

```go
logger.Init(&logger.Config{Level: "debug"})  // 等同于 "DEBUG"
logger.Init(&logger.Config{Level: "Info"})   // 等同于 "INFO"
logger.Init(&logger.Config{Level: "WARNING"}) // 等同于 "WARN"
```

## 彩色 Handler 配置

```go
config := &logger.ColoredHandlerConfig{
    Level:        slog.LevelInfo,
    AddSource:    true,           // 添加源码位置
    EnableColor:  true,           // 启用颜色
    CallerClip:   "/app/",        // 裁剪路径前缀
    PriorityKeys: []string{"time", "level", "msg"},  // 优先显示的字段
    TrailingKeys: []string{"source"},                 // 末尾显示的字段
    TimeFormat:   "datetime",     // 时间格式（见时间格式表）
}
handler := logger.NewColoredHandler(os.Stdout, config)
```

### WithGroup 分组

```go
// 单层分组
log := slog.New(handler).WithGroup("request")
log.Info("received", "method", "GET")
// 输出: {"msg":"received","request.method":"GET"}

// 嵌套分组
log := slog.New(handler).WithGroup("http").WithGroup("request")
log.Info("received", "method", "POST")
// 输出: {"msg":"received","http.request.method":"POST"}
```

## 配置验证

Config 提供 `Validate()` 方法，在 `Init()` 和 `New()` 时自动调用：

```go
cfg := &logger.Config{
    Level:  "TRACE",        // 无效级别
    Format: "yaml",         // 无效格式
}

// 验证会返回详细错误
err := cfg.Validate()
// err: invalid log format: "yaml", valid options: json, text, color

// Init/New 也会自动验证
err := logger.Init(cfg)
// err: invalid log level: "TRACE", valid options: DEBUG, INFO, WARN, ERROR
```

有效的配置选项：

- **Level**: `DEBUG`, `INFO`, `WARN`, `WARNING`, `ERROR`（大小写不敏感）
- **Format**: `json`, `text`, `color`, `colored`

## 最佳实践

1. **应用启动时初始化一次**

   ```go
   func main() {
       logger.InitFromEnv()
       defer logger.Close()
   }
   ```

2. **使用结构化日志**

   ```go
   // 好
   logger.Info("user login", "user_id", userID, "ip", ip)

   // 避免
   logger.Info(fmt.Sprintf("user %s login from %s", userID, ip))
   ```

3. **传递 Context**

   ```go
   func HandleRequest(ctx context.Context) {
       log := logger.FromContext(ctx)
       log.Info("processing request")
   }
   ```

4. **使用适当的日志级别**
   - DEBUG: 开发调试信息
   - INFO: 重要业务事件
   - WARN: 警告，但不影响运行
   - ERROR: 错误，需要关注

## 测试

```bash
go test ./internal/infrastructure/logger/... -v
```
