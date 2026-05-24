package onecard

import (
	"context"
	"fmt"
	"strings"
)

type ChannelWriter interface {
	CreateFromDraft(ctx context.Context, draft *ChannelDraft) error
}

type AccountImporter struct {
	providers *ProviderRegistry
	writer    ChannelWriter
}

func NewAccountImporter(writer ChannelWriter) *AccountImporter {
	return &AccountImporter{
		providers: NewProviderRegistry(),
		writer:    writer,
	}
}

func (i *AccountImporter) Import(ctx context.Context, items []AccountImportItem) (*ImportResult, error) {
	if i == nil || i.writer == nil {
		return nil, fmt.Errorf("onecard importer writer is nil")
	}
	result := &ImportResult{}
	for idx, item := range items {
		pool := strings.TrimSpace(item.Pool)
		if !isEntityPool(pool) {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("item %d invalid pool: %s", idx, item.Pool))
			continue
		}
		providerName := item.Provider
		if providerName == "" {
			providerName = item.Type
		}
		provider := i.providers.GetProviderByName(providerName)
		if provider == nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("item %d unsupported provider: %s", idx, providerName))
			continue
		}
		draft, err := provider.BuildChannel(&item, pool)
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: %s", idx, err.Error()))
			continue
		}
		if err := i.writer.CreateFromDraft(ctx, draft); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("item %d create channel failed: %s", idx, err.Error()))
			continue
		}
		result.Created++
	}
	return result, nil
}

func isEntityPool(pool string) bool {
	switch pool {
	case GroupFree, GroupPlus, GroupPro:
		return true
	default:
		return false
	}
}
