package asr

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"main/client"
	"time"
)

// 流式连接实现实时转文字
func (c *ASRClient) StartWebSocketStream(ctx context.Context, audioStream <-chan []byte, resultChan chan<- string) error {
	voiceId := fmt.Sprintf("voice-%d", time.Now().UnixNano())
	wsURL := buildASRWebSocketURL(client.AppId, client.SecretId, client.SecretKey, voiceId)

	// 2. 建立 WebSocket 连接
	//log.Printf("连接地址: %s", wsURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("WebSocket连接失败: %v", err)
	}
	log.Printf("websocket连接成功")
	defer conn.Close()

	// 创建带 cancel 的子 context，用于控制 goroutine
	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 4 实时处理音频流
	go func() {
		defer func() {
			log.Printf("音频发送协程退出")
			cancel()
		}()
		// 用于拼装, 实现40ms发送
		var buffer []byte

		ticker := time.NewTicker(40 * time.Millisecond)
		defer ticker.Stop()
		for {

			select {
			case data := <-audioStream:
				buffer = append(buffer, data...)
			case <-ticker.C:
				if len(buffer) >= 1280 {
					// 截取前 1280 字节发送
					chunk := buffer[:1280]
					buffer = buffer[1280:] // 剩余数据保留
					//log.Printf("发送给腾讯云的数据: %v", chunk[:10])
					if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
						log.Printf("发送音频数据失败: %v", err)
						cancel()
						return
					}
				}
			case <-innerCtx.Done():
				// 结束时发送剩余数据
				if len(buffer) > 0 {
					//log.Printf("发送最后残留音频数据: %v", buffer[:min(len(buffer), 10)])
					if err := conn.WriteMessage(websocket.BinaryMessage, buffer); err != nil {
						log.Printf("发送残留数据失败: %v", err)
					}
				}
				// 发送结束消息
				if err := conn.WriteJSON(map[string]string{"type": "end"}); err != nil {
					log.Printf("发送结束消息失败: %v", err)
				}
				return
			}

		}
	}()

	// 5 接收结果 goroutine
	go func() {
		defer func() {
			log.Printf("识别结果协程退出")
			conn.Close()
			cancel()
		}()
		for {
			select {
			case <-innerCtx.Done():
				return
			default:
				_, msg, err := conn.ReadMessage()
				if err != nil {
					log.Printf("读取识别结果失败: %v", err)
					//cancel() // 前端WebSocket关闭时，取消上下文
					return
				}
				//log.Printf("收到识别结果: %s", string(msg)) // 打印原始响应
				text, err := extractVoiceText(msg)
				if err != nil {
					log.Println("错误:", err)
				} else {
					//log.Println("识别文本:", text)
					//log.Printf("将文本输入到chan")
					resultChan <- text
				}

			}
		}
	}()

	<-innerCtx.Done()
	return nil
}
