package http

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Response struct {
	Status      string       `json:"status"`
	Description string       `json:"description"`
	Data        interface{}  `json:"data"`
	Errors      []FieldError `json:"errors,omitempty"`
}
