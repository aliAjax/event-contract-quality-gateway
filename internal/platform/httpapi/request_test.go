package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestDecodePreservesBodyLimitError(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"x":"`+strings.Repeat("x", 64)+`"}`))
	var dst map[string]any
	err := Decode(r, &dst, 8)
	var limit *http.MaxBytesError
	if !errors.As(err, &limit) { t.Fatalf("Decode lost MaxBytesError: %v", err) }
}
