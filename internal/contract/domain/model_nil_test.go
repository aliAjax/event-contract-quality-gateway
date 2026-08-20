package domain

import "testing"

func TestSchemaValidateReportsDuplicateWithoutPanic(t *testing.T) {
	defer func() { if recovered := recover(); recovered != nil { t.Fatalf("schema validation panicked: %v", recovered) } }()
	err := (Schema{Kind: JSONSchema, Fields: []Field{{Name: "id", Type: "string"}, {Name: "id", Type: "string"}}}).Validate()
	if err == nil { t.Fatal("duplicate field was accepted") }
}
