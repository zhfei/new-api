package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

func TestValidateOneCardEndpointRejectsCodexChatCompletions(t *testing.T) {
	originalEnabled := setting.OneCardEnabled()
	t.Cleanup(func() {
		setting.SetOneCardEnabled(originalEnabled)
	})
	setting.SetOneCardEnabled(true)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	err := validateOneCardEndpoint(c, &relaycommon.RelayInfo{
		TokenGroup:      "free",
		UserGroup:       "default",
		OriginModelName: "gpt-5",
	}, &model.Channel{
		Id:   1,
		Type: constant.ChannelTypeCodex,
	})
	if err == nil {
		t.Fatal("expected codex chat completions to be rejected")
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 status, got %d", err.StatusCode)
	}
}

func TestValidateOneCardEndpointAllowsCodexResponses(t *testing.T) {
	originalEnabled := setting.OneCardEnabled()
	t.Cleanup(func() {
		setting.SetOneCardEnabled(originalEnabled)
	})
	setting.SetOneCardEnabled(true)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	err := validateOneCardEndpoint(c, &relaycommon.RelayInfo{
		TokenGroup:      "free",
		UserGroup:       "default",
		OriginModelName: "gpt-5",
	}, &model.Channel{
		Id:   1,
		Type: constant.ChannelTypeCodex,
	})
	if err != nil {
		t.Fatalf("expected codex responses to be allowed: %v", err)
	}
}
