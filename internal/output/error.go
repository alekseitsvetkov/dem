package output

import (
	"encoding/json"
	"io"
)

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func WriteErrorJSON(w io.Writer, code string, message string, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}

	return json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
