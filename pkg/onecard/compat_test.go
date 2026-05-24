package onecard

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestCodexEndpointPolicyRejectsChatCompletions(t *testing.T) {
	policy := NewEndpointPolicyRegistry().Match(ChannelInfo{Type: constant.ChannelTypeCodex})
	if policy == nil {
		t.Fatal("expected codex endpoint policy")
	}

	err := policy.ValidateEndpoint(&RequestContext{Path: "/v1/chat/completions"}, ChannelInfo{Type: constant.ChannelTypeCodex})
	if err == nil {
		t.Fatal("expected chat completions to be rejected for codex")
	}

	if err := policy.ValidateEndpoint(&RequestContext{Path: "/v1/responses"}, ChannelInfo{Type: constant.ChannelTypeCodex}); err != nil {
		t.Fatalf("expected responses to be accepted for codex: %v", err)
	}
	if err := policy.ValidateEndpoint(&RequestContext{Path: "/v1/responses/compact"}, ChannelInfo{Type: constant.ChannelTypeCodex}); err != nil {
		t.Fatalf("expected compact responses to be accepted for codex: %v", err)
	}
}

func TestDetectInterfaceType(t *testing.T) {
	cases := map[string]string{
		"/v1/chat/completions":   InterfaceChat,
		"/v1/responses":          InterfaceResponses,
		"/v1/responses/compact":  InterfaceResponsesCompact,
		"/backend-api/responses": InterfaceResponses,
		"/BACKEND-API/RESPONSES": InterfaceResponses,
		"/backend-api/something": InterfaceChat,
	}

	for path, want := range cases {
		if got := DetectInterfaceType(path); got != want {
			t.Fatalf("DetectInterfaceType(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestSupportsEndpoint(t *testing.T) {
	if SupportsEndpoint(constant.ChannelTypeCodex, "/v1/chat/completions") {
		t.Fatal("expected codex to reject chat completions")
	}
	if !SupportsEndpoint(constant.ChannelTypeCodex, "/v1/responses") {
		t.Fatal("expected codex to support responses")
	}
	if !SupportsEndpoint(constant.ChannelTypeOpenAI, "/v1/chat/completions") {
		t.Fatal("expected openai to support chat completions")
	}
	if !SupportsEndpoint(999999, "/v1/chat/completions") {
		t.Fatal("expected unknown channel types to fall back to existing behavior")
	}
}

func TestSupportsRequestEndpointAllowsCodexChatViaResponses(t *testing.T) {
	if !SupportsRequestEndpoint(&RequestContext{
		Model: "gpt-5.4",
		Path:  "/v1/chat/completions",
	}, ChannelInfo{Type: constant.ChannelTypeCodex}) {
		t.Fatal("expected codex gpt chat completions to be allowed via responses compatibility")
	}
	if SupportsRequestEndpoint(&RequestContext{
		Model: "claude-sonnet-4-5",
		Path:  "/v1/chat/completions",
	}, ChannelInfo{Type: constant.ChannelTypeCodex}) {
		t.Fatal("expected non-gpt codex chat completions to be rejected")
	}
	if SupportsRequestEndpoint(&RequestContext{
		Model: "gpt-5.4",
		Path:  "/v1/messages",
	}, ChannelInfo{Type: constant.ChannelTypeCodex}) {
		t.Fatal("expected codex messages endpoint to be rejected")
	}
	if SupportsRequestEndpoint(&RequestContext{
		Model: "gpt-5.4",
		Path:  "/backend-api/something",
	}, ChannelInfo{Type: constant.ChannelTypeCodex}) {
		t.Fatal("expected unknown codex endpoint to be rejected")
	}
}
