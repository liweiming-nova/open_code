package config

import (
	"os"

	"github.com/liweiming-nova/open_code/chatmodel"
	"github.com/liweiming-nova/open_code/prompts"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	CustomAgent Custom `toml:"custom"`
	Plugins     Plugins `toml:"plugins"`
}

type Plugins struct {
	Knowledge KnowledgePlugin `toml:"knowledge"`
}

type KnowledgePlugin struct {
	Enabled   bool                  `toml:"enabled"`
	Dir       string                `toml:"dir"`
	StateFile string                `toml:"state_file"`
	ChunkSize int                   `toml:"chunk_size"`
	Overlap   int                   `toml:"overlap"`
	Redis     KnowledgeRedisConfig  `toml:"redis"`
	Embedding KnowledgeEmbedConfig  `toml:"embedding"`
}

type KnowledgeRedisConfig struct {
	Addr           string `toml:"addr"`
	Password       string `toml:"password"`
	DB             int    `toml:"db"`
	KeyPrefix      string `toml:"key_prefix"`
	IndexName      string `toml:"index_name"`
	DistanceMetric string `toml:"distance_metric"`
}

type KnowledgeEmbedConfig struct {
	BaseURL   string `toml:"base_url"`
	APIKey    string `toml:"api_key"`
	Model     string `toml:"model"`
	Dimension int    `toml:"dimension"`
	BatchSize int    `toml:"batch_size"`
}
type Agent struct {
	Name         string                    `toml:"name"`
	Desc         string                    `toml:"desc"`
	SystemPrompt prompts.Prompts           `toml:"system_prompt"`
	ModelName    string                    `toml:"model_name"`
	Tools        []string                  `toml:"tools,omitempty"`
	ChatModel    chatmodel.ChatModelConfig `toml:"chat_model,omitempty"`
}

type Custom struct {
	Moderator  Agent       `toml:"moderator"`
	Agents     []Agent     `toml:"agents"`
	MCPServers []MCPServer `toml:"mcp_servers"`
}

type MCPServer struct {
	Name          string   `toml:"name"`
	Desc          string   `toml:"desc"`
	Enabled       bool     `toml:"enabled"`
	Timeout       int      `toml:"timeout"`
	URL           string   `toml:"url,omitempty"`
	Command       string   `toml:"command,omitempty"`  // Command: "uvx" or "npx"
	EnvVars       []string `toml:"env_vars,omitempty"` // Environment variables for stdio
	Args          []string `toml:"args,omitempty"`     // Command arguments array
	TransportType string   `toml:"transport_type"`
}

func Unmarshal(filePath string, v any) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	err = toml.Unmarshal(data, v)
	if err != nil {
		return err
	}
	return nil
}

func Get() (*Config, error) {
	var config Config
	err := Unmarshal("config/config.toml", &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
