package asr

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"time"
)

// 构建带签名的 ASR WebSocket URL
func buildASRWebSocketURL(appid, secretID, secretKey, voiceID string) string {
	host := "asr.cloud.tencent.com"

	params := url.Values{}
	params.Add("secretid", secretID)
	params.Add("timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	params.Add("expired", fmt.Sprintf("%d", time.Now().Add(24*time.Hour).Unix()))
	params.Add("nonce", fmt.Sprintf("%d", rand.Int63n(10000000000)))
	params.Add("engine_model_type", "16k_zh")
	params.Add("voice_id", voiceID)
	params.Add("voice_format", "1") // PCM
	params.Add("source_type", "1")  // 实时流式识别

	// 生成签名
	signature := generateSignature(secretKey, appid, host, params)

	// 拼接最终 URL（避免双重编码）
	wsURL := fmt.Sprintf(
		"wss://%s/asr/v2/%s?%s&signature=%s",
		host, appid, params.Encode(), signature,
	)

	return wsURL
}

// 生成签名
func generateSignature(secretKey, appid, host string, params url.Values) string {
	// 1. 构建签名原文：host + path + query params（不包含 protocol）
	path := fmt.Sprintf("/asr/v2/%s", appid)
	fullURL := fmt.Sprintf("%s%s?%s", host, path, params.Encode())

	// 2. 使用 HMAC-SHA1 签名
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(fullURL))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 3. 对签名结果进行一次 URL 编码（关键：只编码一次！）
	encodedSignature := url.QueryEscape(signature)

	// 输出日志用于调试
	//fmt.Printf("签名原文: %s\n", fullURL)
	//fmt.Printf("签名结果: %s\n", encodedSignature)

	return encodedSignature
}

// 处理asr返回的json数据
func extractVoiceText(jsonData []byte) (string, error) {
	var resp ASRResponse
	if err := json.Unmarshal(jsonData, &resp); err != nil {
		return "", fmt.Errorf("解析JSON失败: %v", err)
	}
	return resp.Result.VoiceTextStr, nil
}

// 测试用,接收识别结果并打印
func PrintASRResults(resultChan <-chan string, ctx context.Context) {

	for {
		select {
		case text, ok := <-resultChan:
			if !ok {
				log.Println("【识别结果通道关闭】")
				return
			}
			log.Printf("【实时识别结果】: %s", text)
		case <-ctx.Done():
			log.Println("【上下文取消，停止打印识别结果】")
			return
		}
	}
}
