package onecard

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/constant"
)

type EndpointPolicy interface {
	Name() string
	Match(channelType int) bool
	SupportedInterfaces() []string
	ValidateEndpoint(ctx *RequestContext, channel ChannelInfo) error
}

type BaseEndpointPolicy struct {
	name       string
	types      map[int]struct{}
	interfaces []string
}

func (p *BaseEndpointPolicy) Name() string {
	return p.name
}

func (p *BaseEndpointPolicy) Match(channelType int) bool {
	_, ok := p.types[channelType]
	return ok
}

func (p *BaseEndpointPolicy) SupportedInterfaces() []string {
	return append([]string(nil), p.interfaces...)
}

func (p *BaseEndpointPolicy) ValidateEndpoint(ctx *RequestContext, channel ChannelInfo) error {
	if ctx == nil {
		return fmt.Errorf("onecard endpoint context is empty")
	}
	interfaceType := DetectInterfaceType(ctx.Path)
	for _, supported := range p.interfaces {
		if supported == interfaceType {
			return nil
		}
	}
	return fmt.Errorf("%s channel does not support %s endpoint", p.name, interfaceType)
}

type CodexEndpointPolicy struct {
	BaseEndpointPolicy
}

func NewCodexEndpointPolicy() *CodexEndpointPolicy {
	return &CodexEndpointPolicy{BaseEndpointPolicy: BaseEndpointPolicy{
		name:       "codex",
		types:      map[int]struct{}{constant.ChannelTypeCodex: {}},
		interfaces: []string{InterfaceResponses, InterfaceResponsesCompact},
	}}
}

func (p *CodexEndpointPolicy) ValidateEndpoint(ctx *RequestContext, channel ChannelInfo) error {
	if err := p.BaseEndpointPolicy.ValidateEndpoint(ctx, channel); err != nil {
		return fmt.Errorf("Codex 渠道只支持 /v1/responses 和 /v1/responses/compact，不支持当前接口")
	}
	return nil
}

func NewOpenAICompatibleEndpointPolicy() *BaseEndpointPolicy {
	return &BaseEndpointPolicy{
		name: "openai_compatible",
		types: map[int]struct{}{
			constant.ChannelTypeOpenAI:    {},
			constant.ChannelTypeOpenAIMax: {},
			constant.ChannelTypeCustom:    {},
		},
		interfaces: []string{InterfaceChat, InterfaceResponses},
	}
}

func NewClaudeEndpointPolicy() *BaseEndpointPolicy {
	return &BaseEndpointPolicy{
		name:       "claude",
		types:      map[int]struct{}{constant.ChannelTypeAnthropic: {}},
		interfaces: []string{InterfaceChat},
	}
}

func NewGeminiEndpointPolicy() *BaseEndpointPolicy {
	return &BaseEndpointPolicy{
		name:       "gemini",
		types:      map[int]struct{}{constant.ChannelTypeGemini: {}},
		interfaces: []string{InterfaceChat},
	}
}

type EndpointPolicyRegistry struct {
	policies []EndpointPolicy
}

func NewEndpointPolicyRegistry() *EndpointPolicyRegistry {
	return &EndpointPolicyRegistry{policies: []EndpointPolicy{
		NewCodexEndpointPolicy(),
		NewOpenAICompatibleEndpointPolicy(),
		NewClaudeEndpointPolicy(),
		NewGeminiEndpointPolicy(),
	}}
}

func (r *EndpointPolicyRegistry) Match(channel ChannelInfo) EndpointPolicy {
	for _, policy := range r.policies {
		if policy.Match(channel.Type) {
			return policy
		}
	}
	return nil
}

func SupportsEndpoint(channelType int, path string) bool {
	policy := NewEndpointPolicyRegistry().Match(ChannelInfo{Type: channelType})
	if policy == nil {
		return true
	}
	return policy.ValidateEndpoint(&RequestContext{Path: path}, ChannelInfo{Type: channelType}) == nil
}

func DetectInterfaceType(path string) string {
	normalized := strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.Contains(normalized, "/responses/compact"):
		return InterfaceResponsesCompact
	case strings.Contains(normalized, "/responses"):
		return InterfaceResponses
	default:
		return InterfaceChat
	}
}
