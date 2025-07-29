package tts

import (
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tts/v20190823"
	"math/rand"
	"strconv"
	"time"
)

type TTSClient struct {
	client *tts.Client
}

func NewTTSClient(secretId, secretKey string) (*TTSClient, error) {
	credential := common.NewCredential(secretId, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tts.tencentcloudapi.com"

	client, err := tts.NewClient(credential, "", cpf)
	if err != nil {
		return nil, fmt.Errorf("创建TTS客户端失败: %v", err)
	}
	return &TTSClient{client: client}, nil
}

func ttsSpeaker(speakerType string) int64 {
	var speaker int64
	switch speakerType {
	case "标准女声":
		speaker = 1001
	case "标准男声":
		speaker = 1002
	default:
		speaker = 1003 //温柔女声
	}
	return speaker
}

// 随机数+时间戳sessionId
func GenerateSessionID() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + strconv.Itoa(rand.Intn(10000))
}

func setRequest(request *tts.TextToVoiceRequest, text string, speed, volume float64, speaker int64) *tts.TextToVoiceRequest {
	request.Text = common.StringPtr(text) // 最大150中文
	//一次请求对应一个SessionId，会原样返回，建议传入类似于uuid的字符串防止重复
	request.SessionId = common.StringPtr(GenerateSessionID())
	request.ModelType = common.Int64Ptr(1)       // 深度学习模型
	request.Speed = common.Float64Ptr(speed)     // 语速 [-2,6] 默认0
	request.Volume = common.Float64Ptr(volume)   // 音量 [-10,10] 默认5
	request.VoiceType = common.Int64Ptr(speaker) //音色 ID，包括标准音色、精品音色、大模型音色与基础版复刻音色
	request.Codec = common.StringPtr("wav")      //  返回音频格式，可取值：wav（默认），mp3，pcm
	//EmotionCategory *string 情感，仅支持多情感音色使用。取值:neutral(中性)、sad(悲伤)、happy(高兴)、angry(生气)、fear(恐惧)、news(新闻)、story(故事)、radio(广播)、poetry(诗歌)、call(客服)、sajiao(撒娇)、disgusted(厌恶)、amaze(震惊)、peaceful(平静)、exciting(兴奋)、aojiao(傲娇)、jieshuo(解说)
	//EmotionIntensity *int64 控制合成音频情感程度，取值范围为[50,200],默认为100
	return request
}
