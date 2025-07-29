package LLM

import (
	ark "github.com/sashabaranov/go-openai"
	"main/LLM/llm/server"
	"sync"
)

type LLMContext struct {
	messages []ark.ChatCompletionMessage
	input    chan string
	output   chan string
	wg       sync.WaitGroup
}

func NewLLMContext(system, user string) *LLMContext {
	ctx := &LLMContext{
		input:  make(chan string),
		output: make(chan string),
	}
	//log.Printf("system: %s, user: %s", system, user)
	ctx.messages = server.InitMessage(system, user)
	//log.Printf("messages: %s", ctx.messages)

	// 启动后台处理协程
	ctx.wg.Add(1)
	go func() {
		defer ctx.wg.Done()
		for userText := range ctx.input {
			if userText == "" {
				continue
			}
			answer, updatedMessages := server.GetLLMAnswer(userText, ctx.messages)
			ctx.messages = updatedMessages
			ctx.output <- answer
		}
		close(ctx.output)
	}()

	return ctx
}

func (c *LLMContext) Ask(text string) <-chan string {
	result := make(chan string, 1)
	c.input <- text
	go func() {
		answer := <-c.output // 等待结果
		result <- answer
		close(result)
	}()
	return result
}
