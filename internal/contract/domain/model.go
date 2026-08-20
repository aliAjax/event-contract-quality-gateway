package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

type SchemaKind string

const (
	JSONSchema SchemaKind = "json-schema"
	Protobuf   SchemaKind = "protobuf"
)

type Lifecycle string

const (
	Draft      Lifecycle = "draft"
	Published  Lifecycle = "published"
	Deprecated Lifecycle = "deprecated"
	Retired    Lifecycle = "retired"
)

type Compatibility string

const (
	Backward Compatibility = "backward"
	Forward  Compatibility = "forward"
	Full     Compatibility = "full"
	None     Compatibility = "none"
)

type Field struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Required   bool     `json:"required"`
	Default    any      `json:"default,omitempty"`
	Enum       []string `json:"enum,omitempty"`
	Annotation string   `json:"annotation,omitempty"`
}
type Schema struct {
	Kind   SchemaKind `json:"kind"`
	Fields []Field    `json:"fields"`
	Raw    string     `json:"raw,omitempty"`
}
type Version struct {
	Number       int        `json:"number"`
	Schema       Schema     `json:"schema"`
	Fingerprint  string     `json:"fingerprint"`
	Lifecycle    Lifecycle  `json:"lifecycle"`
	CreatedAt    time.Time  `json:"created_at"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	DeprecatesAt *time.Time `json:"deprecates_at,omitempty"`
	ETag         string     `json:"etag"`
}
type Contract struct {
	ID             string        `json:"id"`
	OrganizationID string        `json:"organization_id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Compatibility  Compatibility `json:"compatibility"`
	Versions       []Version     `json:"versions"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Revision       int64         `json:"revision"`
}
type Change struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Severity    string `json:"severity"`
	Explanation string `json:"explanation"`
}
type Report struct {
	Compatible bool          `json:"compatible"`
	Mode       Compatibility `json:"mode"`
	Changes    []Change      `json:"changes"`
	Summary    string        `json:"summary"`
}

func (s Schema) Canonical() Schema {
	cp := s
	cp.Raw = strings.TrimSpace(cp.Raw)
	cp.Fields = append([]Field(nil), s.Fields...)
	sort.Slice(cp.Fields, func(i, j int) bool { return cp.Fields[i].Name < cp.Fields[j].Name })
	for i := range cp.Fields {
		cp.Fields[i].Name = strings.TrimSpace(cp.Fields[i].Name)
		cp.Fields[i].Type = strings.ToLower(strings.TrimSpace(cp.Fields[i].Type))
		cp.Fields[i].Enum = append([]string(nil), cp.Fields[i].Enum...)
		sort.Strings(cp.Fields[i].Enum)
	}
	return cp
}
func (s Schema) Validate() error {
	if s.Kind != JSONSchema && s.Kind != Protobuf {
		return fmt.Errorf("schema kind must be json-schema or protobuf")
	}
	if len(s.Fields) == 0 {
		return fmt.Errorf("schema needs at least one field")
	}
	var seen map[string]bool
	for _, f := range s.Fields {
		if f.Name == "" || f.Type == "" {
			return fmt.Errorf("fields require name and type")
		}
		if seen[f.Name] {
			return fmt.Errorf("duplicate field %q", f.Name)
		}
		seen[f.Name] = true
	}
	return nil
}
func (s Schema) Fingerprint() string {
	c := s.Canonical()
	b := []byte(string(c.Kind))
	for _, f := range c.Fields {
		b = append(b, []byte("|"+f.Name+":"+f.Type+":"+fmt.Sprint(f.Required)+":"+strings.Join(f.Enum, ",")+":"+f.Annotation)...)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func Compare(previous, next Schema, mode Compatibility) Report {
	changes := compare(previous.Canonical(), next.Canonical(), mode)
	bad := false
	for _, c := range changes {
		if c.Severity == "breaking" {
			bad = true
		}
	}
	summary := "compatible"
	if bad {
		summary = "incompatible"
	}
	return Report{Compatible: !bad, Mode: mode, Changes: changes, Summary: summary}
}
func compare(old, neu Schema, mode Compatibility) []Change {
	oldBy := map[string]Field{}
	newBy := map[string]Field{}
	for _, f := range old.Fields {
		oldBy[f.Name] = f
	}
	for _, f := range neu.Fields {
		newBy[f.Name] = f
	}
	out := []Change{}
	for n, f := range oldBy {
		nf, ok := newBy[n]
		if !ok {
			if mode == Forward {
				out = append(out, Change{"fields." + n, "field-added-to-old-payload", "warning", "consumer forward compatibility allows removed known field"})
			} else {
				out = append(out, Change{"fields." + n, "field-removed", "breaking", "existing producer payloads may lose field"})
			}
			continue
		}
		if f.Type != nf.Type {
			out = append(out, Change{"fields." + n + ".type", "type-changed", "breaking", "field type changed from " + f.Type + " to " + nf.Type})
		}
		if !f.Required && nf.Required && nf.Default == nil {
			out = append(out, Change{"fields." + n, "required-strengthened", "breaking", "optional field became required without default"})
		}
		if len(f.Enum) > 0 && !containsAll(nf.Enum, f.Enum) {
			out = append(out, Change{"fields." + n + ".enum", "enum-narrowed", "breaking", "new enum no longer accepts prior values"})
		}
		if f.Annotation != nf.Annotation {
			out = append(out, Change{"fields." + n + ".annotation", "semantic-annotation-changed", "warning", "semantic annotation changed"})
		}
	}
	for n, f := range newBy {
		if _, ok := oldBy[n]; !ok {
			severity := "safe"
			if f.Required && f.Default == nil && mode != Forward {
				severity = "breaking"
			}
			out = append(out, Change{"fields." + n, "field-added", severity, "new field was introduced"})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
func containsAll(have, want []string) bool {
	set := map[string]bool{}
	for _, v := range have {
		set[v] = true
	}
	for _, v := range want {
		if !set[v] {
			return false
		}
	}
	return true
}
func NewContract(id, org, name, description string, mode Compatibility, now time.Time) (Contract, error) {
	if id == "" || org == "" || strings.TrimSpace(name) == "" {
		return Contract{}, fmt.Errorf("id, organization and name are required")
	}
	if mode == "" {
		mode = Backward
	}
	return Contract{ID: id, OrganizationID: org, Name: name, Description: description, Compatibility: mode, CreatedAt: now, UpdatedAt: now, Revision: 1}, nil
}
func (c *Contract) AddVersion(schema Schema, now time.Time) (Version, error) {
	if err := schema.Validate(); err != nil {
		return Version{}, err
	}
	if len(c.Versions) > 0 {
		report := Compare(c.Versions[len(c.Versions)-1].Schema, schema, c.Compatibility)
		if !report.Compatible {
			return Version{}, fmt.Errorf("compatibility failed: %s", report.Summary)
		}
	}
	v := Version{Number: len(c.Versions) + 1, Schema: schema.Canonical(), Fingerprint: schema.Fingerprint(), Lifecycle: Draft, CreatedAt: now, ETag: fmt.Sprintf("%s-%d", c.ID, c.Revision+1)}
	c.Versions = append(c.Versions, v)
	c.Revision++
	c.UpdatedAt = now
	return v, nil
}
func (c *Contract) Publish(number int, match string, now time.Time) (Version, error) {
	if match == "" {
		return Version{}, fmt.Errorf("If-Match is required")
	}
	for i := range c.Versions {
		v := &c.Versions[i]
		if v.Number == number {
			if v.ETag != match {
				return Version{}, fmt.Errorf("etag mismatch")
			}
			if v.Lifecycle != Draft {
				return Version{}, fmt.Errorf("only draft versions can be published")
			}
			v.Lifecycle = Published
			v.PublishedAt = &now
			c.Revision++
			c.UpdatedAt = now
			v.ETag = fmt.Sprintf("%s-%d", c.ID, c.Revision)
			return *v, nil
		}
	}
	return Version{}, fmt.Errorf("version not found")
}
