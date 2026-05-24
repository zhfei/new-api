package onecard

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

type memoryChannelWriter struct {
	drafts []*ChannelDraft
}

func (w *memoryChannelWriter) CreateFromDraft(ctx context.Context, draft *ChannelDraft) error {
	w.drafts = append(w.drafts, draft)
	return nil
}

func TestAccountImporterImportsSub2APIStyleCodexAccount(t *testing.T) {
	writer := &memoryChannelWriter{}
	importer := NewAccountImporter(writer)

	result, err := importer.Import(context.Background(), []AccountImportItem{
		{
			Pool:     GroupPlus,
			Provider: ProviderCodex,
			Name:     "codex-account",
			Credentials: map[string]interface{}{
				"access_token":       "access-token",
				"refresh_token":      "refresh-token",
				"chatgpt_account_id": "chatgpt-account-id",
			},
			Extra: map[string]interface{}{
				"email": "user@example.com",
			},
		},
	})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Created != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	if len(writer.drafts) != 1 {
		t.Fatalf("expected one draft, got %d", len(writer.drafts))
	}

	draft := writer.drafts[0]
	if draft.Group != GroupPlus {
		t.Fatalf("expected group %q, got %q", GroupPlus, draft.Group)
	}
	if draft.Type == 0 {
		t.Fatal("expected codex channel type")
	}
	if !strings.Contains(draft.Key, "access-token") {
		t.Fatalf("expected serialized credential key to contain access token, got %s", draft.Key)
	}
	if !strings.Contains(draft.Key, "chatgpt-account-id") {
		t.Fatalf("expected serialized credential key to contain account id, got %s", draft.Key)
	}
}

func TestAccountImporterUsesExtraEmailAsName(t *testing.T) {
	writer := &memoryChannelWriter{}
	importer := NewAccountImporter(writer)

	result, err := importer.Import(context.Background(), []AccountImportItem{
		{
			Pool:     GroupFree,
			Provider: ProviderCodex,
			Credentials: map[string]interface{}{
				"access_token": "access-token",
				"account_id":    "account-id",
			},
			Extra: map[string]interface{}{
				"email": "extra@example.com",
			},
		},
	})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("expected one created account, got %+v", result)
	}
	if !strings.Contains(writer.drafts[0].Name, "extra@example.com") {
		t.Fatalf("expected draft name to include extra email, got %q", writer.drafts[0].Name)
	}
}

func TestAccountImporterExtractsCodexAccountIDFromJWT(t *testing.T) {
	writer := &memoryChannelWriter{}
	importer := NewAccountImporter(writer)
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"https://api.openai.com/auth":{"chatgpt_account_id":"jwt-account-id"}}`))
	token := "header." + payload + ".signature"

	result, err := importer.Import(context.Background(), []AccountImportItem{
		{
			Pool:     GroupPlus,
			Provider: ProviderCodex,
			Credentials: map[string]interface{}{
				"access_token": token,
			},
		},
	})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Created != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	if !strings.Contains(writer.drafts[0].Key, "jwt-account-id") {
		t.Fatalf("expected serialized credential key to contain JWT account id, got %s", writer.drafts[0].Key)
	}
}

func TestAccountImporterSkipsCodexCredentialWithoutAccountID(t *testing.T) {
	writer := &memoryChannelWriter{}
	importer := NewAccountImporter(writer)

	result, err := importer.Import(context.Background(), []AccountImportItem{
		{
			Pool:     GroupPlus,
			Provider: ProviderCodex,
			Credentials: map[string]interface{}{
				"access_token": "access-token",
			},
		},
	})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Created != 0 || result.Skipped != 1 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	if len(result.Errors) == 0 || !strings.Contains(result.Errors[0], "account_id") {
		t.Fatalf("expected account_id error, got %+v", result.Errors)
	}
	if len(writer.drafts) != 0 {
		t.Fatalf("expected no drafts, got %d", len(writer.drafts))
	}
}

func TestAccountImporterSkipsInvalidPool(t *testing.T) {
	writer := &memoryChannelWriter{}
	importer := NewAccountImporter(writer)

	result, err := importer.Import(context.Background(), []AccountImportItem{
		{
			Pool:     GroupAuto,
			Provider: ProviderCodex,
			Credentials: map[string]interface{}{
				"access_token": "access-token",
				"account_id":   "account-id",
			},
		},
	})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Created != 0 || result.Skipped != 1 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	if len(writer.drafts) != 0 {
		t.Fatalf("expected no drafts, got %d", len(writer.drafts))
	}
}
