package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestShouldChatCompletionsUseResponsesGlobalOneCardCodexDefault(t *testing.T) {
	if !ShouldChatCompletionsUseResponsesGlobal(1, constant.ChannelTypeCodex, "gpt-5.4") {
		t.Fatal("expected onecard codex gpt models to use responses compatibility by default")
	}
	if ShouldChatCompletionsUseResponsesGlobal(1, constant.ChannelTypeCodex, "claude-sonnet-4-5") {
		t.Fatal("expected non-gpt codex models not to use responses compatibility by default")
	}
}
