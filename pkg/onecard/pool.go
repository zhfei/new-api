package onecard

import (
	"fmt"

	"github.com/QuantumNous/new-api/setting"
)

type PoolStrategy interface {
	Name() string
	ResolvePool(ctx *RequestContext) (*PoolDecision, error)
	ValidateAccess(ctx *RequestContext, decision *PoolDecision) error
	BuildChannelQuery(ctx *RequestContext, decision *PoolDecision) (*ChannelQuery, error)
}

type BasePoolStrategy struct {
	name string
}

func (s *BasePoolStrategy) Name() string {
	return s.name
}

func (s *BasePoolStrategy) ValidateAccess(ctx *RequestContext, decision *PoolDecision) error {
	if ctx == nil || decision == nil {
		return fmt.Errorf("onecard pool context is empty")
	}
	if !setting.IsOneCardGroup(decision.RequestedPool) {
		return fmt.Errorf("invalid onecard pool group: %s", decision.RequestedPool)
	}
	if decision.RequestedPool == GroupAuto && !setting.ValidateOneCardAutoGroups(setting.GetAutoGroups()) {
		return fmt.Errorf("onecard auto requires AutoGroups = [\"free\", \"plus\", \"pro\"]")
	}
	return nil
}

func (s *BasePoolStrategy) BuildChannelQuery(ctx *RequestContext, decision *PoolDecision) (*ChannelQuery, error) {
	if ctx == nil || decision == nil {
		return nil, fmt.Errorf("onecard pool context is empty")
	}
	return &ChannelQuery{Group: decision.Pool, Model: ctx.Model}, nil
}

type FixedPoolStrategy struct {
	BasePoolStrategy
	pool string
}

func NewFixedPoolStrategy(pool string) *FixedPoolStrategy {
	return &FixedPoolStrategy{
		BasePoolStrategy: BasePoolStrategy{name: pool},
		pool:             pool,
	}
}

func (s *FixedPoolStrategy) ResolvePool(ctx *RequestContext) (*PoolDecision, error) {
	return &PoolDecision{RequestedPool: s.pool, Pool: s.pool}, nil
}

type AutoPoolStrategy struct {
	BasePoolStrategy
}

func NewAutoPoolStrategy() *AutoPoolStrategy {
	return &AutoPoolStrategy{BasePoolStrategy: BasePoolStrategy{name: GroupAuto}}
}

func (s *AutoPoolStrategy) ResolvePool(ctx *RequestContext) (*PoolDecision, error) {
	return &PoolDecision{
		RequestedPool: GroupAuto,
		Pool:          GroupAuto,
		FallbackPools: setting.GetAutoGroups(),
	}, nil
}

type PoolRegistry struct {
	strategies map[string]PoolStrategy
}

func NewPoolRegistry() *PoolRegistry {
	r := &PoolRegistry{strategies: map[string]PoolStrategy{}}
	r.Register(NewFixedPoolStrategy(GroupFree))
	r.Register(NewFixedPoolStrategy(GroupPlus))
	r.Register(NewFixedPoolStrategy(GroupPro))
	r.Register(NewAutoPoolStrategy())
	return r
}

func (r *PoolRegistry) Register(strategy PoolStrategy) {
	r.strategies[strategy.Name()] = strategy
}

func (r *PoolRegistry) Get(group string) PoolStrategy {
	return r.strategies[group]
}
