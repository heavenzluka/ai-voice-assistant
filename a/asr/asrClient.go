package asr

import (
	asr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/asr/v20190614"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"main/client"
)

type ASRClient struct {
	client *asr.Client
}
type ASRResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	MessageID string `json:"message_id"`
	Result    struct {
		SliceType    int         `json:"slice_type"`
		Index        int         `json:"index"`
		VoiceTextStr string      `json:"voice_text_str"`
		EmotionType  interface{} `json:"emotion_type"`
		SpeakerInfo  interface{} `json:"speaker_info"`
	} `json:"result"`
}

type ASRStreamer struct {
	client    *ASRClient
	taskID    uint64
	audioChan chan []byte
}

func NewASRClient() (*ASRClient, error) {
	credential := common.NewCredential(
		client.SecretId,
		client.SecretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "asr.tencentcloudapi.com" // 设置接口地址
	client, err := asr.NewClient(credential, "", cpf)
	if err != nil {
		return nil, err
	}

	return &ASRClient{client: client}, nil
}
