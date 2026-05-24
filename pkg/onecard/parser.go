package onecard

import "strings"

type ImportParser interface {
	Name() string
	Parse(envelope AccountImportEnvelope) []AccountImportItem
}

type BaseImportParser struct {
	name string
}

func (p *BaseImportParser) Name() string {
	return p.name
}

func (p *BaseImportParser) sourceItems(envelope AccountImportEnvelope) []AccountImportItem {
	items := envelope.Items
	if len(items) == 0 {
		items = envelope.Accounts
	}
	return items
}

func (p *BaseImportParser) applyEnvelopeDefaults(envelope AccountImportEnvelope, item *AccountImportItem) {
	if item == nil {
		return
	}
	if strings.TrimSpace(item.Pool) == "" {
		item.Pool = envelope.Pool
	}
	if strings.TrimSpace(item.Provider) == "" {
		item.Provider = envelope.Provider
	}
}

type JSONImportParser struct {
	BaseImportParser
}

func NewJSONImportParser() *JSONImportParser {
	return &JSONImportParser{BaseImportParser: BaseImportParser{name: "json"}}
}

func (p *JSONImportParser) Parse(envelope AccountImportEnvelope) []AccountImportItem {
	items := p.sourceItems(envelope)
	for i := range items {
		p.applyEnvelopeDefaults(envelope, &items[i])
	}
	return items
}

type FlatCredentialJSONImportParser struct {
	JSONImportParser
}

func NewFlatCredentialJSONImportParser() *FlatCredentialJSONImportParser {
	return &FlatCredentialJSONImportParser{
		JSONImportParser: JSONImportParser{BaseImportParser: BaseImportParser{name: "flat_credential_json"}},
	}
}

func (p *FlatCredentialJSONImportParser) Parse(envelope AccountImportEnvelope) []AccountImportItem {
	items := p.JSONImportParser.Parse(envelope)
	for i := range items {
		p.normalizeFlatCredential(&items[i])
	}
	return items
}

func (p *FlatCredentialJSONImportParser) normalizeFlatCredential(item *AccountImportItem) {
	if item == nil || len(item.Credential) > 0 || len(item.Credentials) > 0 {
		return
	}
	credential := map[string]interface{}{}
	putString := func(key string, value string) {
		if value = strings.TrimSpace(value); value != "" {
			credential[key] = value
		}
	}
	putString("access_token", item.AccessToken)
	putString("refresh_token", item.RefreshToken)
	putString("account_id", item.AccountID)
	putString("chatgpt_account_id", item.ChatGPTAccountID)
	if len(credential) == 0 {
		return
	}
	item.Credential = credential
}

type ImportParserRegistry struct {
	parser ImportParser
}

func NewImportParserRegistry() *ImportParserRegistry {
	return &ImportParserRegistry{parser: NewFlatCredentialJSONImportParser()}
}

func (r *ImportParserRegistry) Parse(envelope AccountImportEnvelope) []AccountImportItem {
	if r == nil || r.parser == nil {
		return nil
	}
	return r.parser.Parse(envelope)
}
