package agents

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/liweiming-nova/open_code/agents/analyst"
	"github.com/liweiming-nova/open_code/agents/custom"
	"github.com/liweiming-nova/open_code/config"
	"github.com/pterm/pterm"
)

// AgentInfo 智能体信息
type AgentInfo struct {
	Name        string
	Description string
	Creator     func(ctx context.Context) adk.Agent
}

var (
	// Registry 智能体注册表
	Registry     []AgentInfo
	registryOnce sync.Once
)

// initRegistry 初始化注册表（延迟加载）
func initRegistry() {
	registryOnce.Do(func() {
		ctx := context.Background()

		// 定义所有 agent 的创建函数
		var creators []func(context.Context) adk.Agent

		// 基础智能体（始终可用）
		creators = append(creators)

		// 可选智能体（根据环境变量启用）
		if os.Getenv("OPEN_CODE_ANALYST_ENABLED") == "true" {
			creators = append(creators, analyst.NewAgent)
		}

		// 动态构建注册表
		Registry = make([]AgentInfo, 0, len(creators))
		for _, creator := range creators {
			agent := creator(ctx)
			Registry = append(Registry, AgentInfo{
				Name:        agent.Name(ctx),
				Description: agent.Description(ctx),
				Creator: func(cachedAgent adk.Agent) func(ctx context.Context) adk.Agent {
					return func(ctx context.Context) adk.Agent {
						return cachedAgent
					}
				}(agent),
			})
		}

		// 加载配置文件中的自定义智能体
		loadCustomAgents(ctx)
	})
}

// loadCustomAgents 从配置文件加载自定义智能体并添加到注册表
func loadCustomAgents(ctx context.Context) {
	cfg, err := config.Get()
	if err != nil {
		return // 配置文件不存在或解析失败，跳过
	}

	if len(cfg.CustomAgent.Agents) == 0 {
		return
	}

	// 构建已有名称集合用于冲突检测
	existingNames := make(map[string]bool, len(Registry))
	for _, info := range Registry {
		existingNames[info.Name] = true
	}

	for _, agentCfg := range cfg.CustomAgent.Agents {
		if agentCfg.Name == "" {
			continue
		}

		if existingNames[agentCfg.Name] {
			pterm.Warning.Printfln("自定义智能体 \"%s\" 与已有智能体名称重复，不建议使用相同名称", agentCfg.Name)
		}

			agent := custom.NewAgent(ctx, agentCfg)

		Registry = append(Registry, AgentInfo{
			Name:        agentCfg.Name,
			Description: agentCfg.Desc,
			Creator: func(cachedAgent adk.Agent) func(ctx context.Context) adk.Agent {
				return func(ctx context.Context) adk.Agent {
					return cachedAgent
				}
			}(agent),
		})

		fmt.Printf("[tips] 加载自定义智能体: %s (%s)\n", agentCfg.Name, agentCfg.Desc)
	}
}

// GetRegistry 获取智能体注册表
func GetRegistry() []AgentInfo {
	initRegistry()
	return Registry
}

// GetAgentByName 根据名字获取智能体
func GetAgentByName(name string) *AgentInfo {
	initRegistry()
	for i := range Registry {
		if Registry[i].Name == name {
			return &Registry[i]
		}
	}
	return nil
}
