package api_model

// ErrorDetail describes a single validation or domain error detail.
type ErrorDetail struct {
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason"`
}

// ErrorObject contains machine-readable error metadata.
type ErrorObject struct {
	Code      string        `json:"code"`
	Details   []ErrorDetail `json:"details,omitempty"`
	RequestID string        `json:"request_id,omitempty"`
}

// APIResponse is a standard envelope for all API responses.
type APIResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    interface{}  `json:"data,omitempty"`
	Error   *ErrorObject `json:"error,omitempty"`
}
