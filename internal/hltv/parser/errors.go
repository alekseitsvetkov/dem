package parser

const (
	ErrorCodeParse           = "parse_error"
	ErrorCodeUnavailableData = "unavailable_data"
)

// ParseError represents a structured error from HLTV page parsing.
type ParseError struct {
	Code    string
	Area    string
	Field   string
	Message string
	Err     error
}

func (e *ParseError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *ParseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Details returns a structured map of error context fields.
// It never includes raw HTML or response body content.
func (e *ParseError) Details() map[string]any {
	details := map[string]any{}
	if e == nil {
		return details
	}
	if e.Area != "" {
		details["area"] = e.Area
	}
	if e.Field != "" {
		details["field"] = e.Field
	}
	return details
}
