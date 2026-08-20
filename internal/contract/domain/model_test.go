package domain

import (
	"testing"
	"time"
)

func TestCompatibilityRejectsRequiredRemoval(t *testing.T) {
	old := Schema{Kind: JSONSchema, Fields: []Field{{Name: "id", Type: "string", Required: true}}}
	newSchema := Schema{Kind: JSONSchema, Fields: []Field{{Name: "name", Type: "string"}}}
	report := Compare(old, newSchema, Backward)
	if report.Compatible {
		t.Fatal("expected breaking removal")
	}
}

func TestPublishNeedsETag(t *testing.T) {
	c, err := NewContract("c", "t", "subject", "", Backward, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.AddVersion(Schema{Kind: JSONSchema, Fields: []Field{{Name: "id", Type: "string", Required: true}}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Publish(v.Number, "", time.Now()); err == nil {
		t.Fatal("expected missing etag rejection")
	}
}
