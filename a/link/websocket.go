package link

import (
	"context"
	"encoding/json"
	"main/LLM"
	"main/asr"
	"main/tts"
	"sync"

	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

// 初始化大模型设定
type roleInit struct {
	Type   string `json:"type"`
	System string `json:"system"`
	User   string `json:"user"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// HandleWebSocket 处理前端WebSocket连接
func HandleWebSocket(asrClient *asr.ASRClient) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// 升级为WebSocket连接
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket升级失败: %v", err)
			return
		}
		defer func() {
			log.Printf("webSocket连接已关闭")
			wsConn.Close()
		}()

		audioChan := make(chan []byte, 100)
		resultChan := make(chan string, 10)
		answerChan := make(chan string, 10)
		llmChan := make(chan string, 10)
		returnChan := make(chan string, 10)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// 使用 WaitGroup 等待所有协程退
		var wg sync.WaitGroup

		// 缓存识别结果和静音判断变量
		var partialResults []string
		var lastAudioTime time.Time
		// 记录最后一次发送给 LLM 的时间, 避免手动发送后又静音发送导致多次发送
		var lastTTSTime time.Time
		// 地理信息
		var mu sync.Mutex
		const silenceTimeout = 5 * time.Second // 静音超时时间

		go func() {
			for val := range resultChan {
				// 处理未识别到信息的情况
				if val == "" {
					continue
				}
				// 缓存识别结果
				mu.Lock()
				partialResults = append(partialResults, val)
				lastAudioTime = time.Now()
				mu.Unlock()
				// 实时信息发送到前端
				returnChan <- val
			}
			close(llmChan)
			close(returnChan)
		}()

		// 定时检测静音并触发大模型处理
		wg.Add(1)
		go func() {
			defer wg.Done()
			var lastCheckTime time.Time
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					mu.Lock()
					elapsed := time.Since(lastAudioTime)
					currentTime := time.Now()
					// 暂时只做了5秒静音检测, 不可手动退出
					if elapsed > silenceTimeout &&
						currentTime.Sub(lastCheckTime) > silenceTimeout &&
						lastAudioTime.After(lastTTSTime) {
						if len(partialResults) > 0 {
							endText := partialResults[len(partialResults)-1]
							log.Printf("检测到静音，准备调用大模型: %s", endText)
							llmChan <- endText
							lastTTSTime = time.Now()
							partialResults = nil // 清空缓存
							lastCheckTime = currentTime
						}
					}
					mu.Unlock()

				case <-ctx.Done():
					return
				}
			}
		}()

		// 初始化llmCtx, 传参在最下面的协程里
		llmCtx := LLM.NewLLMContext("你是一个一个猫娘", "请在每句话结尾加上'喵~'")
		// 处理 LLM 回复
		wg.Add(1)
		answerTextChan := make(chan string, 10)
		go func() {
			defer wg.Done()
			defer close(answerTextChan)
			for question := range llmChan {
				responseChan := llmCtx.Ask(question)
				for answer := range responseChan {
					answerChan <- answer
					answerTextChan <- answer
				}
			}
		}()

		// 启动ASR流式处理
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(audioChan)
			if err := asrClient.StartWebSocketStream(ctx, audioChan, resultChan); err != nil {
				log.Printf("ASR处理失败: %v", err)
				wsConn.Close()
			}
		}()

		TTSCfg := tts.InitTTSConfig()

		//处理 TTS 请求
		returnAudioChan := make(chan []byte, 100)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for answer := range answerChan {
				//log.Printf("开始TTS转换: %s", answer)

				// 调用TTS函数生成音频数据
				audioData, err := TTSCfg.GetTTSRBytes(answer, "")
				if err != nil {
					log.Printf("TTS转换失败: %v", err)
					continue
				}
				returnAudioChan <- audioData
			}
		}()

		// 处理WebSocket消息, 重要核心
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case answer := <-answerTextChan:
					// 将大模型返回结果返回给前端
					//日志检测内容
					log.Printf("大模型返回给前端的内容: %v", answer)
					if err := wsConn.WriteJSON(map[string]string{
						"answer": answer,
					}); err != nil {
						log.Printf("发送结果失败: %v", err)
						return
					}
				case asrReturn := <-returnChan:
					// 将识别结果返回给前端
					//日志检测内容
					log.Printf("识别内容返回给前端: %v", asrReturn)
					if err := wsConn.WriteJSON(map[string]string{
						"asrReturn": asrReturn,
					}); err != nil {
						log.Printf("发送结果失败: %v", err)
						return
					}
				case audioData := <-returnAudioChan:
					// 将TTS生成的音频数据返回给前端
					log.Printf("发送TTS音频数据，长度: %d 字节", len(audioData))
					if err := wsConn.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
						log.Printf("发送音频数据失败: %v", err)
						return
					}
					// 读取前端发送的音频数据
				default:
					messageType, msg, err := wsConn.ReadMessage()
					if err != nil {
						log.Printf("读取消息失败: %v", err)
						//cancel() // 前端WebSocket关闭时，取消上下文有Bug, 因为没写重新连接
						return
					}
					if messageType == websocket.BinaryMessage {
						//写入wav文件
						//if err := asr.WritePCMToWAVFile(msg); err != nil {
						//	log.Printf("写入WAV失败: %v", err)
						//}
						//log.Printf("收到前端发送的音频数据，长度: %d 字节", len(msg))
						select {
						case audioChan <- msg: // 仅转发二进制消息
						default:
							log.Printf("audioChan 满，丢弃音频包")
						}
					} else {
						// 文本消息：尝试解析 JSON 控制指令
						var cmd cmd
						if err := json.Unmarshal(msg, &cmd); err != nil {
							log.Printf("无法解析JSON消息: %v, 原文: %s", err, string(msg))
							continue
						}

						switch cmd.Type {
						case "init":
							// 初始化或更新 LLM 上下文
							mu.Lock()
							if cmd.System == "" {
								cmd.System = "你是一个一个猫娘"
							}
							if cmd.User == "" {
								cmd.User = "请在每句话结尾加上'喵~'"
							}
							llmCtx = LLM.NewLLMContext(cmd.System, cmd.User)
							partialResults = nil
							lastAudioTime = time.Now()
							mu.Unlock()
							log.Printf("已更新LLM上下文: system=%s, user=%s", cmd.System, cmd.User)
						case "hangup":
							log.Println("收到 hangup 消息，结束会话")
							cancel()
						case "go": // 手动触发：立即使用当前缓存的识别结果调用 LLM
							log.Println("收到 go 消息，手动触发大模型调用")
							mu.Lock()
							if len(partialResults) > 0 {
								endText := partialResults[len(partialResults)-1]
								log.Printf("立即调用大模型: %s", endText)
								llmChan <- endText
								partialResults = nil // 清空缓存
								lastTTSTime = time.Now()
							} else {
								log.Println("无识别内容，跳过 LLM 调用")
							}
							mu.Unlock()
						case "up":
							TTSCfg.AdjustVolume(true)
						case "down":
							TTSCfg.AdjustVolume(false)
						case "fast":
							TTSCfg.AdjustSpeed(true)
						case "late":
							TTSCfg.AdjustSpeed(false)
						//case "updatalocation":
						//	log.Println("收到 updateLocation 消息")
						//	if cmd.Location != nil {
						//		mu.Lock()
						//		// 设置地理位置
						//		location := Location{
						//			Latitude:  cmd.Location.Latitude,
						//			Longitude: cmd.Location.Longitude,
						//			Accuracy:  cmd.Location.Accuracy,
						//		}
						//		log.Printf("地址为: %v", location)
						//		// 拼接为字符串
						//
						//		mu.Unlock()
						//	} else {
						//
						//		log.Printf("updateLocation 消息中缺少 Location 数据")
						//	}
						default:
							log.Printf("未知控制消息类型: %s", cmd.Type)
						}
					}
				}
			}
		}()
		//  等待所有协程退出
		<-ctx.Done()
		wg.Wait()
		log.Println("WebSocket处理已完成")
	}
}
