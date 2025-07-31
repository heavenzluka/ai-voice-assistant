package server

import (
	"context"
	"encoding/json"
	"fmt"
	ark "github.com/sashabaranov/go-openai"
	"log"
	"main/LLM/llm/LLMConfigs"
	"main/LLM/llm/tools"
	"strings"
)

var client = LLMConfigs.Config()

// GetLLMAnswer 返回大模型本次回复和历史记录
func GetLLMAnswer(text string, messages []ark.ChatCompletionMessage) (string, []ark.ChatCompletionMessage) {
	return ContinueConversation(text, messages)
}

func setRequest(messages []ark.ChatCompletionMessage) ark.ChatCompletionRequest {
	tool := tools.GetTools()
	request := ark.ChatCompletionRequest{
		Model:    LLMConfigs.Model,
		Messages: messages,
		Tools:    tool,
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
		case ark.ChatMessageRoleAssistant:
			log.Printf("大模型回答的信息: %v", text)
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
		// 出错也返回, 避免panic
		return resp
	}
	// 检查并记录函数调用
	if len(resp.Choices) > 0 {
		message := resp.Choices[0].Message
		// 检查是否有 tool_calls
		if len(message.ToolCalls) > 0 {
			// 记录调用了哪些函数
			var calledTools []string
			for _, toolCall := range message.ToolCalls {
				calledTools = append(calledTools, toolCall.Function.Name)
			}
			log.Printf("LLM 调用函数: [%s]", strings.Join(calledTools, ", "))
		}
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

	// 检查是否有函数调用
	if len(resp.Choices) > 0 {
		message := resp.Choices[0].Message

		// 检查是否有 tool_calls
		if len(message.ToolCalls) > 0 {
			var finalMessages []ark.ChatCompletionMessage
			var toolResponses []ark.ChatCompletionMessage
			// 开始遍历每个 tool call 并执行
			for _, toolCall := range message.ToolCalls {
				toolFunc := toolCall.Function
				if toolFunc.Arguments == "" {
					log.Printf("工具调用参数为空: %v", toolFunc.Name)
					// 即使参数为空，也要返回一个 tool response，避免 LLM 报错
					toolResponses = append(toolResponses, ark.ChatCompletionMessage{
						Role:       ark.ChatMessageRoleTool,
						Name:       toolFunc.Name,
						Content:    "错误, 工具调用参数为空",
						ToolCallID: toolCall.ID,
					})
					continue
				}
				switch toolFunc.Name {
				case "GetWeatherByCity":
					var args struct {
						City string `json:"city"`
					}
					// 解析 Arguments JSON 字符串
					if err := json.Unmarshal([]byte(toolFunc.Arguments), &args); err != nil {
						log.Printf("未能解组参数: %v", err)
						toolResponses = append(toolResponses, ark.ChatCompletionMessage{
							Role:       ark.ChatMessageRoleTool,
							Name:       toolFunc.Name,
							Content:    fmt.Sprintf("错误: 参数无效 - %v", err),
							ToolCallID: toolCall.ID,
						})
						continue
					}

					if args.City == "" {
						log.Println("缺少参数: city")
						toolResponses = append(toolResponses, ark.ChatCompletionMessage{
							Role:       ark.ChatMessageRoleTool,
							Name:       toolFunc.Name,
							Content:    "错误 缺少参数: city",
							ToolCallID: toolCall.ID,
						})
						continue
					}

					result := tools.GetWeatherByCity(args.City) // ← 你已实现的函数，返回字符串
					// 构造 tool response 消息
					toolResponse := ark.ChatCompletionMessage{
						Role:       ark.ChatMessageRoleTool,
						Name:       toolFunc.Name,
						Content:    result,
						ToolCallID: toolCall.ID,
					}
					toolResponses = append(toolResponses, toolResponse)
				case "GetWeatherByCoordinates":
					var args struct {
						Lat string `json:"lat"`
						Lon string `json:"lon"`
					}
					// 解析 Arguments JSON 字符串
					if err := json.Unmarshal([]byte(toolFunc.Arguments), &args); err != nil {
						log.Printf("未能解组参数: %v", err)
						toolResponses = append(toolResponses, ark.ChatCompletionMessage{
							Role:       ark.ChatMessageRoleTool,
							Name:       toolFunc.Name,
							Content:    fmt.Sprintf("错误: 参数无效 - %v", err),
							ToolCallID: toolCall.ID,
						})
						continue
					}

					if args.Lat == "" || args.Lon == "" {
						log.Println("缺少参数: lat, lon")
						toolResponses = append(toolResponses, ark.ChatCompletionMessage{
							Role:       ark.ChatMessageRoleTool,
							Name:       toolFunc.Name,
							Content:    "错误 缺少参数: lat, lon",
							ToolCallID: toolCall.ID,
						})
						continue
					}

					result := tools.GetWeatherByCoordinates(args.Lat, args.Lon) // ← 你已实现的函数，返回字符串
					// 构造 tool response 消息
					toolResponse := ark.ChatCompletionMessage{
						Role:       ark.ChatMessageRoleTool,
						Name:       toolFunc.Name,
						Content:    result,
						ToolCallID: toolCall.ID,
					}
					toolResponses = append(toolResponses, toolResponse)
				}
			}

			// 把原始 assistant 消息 + 所有 tool responses 加入对话
			assistantMsg := message
			finalMessages = append(finalMessages, messages...)
			finalMessages = append(finalMessages, assistantMsg) // 包含 function_call 的 assistant 消息
			finalMessages = append(finalMessages, toolResponses...)

			// 第二次调用 LLM，让它基于 tool 结果生成自然语言回复
			resp2 := getResponse(finalMessages)
			if len(resp2.Choices) > 0 && resp2.Choices[0].Message.Content != "" {
				finalAnswer := resp2.Choices[0].Message.Content
				log.Println("bot answer: ", finalAnswer)

				// 将最终 assistant 回复也加入对话历史
				finalMessages = append(finalMessages, resp2.Choices[0].Message)

				return finalAnswer, finalMessages
			}
		}
	}

	// 如果没有调用直接返回llm回答
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
