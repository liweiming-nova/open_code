package prompts

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/liweiming-nova/open_code/variable"
)

// RuntimeResolver 运行时变量解析器
type RuntimeResolver struct {
	registry *variable.VariableRegistry
	// 占位符正则: 匹配 {var_name} 格式
	placeholderRegex *regexp.Regexp
}

// NewRuntimeResolver 创建解析器
func NewRuntimeResolver(registry *variable.VariableRegistry) *RuntimeResolver {
	return &RuntimeResolver{
		registry:         registry,
		placeholderRegex: regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`),
	}
}

// ResolveResult 解析结果
type ResolveResult struct {
	Values  map[string]string // 成功解析的变量
	Errors  map[string]error  // 解析失败的变量及错误
	Unknown []string          // 未注册的变量名
}

// Resolve 解析文本中的所有变量占位符
func (rr *RuntimeResolver) Resolve(ctx context.Context, text string) (*ResolveResult, error) {
	result := &ResolveResult{
		Values:  make(map[string]string),
		Errors:  make(map[string]error),
		Unknown: make([]string, 0),
	}

	// 1. 找出所有占位符
	matches := rr.placeholderRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return result, nil // 无需解析
	}

	// 2. 去重
	varNames := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			varNames[match[1]] = true
		}
	}

	// 3. 逐个解析
	for varName := range varNames {
		meta, ok := rr.registry.Get(varName)
		if !ok {
			// 未注册的变量
			result.Unknown = append(result.Unknown, varName)
			continue
		}

		// 调用 Getter 获取值
		value, err := meta.Getter(ctx)
		if err != nil {
			// 获取失败：尝试使用默认值
			if meta.Default != "" {
				value = meta.Default
			} else if meta.Required {
				result.Errors[varName] = fmt.Errorf("required variable '%s' get failed: %w", varName, err)
				continue
			} else {
				// 非必填且无默认值：跳过替换
				continue
			}
		}

		// 可选：执行校验
		if meta.Validator != nil {
			if err := meta.Validator(value); err != nil {
				result.Errors[varName] = fmt.Errorf("variable '%s' validation failed: %w", varName, err)
				continue
			}
		}

		result.Values[varName] = value
	}

	// 4. 如果有必填变量解析失败，返回错误
	if len(result.Errors) > 0 {
		return result, fmt.Errorf("variable resolution errors: %v", result.Errors)
	}

	return result, nil
}

// Inject 将解析结果注入到文本中
func (rr *RuntimeResolver) Inject(text string, values map[string]string) string {
	result := text
	for varName, value := range values {
		placeholder := fmt.Sprintf("{%s}", varName)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
