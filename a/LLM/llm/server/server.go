package server

import (
	"context"
	ark "github.com/sashabaranov/go-openai"
	"log"
	"main/LLM/llm/LLMConfigs"
)

var client = LLMConfigs.Config()

// GetLLMAnswer 返回大模型本次回复和历史记录
func GetLLMAnswer(text string, messages []ark.ChatCompletionMessage) (string, []ark.ChatCompletionMessage) {
	return ContinueConversation(text, messages)
}

func setRequest(messages []ark.ChatCompletionMessage) ark.ChatCompletionRequest {
	request := ark.ChatCompletionRequest{
		Model:    LLMConfigs.Model,
		Messages: messages,
	}
	return request
}

// getResponse 接受一个message,{Role, Content}
// 返回的信息在resp.Choices[0].Message.Content
func getResponse(messages []ark.ChatCompletionMessage) ark.ChatCompletionResponse {
	// 日志检查是否传入有效content
	for _, text := range messages {
		switch text.Role {
		case ark.ChatMessageRoleSystem:
			log.Printf("llm记录信息: %v", text)
		case ark.ChatMessageRoleUser:
			log.Printf("用户传入信息记录: %v", text)
		default:
			log.Printf("其他类型信息: %v", text)
		}
	}

	request := setRequest(messages)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		request,
	)

	if err != nil {
		log.Printf("ChatCompletion error: %v\n", err)
	}
	return resp
}

func AddUserMessage(text string, messages []ark.ChatCompletionMessage) []ark.ChatCompletionMessage {
	return append(messages, ark.ChatCompletionMessage{
		Role:    ark.ChatMessageRoleUser,
		Content: text,
	})
}

func AddAssistantMessage(answer string, messages []ark.ChatCompletionMessage) []ark.ChatCompletionMessage {
	return append(messages, ark.ChatCompletionMessage{
		Role:    ark.ChatMessageRoleAssistant,
		Content: answer,
	})
}

func ContinueConversation(text string, messages []ark.ChatCompletionMessage) (string, []ark.ChatCompletionMessage) {
	messages = AddUserMessage(text, messages)
	resp := getResponse(messages)
	answer := resp.Choices[0].Message.Content
	log.Println("bot answer: ", answer)
	messages = AddAssistantMessage(answer, messages)
	return answer, messages
}

func InitMessage(args ...string) []ark.ChatCompletionMessage {
	var system, user string
	if len(args) >= 1 {
		system = args[0]
	}
	if system == "" {
		system = "你是一个人工智能小助手"
	}
	if len(args) >= 2 {
		user = args[1]
	}
	content := user
	if content == "" {
		content = system
	}

	return []ark.ChatCompletionMessage{
		{
			Role:    ark.ChatMessageRoleSystem,
			Content: system,
		},
		{
			Role:    ark.ChatMessageRoleUser,
			Content: content,
		},
	}
}
