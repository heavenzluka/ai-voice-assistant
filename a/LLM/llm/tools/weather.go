// tools/weather.go
package tools

import (
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
)

type WeatherResponse struct {
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Sys struct {
		Country string `json:"country"`
	} `json:"sys"`
	Name string `json:"name"`
	Cod  int    `json:"cod"`
}

func GetWeatherByCoordinates(lat, lon string) string {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%s&lon=%s&appid=%s&units=metric", lat, lon, WeatherAPIKey)
	resp, err := getWeatherByUrl(url)
	if err != nil {
		log.Println(err)
		return ""
	}
	return getWeatherText(resp)
}

func GetWeatherByCity(city string) string {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s,cn&APPID=%s&units=metric", city, WeatherAPIKey)
	resp, err := getWeatherByUrl(url)
	if err != nil {
		log.Println(err)
		return ""
	}
	return getWeatherText(resp)
}

func getWeatherByUrl(url string) (*WeatherResponse, error) {
	if url == "" {
		return nil, fmt.Errorf("url is empty")
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	var weatherResp WeatherResponse
	err = json.Unmarshal(body, &weatherResp)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %v", err)
	}

	if weatherResp.Cod != 200 {
		return nil, fmt.Errorf("API 返回错误: %d, %v", weatherResp.Cod, weatherResp)
	}

	return &weatherResp, nil
}

// getWeatherText 解码为字符串参数
func getWeatherText(weather *WeatherResponse) string {
	if weather.Cod != 200 {
		return "天气api调用失败"
	}
	var answer string
	if weather.Name != "" {
		answer = fmt.Sprintf("为您播报%s的天气情况：", weather.Name)
	} else {
		answer = "为您播报的天气情况: "
	}

	if len(weather.Weather) > 0 {
		answer += fmt.Sprintf("今天天气%s，", weather.Weather[0].Description)
	}
	answer += fmt.Sprintf("当前温度%.1f摄氏度, 体感: %.1f摄氏度\n", weather.Main.Temp, weather.Main.FeelsLike)
	//answer += fmt.Sprintf("最高%.1f摄氏度, 最低:  %.1f摄氏度\n", weather.Main.TempMax, weather.Main.TempMin)
	//answer += fmt.Sprintf("空气湿度 %d%%\n", weather.Main.Humidity)
	//answer += fmt.Sprintf("大气气压 %d 百帕\n", weather.Main.Pressure)
	//answer += fmt.Sprintf("云量 %d%%\n", weather.Clouds.All)
	//answer += "以上就是今天的天气情况"
	return answer
}

// getWeatherByCityFunction 注册城市天气查询函数到 Tool 调用系统
func getWeatherByCityFunction() openai.Tool {
	return openai.Tool{
		Type: "function",
		Function: &openai.FunctionDefinition{
			Name:        "GetWeatherByCity",
			Description: "通过城市名称查询当前天气",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type":        "string",
						"description": "城市名称，如：北京市",
					},
				},
				"required": []string{"city"},
			},
		},
	}
}

// getWeatherByCoordinatesFunction 注册经纬度天气查询函数到 Tool 调用系统
func getWeatherByCoordinatesFunction() openai.Tool {
	return openai.Tool{
		Type: "function",
		Function: &openai.FunctionDefinition{
			Name:        "GetWeatherByCoordinates",
			Description: "通过经纬度查询当前天气",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"lat": map[string]interface{}{
						"type":        "number",
						"description": "纬度，例如：39.9042",
					},
					"lon": map[string]interface{}{
						"type":        "number",
						"description": "经度，例如：116.4074",
					},
				},
				"required": []string{"lat", "lon"},
			},
		},
	}
}
