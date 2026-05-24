package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/onecard"
)

func TestRecoverOneCard429MultiKeyChannelIfDue(t *testing.T) {
	truncate(t)
	common.MemoryCacheEnabled = false

	now := int64(1716547200)
	channel := seedOneCard429MultiKeyChannel(t, now)

	recoverOneCard429MultiKeyChannelIfDue(context.Background(), channel, now)

	var reloaded model.Channel
	require.NoError(t, model.DB.First(&reloaded, channel.Id).Error)
	require.Equal(t, common.ChannelStatusEnabled, reloaded.Status)
	require.NotContains(t, reloaded.ChannelInfo.MultiKeyStatusList, 0)
	require.Equal(t, common.ChannelStatusAutoDisabled, reloaded.ChannelInfo.MultiKeyStatusList[1])

	info := reloaded.GetOtherInfo()
	require.True(t, onecard.Is429AutoRecoverInfo(info))
	require.True(t, onecard.Has429AutoRecoverKeys(info))
	require.Equal(t, []int{1}, onecard.Due429AutoRecoverKeyIndexes(info, now+7200))

	var ability model.Ability
	require.NoError(t, model.DB.First(&ability, "channel_id = ?", channel.Id).Error)
	require.True(t, ability.Enabled)
}

func TestRunOneCard429AutoRecoverOnceConcurrent(t *testing.T) {
	truncate(t)
	common.MemoryCacheEnabled = false

	now := common.GetTimestamp()
	channel := seedOneCard429SingleKeyChannel(t, now-1)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runOneCard429AutoRecoverOnce()
		}()
	}
	wg.Wait()

	var reloaded model.Channel
	require.NoError(t, model.DB.First(&reloaded, channel.Id).Error)
	require.Equal(t, common.ChannelStatusEnabled, reloaded.Status)
	require.False(t, onecard.Is429AutoRecoverInfo(reloaded.GetOtherInfo()))

	var ability model.Ability
	require.NoError(t, model.DB.First(&ability, "channel_id = ?", channel.Id).Error)
	require.True(t, ability.Enabled)
}

func seedOneCard429SingleKeyChannel(t *testing.T, recoverAt int64) *model.Channel {
	t.Helper()
	channel := &model.Channel{
		Type:    constant.ChannelTypeCodex,
		Key:     "single-key",
		Status:  common.ChannelStatusAutoDisabled,
		Name:    "onecard-single-key",
		Models:  "gpt-5.4",
		Group:   onecard.GroupPlus,
		AutoBan: common.GetPointer(1),
	}
	channel.SetOtherInfo(map[string]interface{}{
		onecard.OtherInfoStatusReasonKey:      onecard.AutoRecoverStatusReason429,
		onecard.OtherInfoStatusTimeKey:        recoverAt - 1,
		onecard.OtherInfoAutoRecoverReasonKey: onecard.AutoRecoverReason429,
		onecard.OtherInfoAutoRecoverAtKey:     recoverAt,
	})
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     onecard.GroupPlus,
		Model:     "gpt-5.4",
		ChannelId: channel.Id,
		Enabled:   false,
	}).Error)
	return channel
}

func seedOneCard429MultiKeyChannel(t *testing.T, now int64) *model.Channel {
	t.Helper()
	channel := &model.Channel{
		Type:    constant.ChannelTypeCodex,
		Key:     "key-a\nkey-b",
		Status:  common.ChannelStatusAutoDisabled,
		Name:    "onecard-multi-key",
		Models:  "gpt-5.4",
		Group:   onecard.GroupPlus,
		AutoBan: common.GetPointer(1),
		ChannelInfo: model.ChannelInfo{
			IsMultiKey:   true,
			MultiKeySize: 2,
			MultiKeyMode: constant.MultiKeyModePolling,
			MultiKeyStatusList: map[int]int{
				0: common.ChannelStatusAutoDisabled,
				1: common.ChannelStatusAutoDisabled,
			},
			MultiKeyDisabledReason: map[int]string{
				0: onecard.AutoRecoverStatusReason429,
				1: onecard.AutoRecoverStatusReason429,
			},
			MultiKeyDisabledTime: map[int]int64{
				0: now - 10,
				1: now - 10,
			},
		},
	}
	info := map[string]interface{}{
		onecard.OtherInfoStatusReasonKey:      "All keys are disabled",
		onecard.OtherInfoStatusTimeKey:        now - 10,
		onecard.OtherInfoAutoRecoverReasonKey: onecard.AutoRecoverReason429,
		onecard.OtherInfoAutoRecoverAtKey:     now - 1,
	}
	info = onecard.Set429AutoRecoverKeyInfo(info, 0, map[string]interface{}{
		onecard.OtherInfoAutoRecoverReasonKey: onecard.AutoRecoverReason429,
		onecard.OtherInfoAutoRecoverAtKey:     now - 1,
	})
	info = onecard.Set429AutoRecoverKeyInfo(info, 1, map[string]interface{}{
		onecard.OtherInfoAutoRecoverReasonKey: onecard.AutoRecoverReason429,
		onecard.OtherInfoAutoRecoverAtKey:     now + 3600,
	})
	channel.SetOtherInfo(info)
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     onecard.GroupPlus,
		Model:     "gpt-5.4",
		ChannelId: channel.Id,
		Enabled:   false,
	}).Error)
	return channel
}
