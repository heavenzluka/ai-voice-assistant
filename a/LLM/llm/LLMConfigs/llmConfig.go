package LLMConfigs

import (
	ark "github.com/sashabaranov/go-openai"
	"main/client"
)

const (
	DoubaoAPIKey string = client.DoubaoAPIKey
	Model        string = client.Model
	BaseURL      string = client.BaseURL
)

func Config() *ark.Client {
	config := ark.DefaultConfig(DoubaoAPIKey)
	config.BaseURL = BaseURL
	client := ark.NewClientWithConfig(config)
	return client
}
