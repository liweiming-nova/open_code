package analyst

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cloudwego/eino/schema"
	"github.com/liweiming-nova/open_code/prompts"
)

// 固定路径 可以通过save_prompt 去更新analyst_lastest 文件
const path = "prompts/analyst/analyst_lastest.json"

func loadPrompt(ctx context.Context) ([]*schema.Message, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var myPrompt prompts.Prompts
	err = json.Unmarshal(data, &myPrompt)
	if err != nil {
		return nil, err
	}

	formattedPrompt, err := myPrompt.Format(ctx)
	if err != nil {
		return nil, err
	}

	return []*schema.Message{schema.SystemMessage(formattedPrompt)}, nil
}
