package tools

import (
	"github.com/sashabaranov/go-openai"
	"main/client"
)

var Tools = []openai.Tool{}
var WeatherAPIKey = client.WeatherAPIKey

func GetTools() []openai.Tool {
	Tools = append(Tools, getWeatherByCoordinatesFunction())
	Tools = append(Tools, getWeatherByCityFunction())
	return Tools
}
