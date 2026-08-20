package store

import (
	"context"
	"fmt"
	"github.com/example/event-contract-quality-gateway/internal/contract/domain"
	"sort"
	"sync"
)

type MemoryContracts struct {
	mu     sync.RWMutex
	values map[string]domain.Contract
}

func NewMemoryContracts() *MemoryContracts {
	return &MemoryContracts{values: map[string]domain.Contract{}}
}
func key(tenant, id string) string { return tenant + "/" + id }
func (m *MemoryContracts) Create(_ context.Context, c domain.Contract) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := key(c.OrganizationID, c.ID)
	if _, ok := m.values[k]; ok {
		return fmt.Errorf("contract already exists")
	}
	m.values[k] = c
	return nil
}
func (m *MemoryContracts) Find(_ context.Context, tenant, id string) (domain.Contract, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.values[key(tenant, id)]
	if !ok {
		return domain.Contract{}, fmt.Errorf("contract not found")
	}
	return clone(c), nil
}
func (m *MemoryContracts) Save(_ context.Context, c domain.Contract) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := key(c.OrganizationID, c.ID)
	if _, ok := m.values[k]; !ok {
		return fmt.Errorf("contract not found")
	}
	m.values[k] = clone(c)
	return nil
}
func (m *MemoryContracts) List(_ context.Context, tenant string, limit int, cursor string) ([]domain.Contract, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Contract, 0)
	for _, c := range m.values {
		if c.OrganizationID == tenant {
			out = append(out, clone(c))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	start := 0
	for i, c := range out {
		if c.ID == cursor {
			start = i + 1
			break
		}
	}
	end := start + limit
	if end > len(out) {
		end = len(out)
	}
	next := ""
	if end < len(out) && end > start {
		next = out[end-1].ID
	}
	return out[start:end], next, nil
}
func clone(c domain.Contract) domain.Contract {
	c.Versions = append([]domain.Version(nil), c.Versions...)
	for i := range c.Versions {
		c.Versions[i].Schema.Fields = append([]domain.Field(nil), c.Versions[i].Schema.Fields...)
	}
	return c
}
