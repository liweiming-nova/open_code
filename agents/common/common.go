package common

// agent 通用设置
import (
	"context"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultMaxIterations = 60
	DefaultMaxRetries    = 3
)

var (
	// MaxIterations 可通过 OPEN_CODE_AGENT_MAX_ITERATIONS 覆盖
	MaxIterations = DefaultMaxIterations
	// MaxRetries 可通过 OPEN_CODE_AGENT_MAX_RETRIES 覆盖
	MaxRetries = DefaultMaxRetries
)

func init() {
	MaxIterations = getEnvPositiveInt("OPEN_CODE_AGENT_MAX_ITERATIONS", DefaultMaxIterations)
	MaxRetries = getEnvPositiveInt("OPEN_CODE_AGENT_MAX_RETRIES", DefaultMaxRetries)
}

func getEnvPositiveInt(key string, fallback int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func IsRetryAble(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}

	// context 已被取消或超时，不应重试
	if ctx.Err() != nil {
		return false
	}

	// 网络错误（超时、连接中断等）
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "status code: 429") ||
		strings.Contains(msg, "status code: 500") ||
		strings.Contains(msg, "status code: 502") ||
		strings.Contains(msg, "status code: 503") ||
		strings.Contains(msg, "status code: 504") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF")
}
