package hltv

const (
	ErrorCodeNetwork = "network_error"
	ErrorCodeHTTP    = "http_error"
)

type ProviderError struct {
	Code       string
	Message    string
	URL        string
	StatusCode int
	Err        error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Message
	if msg == "" && e.Err != nil {
		msg = e.Err.Error()
	}
	if msg == "" {
		msg = e.Code
	}
	if e.Err != nil && e.Err.Error() != msg {
		msg = msg + ": " + e.Err.Error()
	}
	if e.URL != "" {
		msg = msg + " [" + e.URL + "]"
	}
	return msg
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *ProviderError) Details() map[string]any {
	details := map[string]any{}
	if e == nil {
		return details
	}
	if e.URL != "" {
		details["url"] = e.URL
	}
	if e.StatusCode != 0 {
		details["status_code"] = e.StatusCode
	}
	return details
}
