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
	Role         string `json:"role" toml:"role"`
	Context      string `json:"context" toml:"context"`
	Capabilities string `json:"capabilities" toml:"capabilities"`
	Constraints  string `json:"constraints" toml:"constraints"`
	Workflow     string `json:"workflow" toml:"workflow"`
	OutputFormat string `json:"output_format" toml:"output_format"`
	FewShot      string `json:"few_shot" toml:"few_shot"`
}

// Format 将 Prompts 自身格式化为最终 Prompt 字符串（支持变量注入）
func (p *Prompts) Format(ctx context.Context) (string, error) {
	pf := NewPromptFormatter()
	basePrompt, err := pf.Format(*p)
	if err != nil {
		return "", err
	}
	rendered := basePrompt
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
	templateData := map[string]string{
		"Role":         prompts.Role,
		"Context":      prompts.Context,
		"Capabilities": prompts.Capabilities,
		"Constraints":  prompts.Constraints,
		"Workflow":     prompts.Workflow,
		"OutputFormat": prompts.OutputFormat,
		"FewShot":      prompts.FewShot,
	}

	tmpl, err := template.New("prompt").Parse(pf.template)
	if err != nil {
		return "", fmt.Errorf("parse template failed: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("execute template failed: %w", err)
	}

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

// buildDefaultTemplate 构建默认模板
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
