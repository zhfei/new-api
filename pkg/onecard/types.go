package onecard

const (
	GroupFree = "free"
	GroupPlus = "plus"
	GroupPro  = "pro"
	GroupAuto = "auto"

	ProviderCodex  = "codex"
	ProviderOpenAI = "openai"
	ProviderClaude = "claude"
	ProviderGemini = "gemini"

	InterfaceChat             = "chat"
	InterfaceResponses        = "responses"
	InterfaceResponsesCompact = "responses_compact"
)

type RequestContext struct {
	TokenGroup string
	UserGroup  string
	Model      string
	Path       string
}

type PoolDecision struct {
	RequestedPool string
	Pool          string
	FallbackPools []string
}

type ChannelQuery struct {
	Group string
	Model string
}

type ChannelInfo struct {
	ID   int
	Type int
}

type RouteDecision struct {
	RequestedPool  string
	Pool           string
	UserModel      string
	UpstreamModel  string
	ChannelID      int
	ChannelType    int
	InterfaceType  string
	EndpointPolicy string
	FallbackUsed   bool
}

type AccountImportItem struct {
	Pool             string                 `json:"pool"`
	Provider         string                 `json:"provider"`
	Type             string                 `json:"type"`
	Email            string                 `json:"email"`
	Name             string                 `json:"name"`
	AccessToken      string                 `json:"access_token"`
	RefreshToken     string                 `json:"refresh_token"`
	AccountID        string                 `json:"account_id"`
	ChatGPTAccountID string                 `json:"chatgpt_account_id"`
	Credential       map[string]interface{} `json:"credential"`
	Credentials      map[string]interface{} `json:"credentials"`
	Extra            map[string]interface{} `json:"extra"`
	Models           []string               `json:"models"`
	ModelMapping     string                 `json:"model_mapping"`
	BaseURL          string                 `json:"base_url"`
	Tag              string                 `json:"tag"`
	Priority         int64                  `json:"priority"`
	Weight           uint                   `json:"weight"`
}

type AccountImportEnvelope struct {
	Pool     string              `json:"pool"`
	Provider string              `json:"provider"`
	Items    []AccountImportItem `json:"items"`
	Accounts []AccountImportItem `json:"accounts"`
}

type ChannelDraft struct {
	Name         string
	Type         int
	Key          string
	BaseURL      string
	Models       string
	Group        string
	ModelMapping string
	Tag          string
	Priority     int64
	Weight       uint
}

type ImportResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}
