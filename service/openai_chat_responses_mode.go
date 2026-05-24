package service

import (
	"github.com/QuantumNous/new-api/pkg/onecard"
	"github.com/QuantumNous/new-api/service/openaicompat"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	return openaicompat.ShouldChatCompletionsUseResponsesPolicy(policy, channelID, channelType, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	if onecard.ShouldUseChatCompletionsToResponses(&onecard.RequestContext{
		Model: model,
		Path:  "/v1/chat/completions",
	}, onecard.ChannelInfo{ID: channelID, Type: channelType}) {
		return true
	}
	return openaicompat.ShouldChatCompletionsUseResponsesGlobal(channelID, channelType, model)
}
