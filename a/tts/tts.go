package tts

import (
	"encoding/base64"
	"fmt"
	"log"
	"main/client"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tts/v20190823"
)

func (config *TTSConfig) GetTTSRBytes(text string, speakerType string) ([]byte, error) {
	ttsClient, err := NewTTSClient(client.SecretId, client.SecretKey)
	if err != nil {
		log.Printf("tts初始化错误: %v", err)
	}

	speed := config.Speed
	volume := config.Volume
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

func (config *TTSConfig) AdjustVolume(add bool) {
	config.StateMutex.Lock()
	defer config.StateMutex.Unlock()
	if add {
		config.Volume++
	} else {
		config.Volume--
	}

	// 限制范围
	if config.Volume < -10.0 {
		config.Volume = -10.0
	} else if config.Volume > 10.0 {
		config.Volume = 10.0
	}
	log.Printf("音量调整为: %.1f", config.Volume)
}

func (config *TTSConfig) AdjustSpeed(add bool) {
	config.StateMutex.Lock()
	defer config.StateMutex.Unlock()
	if add {
		config.Speed++
	} else {
		config.Speed--
	}
	// 限制范围
	if config.Speed < -2.0 {
		config.Speed = -2.0
	} else if config.Speed > 6.0 {
		config.Speed = 6.0
	}
	log.Printf("语速调整为: %.1f", config.Speed)
}
