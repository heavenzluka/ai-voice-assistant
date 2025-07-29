package LLMConfigs

import (
	ark "github.com/sashabaranov/go-openai"
)

const (
	DoubaoAPIKey string = "your-llm-key"
	Model        string = "your-llm-model"
	BaseURL      string = "https://ark.cn-beijing.volces.com/api/v3"//一个例子
)

func Config() *ark.Client {
	config := ark.DefaultConfig(DoubaoAPIKey)
	config.BaseURL = BaseURL
	client := ark.NewClientWithConfig(config)
	return client
}
