package tts

import (
	"encoding/base64"
	"fmt"
	"log"
	"main/client"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tts/v20190823"
)

func GetTTSRBytes(text string, speakerType string, opts ...float64) ([]byte, error) {
	ttsClient, err := NewTTSClient(client.SecretId, client.SecretKey)
	if err != nil {
		log.Printf("tts初始化错误: %v", err)
	}
	speed := 0.0
	volume := 5.0
	if len(opts) >= 1 {
		speed = opts[0]
	}
	if len(opts) >= 2 {
		volume = opts[1]
	}
	speaker := ttsSpeaker(speakerType)
	request := tts.NewTextToVoiceRequest()
	request = setRequest(request, text, speed, volume, speaker)
	response, err := ttsClient.client.TextToVoice(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		log.Printf("ttsApi错误: %s", err)
	}
	if err != nil {
		log.Printf("获取ttsResponse错误: %s", err)
	}

	return ttsClient.GetBytes(response)
}

func (t *TTSClient) GetBytes(response *tts.TextToVoiceResponse) ([]byte, error) {
	audioStr := *response.Response.Audio
	audioBytes, err := base64.StdEncoding.DecodeString(audioStr)
	if err != nil {
		return nil, fmt.Errorf("Base64解码失败: %v", err)
	}
	return audioBytes, nil
}
