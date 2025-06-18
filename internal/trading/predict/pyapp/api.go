package pyapp

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nikita55612/httpx"
)

func pingApp() string {
	path := fmt.Sprintf("http://%s/ping", appAddr)
	res, err := httpx.Get(path).Build().Do()
	if err != nil {
		return ""
	}
	defer res.Close()
	body, err := res.ReadBody()
	if err != nil {
		return ""
	}
	return string(body)
}

type Request struct {
	Features [][]float64 `json:"features"` // Массив признаков для предсказания
	Model    string      `json:"model"`    // Название модели для предсказания
}

type Response struct {
	Predict map[string][]float64 `json:"predict"` // Результаты предсказаний
	Error   string               `json:"error"`   // Описание ошибки, если возникла
}

func (r *Response) Unwrap() ([]float64, error) {
	if r.Error != "" {
		return nil, errors.New(r.Error)
	}
	for _, v := range r.Predict {
		return v, nil
	}
	return nil, fmt.Errorf("empty response")
}

func GetPrediction(features [][]float64, model string) *Response {
	request := &Request{
		Features: features,
		Model:    model,
	}
	requestData, _ := json.Marshal(request)

	mu.Lock()
	fullURL := fmt.Sprintf("http://%s/predict", appAddr)
	mu.Unlock()

	res, err := httpx.Post(fullURL).
		WithData(requestData).
		WithHeader("Content-Type", "application/json").
		Build().
		Do()
	if err != nil {
		return &Response{
			Error: fmt.Sprintf("request failed: %v", err),
		}
	}
	defer res.Close()

	var response Response
	if err := res.UnmarshalBody(&response); err != nil {
		return &Response{
			Error: fmt.Sprintf("could not parse the server response: %v", err),
		}
	}
	return &response
}
