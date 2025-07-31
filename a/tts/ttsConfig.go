package tts

import "sync"

type TTSConfig struct {
	Volume     float64
	Speed      float64
	StateMutex sync.Mutex
}

func InitTTSConfig() *TTSConfig {
	config := TTSConfig{}
	config.Volume = 5.0
	config.Speed = 0.0
	return &config
}
