package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/liweiming-nova/open_code/variable"
)

// SecureFormatter 安全 Prompt 格式化器，支持运行时变量注入与结构校验
type SecureFormatter struct {
	registry *variable.VariableRegistry
	resolver *RuntimeResolver
	// 是否严格模式：未知变量报错 vs 保留原样
	strictMode bool
}

// NewSecureFormatter 创建格式化器
func NewSecureFormatter(registry *variable.VariableRegistry) *SecureFormatter {
	return &SecureFormatter{
		registry:   registry,
		resolver:   NewRuntimeResolver(registry),
		strictMode: true, // 默认严格模式
	}
}

// WithStrictMode 设置严格模式
func (sf *SecureFormatter) WithStrictMode(strict bool) *SecureFormatter {
	sf.strictMode = strict
	return sf
}

// Format 执行完整的"解析→校验→注入→格式化"流程
// templateText 中的占位符使用 {var_name} 形式，具体变量需在 VariableRegistry 中注册
func (sf *SecureFormatter) Format(ctx context.Context, templateText string) (string, error) {
	// 1. 运行时解析变量
	resolveResult, err := sf.resolver.Resolve(ctx, templateText)
	if err != nil && sf.strictMode {
		return "", fmt.Errorf("resolve variables failed: %w", err)
	}

	// 2. 严格模式下：未知变量报错
	if sf.strictMode && len(resolveResult.Unknown) > 0 {
		return "", fmt.Errorf("unknown variables not registered: %v", resolveResult.Unknown)
	}

	// 3. 注入变量值
	rendered := sf.resolver.Inject(templateText, resolveResult.Values)

	// 4. 后置校验：检查 Prompt 结构完整性
	if err := validatePromptStructure(rendered); err != nil {
		return "", fmt.Errorf("prompt structure validation failed: %w", err)
	}

	// 5. 清理多余空行
	return cleanupOutput(rendered), nil
}

// FormatWithFallback 带降级保护的 Format
func (sf *SecureFormatter) FormatWithFallback(ctx context.Context, templateText, fallbackPrompt string) string {
	result, err := sf.Format(ctx, templateText)
	if err != nil {
		// 这里使用标准日志输出，避免在库代码中引入额外依赖
		fmt.Printf("Prompt format failed, using fallback: %v\n", err)
		return fallbackPrompt
	}
	return result
}

// validatePromptStructure 校验 Prompt 结构（防注入）
func validatePromptStructure(prompt string) error {
	// 简单检查：是否有异常多的顶级标题
	lines := strings.Split(prompt, "\n")
	headerCount := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			headerCount++
		}
	}
	if headerCount > 20 {
		return fmt.Errorf("abnormal prompt structure: too many headers (%d)", headerCount)
	}

	// 检查是否有常见注入关键词
	injectionKeywords := []string{
		"忽略以上指令",
		"ignore above instructions",
		"system prompt override",
	}
	promptLower := strings.ToLower(prompt)
	for _, kw := range injectionKeywords {
		if strings.Contains(promptLower, strings.ToLower(kw)) {
			return fmt.Errorf("potential prompt injection detected: %s", kw)
		}
	}

	return nil
}

// cleanupOutput 清理多余空行
func cleanupOutput(text string) string {
	lines := strings.Split(text, "\n")
	var cleaned []string
	var prevEmpty bool

	for _, line := range lines {
		isEmpty := strings.TrimSpace(line) == ""
		if isEmpty {
			if !prevEmpty {
				cleaned = append(cleaned, line)
			}
			prevEmpty = true
		} else {
			cleaned = append(cleaned, line)
			prevEmpty = false
		}
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
