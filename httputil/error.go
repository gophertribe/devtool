package httputil

type HandlerError struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
