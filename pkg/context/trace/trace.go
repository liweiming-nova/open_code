package trace

import (
	"context"

	"github.com/google/uuid" // 需要安装：go get github.com/google/uuid
)

// contextKey 使用自定义类型定义 context key，防止键名冲突
type contextKey string

const (
	// TraceIDKey 定义 trace_id 的键
	TraceIDKey contextKey = "trace_id"
)

// SetTraceID 将 traceID 存入 context
func SetTraceID(ctx context.Context, traceID string) context.Context {
	// 直接使用 context.WithValue，无需额外的 SetMetaValue 包装
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从 context 获取 traceID，如果不存在则生成一个新的
func GetTraceID(ctx context.Context) string {
	// 1. 获取值
	val := ctx.Value(TraceIDKey)

	// 2. 检查是否为 nil
	if val == nil {
		return uuid.New().String()
	}

	// 3. 类型断言 (interface{} -> string)
	traceID, ok := val.(string)

	// 4. 如果断言失败或字符串为空，生成新的 UUID
	if !ok || traceID == "" {
		return uuid.New().String()
	}

	return traceID
}
