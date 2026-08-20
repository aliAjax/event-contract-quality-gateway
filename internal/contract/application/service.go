package application

import (
	"context"
	"fmt"
	"github.com/example/event-contract-quality-gateway/internal/contract/domain"
	"time"
)

type Repository interface {
	Create(context.Context, domain.Contract) error
	Find(context.Context, string, string) (domain.Contract, error)
	Save(context.Context, domain.Contract) error
	List(context.Context, string, int, string) ([]domain.Contract, string, error)
}
type Service struct {
	repo Repository
	now  func() time.Time
	ids  func() string
}

func New(repo Repository, ids func() string) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }, ids: ids}
}

type RegisterCommand struct {
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	Compatibility domain.Compatibility `json:"compatibility"`
}

func (s *Service) Register(ctx context.Context, tenant string, cmd RegisterCommand) (domain.Contract, error) {
	c, e := domain.NewContract(s.ids(), tenant, cmd.Name, cmd.Description, cmd.Compatibility, s.now())
	if e != nil {
		return domain.Contract{}, e
	}
	e = s.repo.Create(ctx, c)
	return c, e
}
func (s *Service) AddVersion(ctx context.Context, tenant, id string, schema domain.Schema) (domain.Version, error) {
	c, e := s.repo.Find(ctx, tenant, id)
	if e != nil {
		return domain.Version{}, e
	}
	v, e := c.AddVersion(schema, s.now())
	if e != nil {
		return domain.Version{}, e
	}
	return v, s.repo.Save(ctx, c)
}
func (s *Service) Publish(ctx context.Context, tenant, id string, n int, etag string) (domain.Version, error) {
	c, e := s.repo.Find(ctx, tenant, id)
	if e != nil {
		return domain.Version{}, e
	}
	v, e := c.Publish(n, etag, s.now())
	if e != nil {
		return domain.Version{}, e
	}
	return v, s.repo.Save(ctx, c)
}
func (s *Service) Get(ctx context.Context, tenant, id string) (domain.Contract, error) {
	return s.repo.Find(ctx, tenant, id)
}
func (s *Service) Check(ctx context.Context, tenant, id string, candidate domain.Schema) (domain.Report, error) {
	c, e := s.repo.Find(ctx, tenant, id)
	if e != nil {
		return domain.Report{}, e
	}
	if len(c.Versions) == 0 {
		return domain.Report{}, fmt.Errorf("contract has no versions")
	}
	if e := candidate.Validate(); e != nil {
		return domain.Report{}, e
	}
	return domain.Compare(c.Versions[len(c.Versions)-1].Schema, candidate, c.Compatibility), nil
}
