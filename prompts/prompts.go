package prompts

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/liweiming-nova/open_code/variable"
)

// Prompts 结构化 Prompt 定义
type Prompts struct {
	Role         string `json:"role"`
	Context      string `json:"context"`
	Capabilities string `json:"capabilities"`
	Constraints  string `json:"constraints"` // 修正拼写
	Workflow     string `json:"workflow"`
	OutputFormat string `json:"output_format"`
	FewShot      string `json:"few_shot"`
}

// Format 将 Prompts 自身格式化为最终 Prompt 字符串（支持变量注入）
// - 先使用 PromptFormatter 根据结构化字段生成基础 Prompt
// - 再使用 Prompts.Variables 进行自定义占位符替换（{var_name}）
// - 最后交给 SecureFormatter 做运行时变量解析与结构安全校验
func (p *Prompts) Format(ctx context.Context) (string, error) {
	// 1. 使用结构化模板生成基础 Prompt
	pf := NewPromptFormatter()
	basePrompt, err := pf.Format(*p)
	if err != nil {
		return "", err
	}
	rendered := basePrompt
	// 3. 使用 SecureFormatter 做运行时变量解析 + 结构校验
	sf := NewSecureFormatter(variable.GlobalRegistry).WithStrictMode(true)
	return sf.Format(ctx, rendered)
}

// PromptFormatter Prompt 格式化器
type PromptFormatter struct {
	template string
}

// NewPromptFormatter 创建格式化器
func NewPromptFormatter() *PromptFormatter {
	return &PromptFormatter{
		template: buildDefaultTemplate(),
	}
}

// Format 将结构化数据格式化为 Prompt 字符串
func (pf *PromptFormatter) Format(prompts Prompts) (string, error) {

	// 1. 构建模板数据
	templateData := map[string]string{
		"Role":         prompts.Role,
		"Context":      prompts.Context,
		"Capabilities": prompts.Capabilities,
		"Constraints":  prompts.Constraints,
		"Workflow":     prompts.Workflow,
		"OutputFormat": prompts.OutputFormat,
		"FewShot":      prompts.FewShot,
	}

	// 2. 渲染模板
	tmpl, err := template.New("prompt").Parse(pf.template)
	if err != nil {
		return "", fmt.Errorf("parse template failed: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("execute template failed: %w", err)
	}

	// 3. 清理多余空行
	return pf.cleanupOutput(buf.String()), nil
}

// cleanupOutput 清理多余空行
func (pf *PromptFormatter) cleanupOutput(text string) string {
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

	return strings.Join(cleaned, "\n")
}

// buildDefaultTemplate 构建默认模板（匹配您的 analystPrompt 风格）
func buildDefaultTemplate() string {
	return `{{if .Role}}
# {{.Role}}
{{end}}
{{if .Context}}
## Profile
{{.Context}}
{{end}}
{{if .Capabilities}}
## 核心能力
{{.Capabilities}}
{{end}}
{{if .Constraints}}
## 核心原则
{{.Constraints}}
{{end}}
{{if .Workflow}}
## 工作流程
{{.Workflow}}
{{end}}
{{if .OutputFormat}}
## 输出规范
{{.OutputFormat}}
{{end}}
{{if .FewShot}}
## 参考示例
{{.FewShot}}
{{end}}
`
}
