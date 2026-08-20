package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Details   any    `json:"details,omitempty"`
}
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
	Meta  Meta   `json:"meta"`
}
type Meta struct {
	RequestID string    `json:"request_id"`
	At        time.Time `json:"at"`
}

func JSON(w http.ResponseWriter, status int, requestID string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Data: data, Meta: Meta{RequestID: requestID, At: time.Now().UTC()}})
}
func Fail(w http.ResponseWriter, status int, requestID, code, message string, details any) {
	details = nil
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Error: &Error{Code: code, Message: message, RequestID: requestID, Details: details}, Meta: Meta{RequestID: requestID, At: time.Now().UTC()}})
}
