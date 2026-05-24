package onecard

import "testing"

func TestJSONImportParserFillsEnvelopeDefaults(t *testing.T) {
	items := NewJSONImportParser().Parse(AccountImportEnvelope{
		Pool:     GroupFree,
		Provider: ProviderCodex,
		Accounts: []AccountImportItem{
			{Name: "account-1"},
		},
	})

	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if items[0].Pool != GroupFree {
		t.Fatalf("expected default pool %q, got %q", GroupFree, items[0].Pool)
	}
	if items[0].Provider != ProviderCodex {
		t.Fatalf("expected default provider %q, got %q", ProviderCodex, items[0].Provider)
	}
}

func TestJSONImportParserKeepsItemOverrides(t *testing.T) {
	items := NewJSONImportParser().Parse(AccountImportEnvelope{
		Pool:     GroupFree,
		Provider: ProviderCodex,
		Items: []AccountImportItem{
			{Pool: GroupPro, Provider: ProviderOpenAI},
		},
	})

	if items[0].Pool != GroupPro {
		t.Fatalf("expected item pool override %q, got %q", GroupPro, items[0].Pool)
	}
	if items[0].Provider != ProviderOpenAI {
		t.Fatalf("expected item provider override %q, got %q", ProviderOpenAI, items[0].Provider)
	}
}

func TestFlatCredentialJSONImportParserWrapsSub2APIFlatFields(t *testing.T) {
	items := NewFlatCredentialJSONImportParser().Parse(AccountImportEnvelope{
		Pool:     GroupPlus,
		Provider: ProviderCodex,
		Accounts: []AccountImportItem{
			{
				Name:             "flat-account",
				AccessToken:      "access-token",
				RefreshToken:     "refresh-token",
				ChatGPTAccountID: "chatgpt-account-id",
			},
		},
	})

	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if items[0].Pool != GroupPlus {
		t.Fatalf("expected default pool %q, got %q", GroupPlus, items[0].Pool)
	}
	if items[0].Provider != ProviderCodex {
		t.Fatalf("expected default provider %q, got %q", ProviderCodex, items[0].Provider)
	}
	if items[0].Credential["access_token"] != "access-token" {
		t.Fatalf("expected access_token to be wrapped into credential, got %+v", items[0].Credential)
	}
	if items[0].Credential["refresh_token"] != "refresh-token" {
		t.Fatalf("expected refresh_token to be wrapped into credential, got %+v", items[0].Credential)
	}
	if items[0].Credential["chatgpt_account_id"] != "chatgpt-account-id" {
		t.Fatalf("expected chatgpt_account_id to be wrapped into credential, got %+v", items[0].Credential)
	}
}

func TestFlatCredentialJSONImportParserKeepsNestedCredential(t *testing.T) {
	items := NewFlatCredentialJSONImportParser().Parse(AccountImportEnvelope{
		Pool:     GroupFree,
		Provider: ProviderCodex,
		Accounts: []AccountImportItem{
			{
				AccessToken: "flat-access-token",
				Credential: map[string]interface{}{
					"access_token": "nested-access-token",
					"account_id":   "nested-account-id",
				},
			},
		},
	})

	if items[0].Credential["access_token"] != "nested-access-token" {
		t.Fatalf("expected nested credential to win, got %+v", items[0].Credential)
	}
	if _, ok := items[0].Credential["chatgpt_account_id"]; ok {
		t.Fatalf("expected flat fields not to be merged into existing credential, got %+v", items[0].Credential)
	}
}

func TestImportParserRegistryUsesFlatCredentialParser(t *testing.T) {
	items := NewImportParserRegistry().Parse(AccountImportEnvelope{
		Pool:     GroupPro,
		Provider: ProviderCodex,
		Accounts: []AccountImportItem{
			{AccessToken: "access-token", AccountID: "account-id"},
		},
	})

	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if items[0].Credential["account_id"] != "account-id" {
		t.Fatalf("expected registry to normalize flat credentials, got %+v", items[0].Credential)
	}
}
