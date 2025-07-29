// 这个文件仅用于测试是否能收到音频/tts合成音频
package asr

import (
	"io"
	"net/http"
	"os"
	"sync"
)

// 全局变量用于保存当前的 WAV 文件
var (
	wavFile *os.File
	mu      sync.Mutex
)

// 保存 PCM 数据到 WAV 文件
func WritePCMToWAVFile(pcmData []byte) error {
	mu.Lock()
	defer mu.Unlock()

	// 如果文件不存在，创建新的 WAV 文件
	if wavFile == nil {
		var err error
		wavFile, err = os.Create("output.wav")
		if err != nil {
			return err
		}

		// 写入 WAV 文件头（16bit 单声道 16kHz）
		writeWAVHeader(wavFile, 16000, 1, 16)
	}

	// 写入 PCM 数据
	_, err := wavFile.Write(pcmData)
	return err
}

// 手动写入 WAV 文件头
func writeWAVHeader(w io.Writer, sampleRate int, channels int, bitsPerSample int) {
	dataSize := 0x7fffffff // 一个非常大的值，表示未知长度
	totalSize := dataSize + 36

	header := make([]byte, 44)
	// RIFF header
	copy(header[0:], []byte("RIFF"))
	header[4] = byte(totalSize)
	header[5] = byte(totalSize >> 8)
	header[6] = byte(totalSize >> 16)
	header[7] = byte(totalSize >> 24)
	copy(header[8:], []byte("WAVE"))

	// fmt subchunk
	copy(header[12:], []byte("fmt "))
	header[16] = 16 // subchunk1Size
	header[17] = 0
	header[18] = 0
	header[19] = 0
	header[20] = 1 // audioFormat = PCM
	header[21] = 0
	header[22] = byte(channels) // numChannels
	header[23] = 0
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	header[28] = byte((sampleRate * channels * bitsPerSample / 8)) // byteRate
	header[29] = byte((sampleRate * channels * bitsPerSample / 8) >> 8)
	header[30] = byte((sampleRate * channels * bitsPerSample / 8) >> 16)
	header[31] = byte((sampleRate * channels * bitsPerSample / 8) >> 24)
	header[32] = byte((channels * bitsPerSample / 8)) // blockAlign
	header[33] = 0
	header[34] = byte(bitsPerSample) // bitsPerSample
	header[35] = 0

	// data subchunk
	copy(header[36:], []byte("data"))
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	w.Write(header)
}

func ServeWAVFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "output.wav")
}
