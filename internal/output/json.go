package output

import (
	"encoding/json"
	"io"
)

type Response[T any] struct {
	Data T              `json:"data"`
	Meta map[string]any `json:"meta"`
}

func WriteJSON[T any](w io.Writer, data T, meta map[string]any) error {
	if meta == nil {
		meta = map[string]any{}
	}

	return json.NewEncoder(w).Encode(Response[T]{
		Data: data,
		Meta: meta,
	})
}
