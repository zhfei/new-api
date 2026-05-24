package onecard

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type ProviderAdapter interface {
	Type() int
	Name() string
	SupportedInterfaces() []string
	NormalizeCredential(raw map[string]interface{}) (map[string]interface{}, error)
	BuildChannel(input *AccountImportItem, pool string) (*ChannelDraft, error)
}

type BaseProvider struct {
	channelType int
	name        string
	baseURL     string
	models      []string
	interfaces  []string
}

func (p *BaseProvider) Type() int {
	return p.channelType
}

func (p *BaseProvider) Name() string {
	return p.name
}

func (p *BaseProvider) SupportedInterfaces() []string {
	return append([]string(nil), p.interfaces...)
}

func (p *BaseProvider) NormalizeCredential(raw map[string]interface{}) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%s credential is empty", p.name)
	}
	return raw, nil
}

func (p *BaseProvider) BuildChannel(input *AccountImportItem, pool string) (*ChannelDraft, error) {
	return p.buildChannelWithAdapter(p, input, pool)
}

func (p *BaseProvider) buildChannelWithAdapter(adapter ProviderAdapter, input *AccountImportItem, pool string) (*ChannelDraft, error) {
	credential := input.Credential
	if len(credential) == 0 {
		credential = input.Credentials
	}
	normalized, err := adapter.NormalizeCredential(credential)
	if err != nil {
		return nil, err
	}
	key, err := credentialToKey(normalized)
	if err != nil {
		return nil, err
	}
	models := input.Models
	if len(models) == 0 {
		models = p.models
	}
	baseURL := input.BaseURL
	if baseURL == "" {
		baseURL = p.baseURL
	}
	name := input.Name
	if name == "" {
		name = input.Email
	}
	if name == "" {
		if email, ok := input.Extra["email"].(string); ok {
			name = strings.TrimSpace(email)
		}
	}
	if name == "" {
		name = fmt.Sprintf("%s-%s", p.name, pool)
	}
	tag := input.Tag
	if tag == "" {
		tag = p.name + "-" + pool
	}
	return &ChannelDraft{
		Name:         fmt.Sprintf("%s-%s-%s", p.name, pool, name),
		Type:         adapter.Type(),
		Key:          key,
		BaseURL:      baseURL,
		Models:       strings.Join(models, ","),
		Group:        pool,
		ModelMapping: input.ModelMapping,
		Tag:          tag,
		Priority:     input.Priority,
		Weight:       input.Weight,
	}, nil
}

type CodexProvider struct {
	BaseProvider
}

func NewCodexProvider() *CodexProvider {
	return &CodexProvider{BaseProvider: BaseProvider{
		channelType: constant.ChannelTypeCodex,
		name:        ProviderCodex,
		baseURL:     "https://chatgpt.com",
		models:      codexDefaultModels(),
		interfaces:  []string{InterfaceResponses, InterfaceResponsesCompact},
	}}
}

func codexDefaultModels() []string {
	base := []string{
		"gpt-5", "gpt-5-codex", "gpt-5-codex-mini",
		"gpt-5.1", "gpt-5.1-codex", "gpt-5.1-codex-max", "gpt-5.1-codex-mini",
		"gpt-5.2", "gpt-5.2-codex", "gpt-5.3-codex", "gpt-5.3-codex-spark",
		"gpt-5.4",
	}
	models := make([]string, 0, len(base)*2)
	models = append(models, base...)
	for _, model := range base {
		models = append(models, ratio_setting.WithCompactModelSuffix(model))
	}
	return models
}

func (p *CodexProvider) BuildChannel(input *AccountImportItem, pool string) (*ChannelDraft, error) {
	return p.BaseProvider.buildChannelWithAdapter(p, input, pool)
}

func (p *CodexProvider) NormalizeCredential(raw map[string]interface{}) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("codex credential is empty")
	}
	accessToken, ok := raw["access_token"].(string)
	if !ok || strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("codex credential requires access_token")
	}
	normalized := cloneCredential(raw)
	if accountID := firstString(normalized, "account_id", "chatgpt_account_id"); accountID != "" {
		normalized["account_id"] = accountID
		return normalized, nil
	}
	if accountID, ok := extractCodexAccountIDFromJWT(accessToken); ok {
		normalized["account_id"] = accountID
		return normalized, nil
	}
	return nil, fmt.Errorf("codex credential requires account_id or chatgpt_account_id")
}

type ProviderRegistry struct {
	byName map[string]ProviderAdapter
}

func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{byName: map[string]ProviderAdapter{}}
	r.Register(NewCodexProvider())
	r.Register(&BaseProvider{channelType: constant.ChannelTypeOpenAI, name: ProviderOpenAI, baseURL: "https://api.openai.com", models: []string{"gpt-5"}, interfaces: []string{InterfaceChat, InterfaceResponses}})
	r.Register(&BaseProvider{channelType: constant.ChannelTypeAnthropic, name: ProviderClaude, baseURL: "https://api.anthropic.com", models: []string{"claude-sonnet-4-5"}, interfaces: []string{InterfaceChat}})
	r.Register(&BaseProvider{channelType: constant.ChannelTypeGemini, name: ProviderGemini, baseURL: "https://generativelanguage.googleapis.com", models: []string{"gemini-2.5-pro"}, interfaces: []string{InterfaceChat}})
	return r
}

func (r *ProviderRegistry) Register(provider ProviderAdapter) {
	r.byName[provider.Name()] = provider
}

func (r *ProviderRegistry) GetProviderByName(name string) ProviderAdapter {
	return r.byName[strings.ToLower(strings.TrimSpace(name))]
}

func credentialToKey(credential map[string]interface{}) (string, error) {
	for _, key := range []string{"api_key", "key"} {
		if value, ok := credential[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), nil
		}
	}
	bytes, err := common.Marshal(credential)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func cloneCredential(raw map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{}, len(raw)+1)
	for key, value := range raw {
		normalized[key] = value
	}
	return normalized
}

func firstString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return ""
}

func extractCodexAccountIDFromJWT(token string) (string, bool) {
	claims, ok := decodeJWTClaims(token)
	if !ok {
		return "", false
	}
	if accountID := firstString(claims,
		"https://api.openai.com/auth.chatgpt_account_id",
		"https://api.openai.com/auth.account_id",
		"chatgpt_account_id",
		"account_id",
	); accountID != "" {
		return accountID, true
	}
	for _, claim := range []string{
		"https://api.openai.com/auth",
		"https://api.openai.com/auth_claims",
	} {
		if accountID := accountIDFromClaim(claims[claim]); accountID != "" {
			return accountID, true
		}
	}
	return "", false
}

func accountIDFromClaim(raw interface{}) string {
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	return firstString(obj, "chatgpt_account_id", "account_id")
}

func decodeJWTClaims(token string) (map[string]interface{}, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, false
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	var claims map[string]interface{}
	if err := common.Unmarshal(payloadRaw, &claims); err != nil {
		return nil, false
	}
	return claims, true
}
