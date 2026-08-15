package transport

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

type Request struct {
	Inventory map[string]int `json:"inventory"`
	Order     domain.Order   `json:"order"`
}

func DecodeRequest(reader io.Reader) (Request, error) {
	var request Request
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("decode request: %w", err)
	}
	if len(request.Inventory) == 0 {
		return Request{}, fmt.Errorf("inventory is required")
	}
	return request, nil
}
