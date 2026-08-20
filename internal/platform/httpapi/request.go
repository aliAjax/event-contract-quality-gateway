package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func RequestID(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Request-ID")); id != "" {
		return id
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
func Decode(r *http.Request, dst any, limit int64) error {
	r.Body = http.MaxBytesReader(nil, r.Body, limit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain one JSON document")
	}
	return nil
}

func decodeError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("request decode failed: %w", err)
}
func Tenant(r *http.Request) (string, error) {
	tenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenant == "" {
		return "", fmt.Errorf("X-Tenant-ID is required")
	}
	if len(tenant) > 120 {
		return "", fmt.Errorf("tenant id is too long")
	}
	return tenant, nil
}
