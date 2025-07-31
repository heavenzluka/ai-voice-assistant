package link

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy"` // 精度，单位米
}

type cmd struct {
	Type     string    `json:"type"`
	System   string    `json:"system"`
	User     string    `json:"user"`
	Location *Location `json:"location"`
}
