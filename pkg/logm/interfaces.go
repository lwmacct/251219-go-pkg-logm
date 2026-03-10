package logm

import (
	"context"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
	writerpkg "github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// Formatter 是 formatter.Formatter 的别名，方便在主包中使用。
//
// Formatter 将日志记录格式化为字节序列。
// 不同的 Formatter 实现决定日志的输出格式，如 JSON、文本或彩色输出。
type Formatter = formatter.Formatter

// Record 是 formatter.Record 的别名，方便在主包中使用。
//
// Record 封装单条日志记录的所有信息，是 Formatter 的输入。
type Record = formatter.Record

// Writer 是 writer.Writer 的别名，方便在主包中使用。
type Writer = writerpkg.Writer

// Interceptor 拦截并可选修改日志记录。
//
// 拦截器在日志写入前被调用，可用于：
//   - 添加通用字段（如 trace_id）
//   - 过滤敏感信息
//   - 采样高频日志
//
// 返回 nil 表示丢弃该条日志。
type Interceptor func(ctx context.Context, r *Record) *Record
