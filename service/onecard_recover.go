package service

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/onecard"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
)

func HandleOneCard429AutoDisable(c *gin.Context, channelError types.ChannelError, err *types.NewAPIError) bool {
	if c == nil || err == nil || !channelError.AutoBan {
		return false
	}
	if !setting.OneCardEnabled() {
		return false
	}

	requestCtx := buildOneCardRecoverRequestContext(c)
	if !setting.IsOneCardGroup(requestCtx.TokenGroup) {
		return false
	}

	channel := onecard.ChannelInfo{
		ID:   channelError.ChannelId,
		Type: channelError.ChannelType,
	}
	if !onecard.ShouldAutoDisable429(requestCtx, channel, err) {
		return false
	}

	now := common.GetTimestamp()
	recoverInfo := onecard.Build429AutoRecoverInfo(requestCtx, channel, err, now)
	if saveOneCard429AutoRecoverInfo(channelError, recoverInfo) != nil {
		return false
	}

	common.SysLog(fmt.Sprintf("通道「%s」（#%d）触发 OneCard 429 自动禁用，计划恢复时间：%d", channelError.ChannelName, channelError.ChannelId, recoverInfo[onecard.OtherInfoAutoRecoverAtKey]))
	success := model.UpdateChannelStatus(channelError.ChannelId, channelError.UsingKey, common.ChannelStatusAutoDisabled, onecard.AutoRecoverStatusReason429)
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已因 429 被自动禁用", channelError.ChannelName, channelError.ChannelId)
		content := fmt.Sprintf("通道「%s」（#%d）已因 429 被自动禁用，计划恢复时间：%d", channelError.ChannelName, channelError.ChannelId, recoverInfo[onecard.OtherInfoAutoRecoverAtKey])
		NotifyRootUser(formatNotifyType(channelError.ChannelId, common.ChannelStatusAutoDisabled), subject, content)
		return true
	}

	if isChannelAutoDisabled(channelError.ChannelId) {
		return true
	}

	common.SysLog(fmt.Sprintf("通道「%s」（#%d）触发 OneCard 429，但自动禁用失败", channelError.ChannelName, channelError.ChannelId))
	return false
}

func buildOneCardRecoverRequestContext(c *gin.Context) *onecard.RequestContext {
	path := ""
	if c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	tokenGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	if autoGroup := common.GetContextKeyString(c, constant.ContextKeyAutoGroup); autoGroup != "" {
		tokenGroup = autoGroup
	}
	if tokenGroup == "" {
		tokenGroup = common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
	}
	return &onecard.RequestContext{
		TokenGroup: tokenGroup,
		UserGroup:  common.GetContextKeyString(c, constant.ContextKeyUserGroup),
		Model:      common.GetContextKeyString(c, constant.ContextKeyOriginalModel),
		Path:       path,
	}
}

func saveOneCard429AutoRecoverInfo(channelError types.ChannelError, recoverInfo map[string]interface{}) error {
	channel, err := model.GetChannelById(channelError.ChannelId, true)
	if err != nil {
		common.SysLog(fmt.Sprintf("OneCard 429 自动禁用写入恢复信息失败：channel_id=%d, error=%v", channelError.ChannelId, err))
		return err
	}
	info := channel.GetOtherInfo()
	for key, value := range recoverInfo {
		info[key] = value
	}
	if channel.ChannelInfo.IsMultiKey {
		keyIndex, ok := findChannelKeyIndex(channel, channelError.UsingKey)
		if !ok {
			err := fmt.Errorf("multi-key channel key not found")
			common.SysLog(fmt.Sprintf("OneCard 429 自动禁用保存 key 级恢复信息失败：channel_id=%d, error=%v", channelError.ChannelId, err))
			return err
		}
		info = onecard.Set429AutoRecoverKeyInfo(info, keyIndex, recoverInfo)
	}
	channel.SetOtherInfo(info)
	if err := channel.SaveWithoutKey(); err != nil {
		common.SysLog(fmt.Sprintf("OneCard 429 自动禁用保存恢复信息失败：channel_id=%d, error=%v", channelError.ChannelId, err))
		return err
	}
	return nil
}

func findChannelKeyIndex(channel *model.Channel, usingKey string) (int, bool) {
	if channel == nil || usingKey == "" {
		return 0, false
	}
	keys := channel.GetKeys()
	for index, key := range keys {
		if key == usingKey {
			return index, true
		}
	}
	return 0, false
}

func isChannelAutoDisabled(channelId int) bool {
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		return false
	}
	return channel.Status == common.ChannelStatusAutoDisabled
}
