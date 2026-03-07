package variable

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// VariableGetter 变量获取函数类型（支持 ctx 传递）
type VariableGetter func(ctx context.Context) (string, error)

// VariableValidator 变量校验函数类型
type VariableValidator func(value string) error

// VariableMeta 变量元数据
type VariableMeta struct {
	Name        string            // 变量名（不含 {}）
	Description string            // 描述（用于文档/调试）
	Getter      VariableGetter    // 获取值的函数（硬编码可信逻辑）
	Validator   VariableValidator // 可选：校验规则
	Default     string            // 可选：获取失败时的默认值
	Required    bool              // 是否必填
}

// VariableRegistry 变量注册中心（单例）
type VariableRegistry struct {
	variables map[string]*VariableMeta
}

// GlobalRegistry 全局注册中心实例
var GlobalRegistry = NewVariableRegistry()

// NewVariableRegistry 创建注册中心
func NewVariableRegistry() *VariableRegistry {
	reg := &VariableRegistry{
		variables: make(map[string]*VariableMeta),
	}
	// 初始化时注册系统预定义变量
	reg.registerSystemVariables()
	return reg
}

// Register 注册一个新变量（支持业务自定义扩展）
func (vr *VariableRegistry) Register(meta VariableMeta) error {
	if meta.Name == "" {
		return fmt.Errorf("variable name cannot be empty")
	}
	if vr.variables[meta.Name] != nil {
		return fmt.Errorf("variable '%s' already registered", meta.Name)
	}
	vr.variables[meta.Name] = &meta
	return nil
}

// Get 获取变量元数据
func (vr *VariableRegistry) Get(name string) (*VariableMeta, bool) {
	v, ok := vr.variables[name]
	return v, ok
}

// List 列出所有已注册变量（用于文档/调试）
func (vr *VariableRegistry) List() []VariableMeta {
	result := make([]VariableMeta, 0, len(vr.variables))
	for _, v := range vr.variables {
		result = append(result, *v)
	}
	return result
}

// 全局预定义变量 registerSystemVariables 注册系统预定义变量（硬编码可信来源）
func (vr *VariableRegistry) registerSystemVariables() {
	// ✅ current_time: 当前时间
	vr.Register(VariableMeta{
		Name:        "current_time",
		Description: "当前系统时间，格式: 2006-01-02 15:04:05",
		Getter: func(ctx context.Context) (string, error) {
			return time.Now().Format("2006-01-02 15:04:05"), nil
		},
		Required: true,
	})
	// ✅ os: 操作系统类型
	vr.Register(VariableMeta{
		Name:        "os",
		Description: "运行时操作系统: linux/darwin/windows",
		Getter: func(ctx context.Context) (string, error) {
			return runtime.GOOS, nil
		},
		Required: true,
	})

	// ✅ workspace_dir: 工作目录
	vr.Register(VariableMeta{
		Name:        "workspace_dir",
		Description: "Agent 工作目录，默认 ./workspace，可通过 FEIKONG_WORKSPACE_DIR 环境变量覆盖",

		// Getter 内置配置优先级：ctx > 环境变量 > 默认值
		Getter: func(ctx context.Context) (string, error) {
			// 优先级 1: ctx 显式传入（最高优先级，用于运行时动态覆盖）
			if val := ctx.Value("workspace_dir"); val != nil {
				if s, ok := val.(string); ok && s != "" {
					return sanitizeWorkspacePath(s)
				}
			}

			// 优先级 2: 环境变量 FEIKONG_WORKSPACE_DIR
			if envVal := os.Getenv("FEIKONG_WORKSPACE_DIR"); envVal != "" {
				return sanitizeWorkspacePath(envVal)
			}

			// 优先级 3: 默认值 ./workspace
			return sanitizeWorkspacePath("./workspace")
		},

		// 校验规则
		Validator: func(value string) error {
			if value == "" {
				return fmt.Errorf("workspace_dir cannot be empty")
			}
			// 允许相对路径或绝对路径
			// 但禁止路径遍历
			if strings.Contains(value, "..") {
				return fmt.Errorf("workspace_dir cannot contain '..'")
			}
			return nil
		},

		Required: true,
		Default:  "./workspace", // 文档/降级用
	})
}

// sanitizeWorkspacePath 路径清洗和标准化
func sanitizeWorkspacePath(path string) (string, error) {
	// 1. 去除首尾空格
	path = strings.TrimSpace(path)

	// 2. 如果是相对路径，转换为绝对路径（基于当前工作目录）
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for '%s': %w", path, err)
		}
		path = absPath
	}

	// 3. 清理路径（去除多余的 / 或 \）
	path = filepath.Clean(path)

	// 4. 可选：限制在允许的根目录下（沙箱）
	// allowedRoots := []string{"/home/user", "/tmp"}
	// if !isPathUnderRoots(path, allowedRoots) {
	//     return "", fmt.Errorf("workspace_dir must be under allowed roots")
	// }

	return path, nil
}

func GetAllowVariableName(name string) ([]string, error) {
	variables := GlobalRegistry.List()
	for _, variable := range variables {
		if variable.Name == name {
			return []string{variable.Name}, nil
		}
	}
	return nil, fmt.Errorf("variable '%s' not found", name)
}
