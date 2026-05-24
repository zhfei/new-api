package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/util/gopool"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/onecard"
	"github.com/QuantumNous/new-api/setting"
)

const (
	oneCard429AutoRecoverTickInterval = 30 * time.Second
	oneCard429AutoRecoverBatchSize    = 200
)

var (
	oneCard429AutoRecoverOnce    sync.Once
	oneCard429AutoRecoverRunning atomic.Bool
)

func StartOneCard429AutoRecoverTask() {
	oneCard429AutoRecoverOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("OneCard 429 auto-recover task started: tick=%s", oneCard429AutoRecoverTickInterval))

			ticker := time.NewTicker(oneCard429AutoRecoverTickInterval)
			defer ticker.Stop()

			runOneCard429AutoRecoverOnce()
			for range ticker.C {
				runOneCard429AutoRecoverOnce()
			}
		})
	})
}

func runOneCard429AutoRecoverOnce() {
	if !setting.OneCardEnabled() {
		return
	}
	if !oneCard429AutoRecoverRunning.CompareAndSwap(false, true) {
		return
	}
	defer oneCard429AutoRecoverRunning.Store(false)

	ctx := context.Background()
	now := common.GetTimestamp()
	lastID := 0

	for {
		var channels []*model.Channel
		err := model.DB.
			Select("id", "name", "status", "other_info", "channel_info").
			Where("(status = ? OR status = ?) AND id > ?", common.ChannelStatusEnabled, common.ChannelStatusAutoDisabled, lastID).
			Order("id asc").
			Limit(oneCard429AutoRecoverBatchSize).
			Find(&channels).Error
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("OneCard 429 auto-recover: query channels failed: %v", err))
			return
		}
		if len(channels) == 0 {
			return
		}

		for _, channel := range channels {
			if channel == nil {
				continue
			}
			lastID = channel.Id
			recoverOneCard429ChannelIfDue(ctx, channel, now)
		}
	}
}

func recoverOneCard429ChannelIfDue(ctx context.Context, channel *model.Channel, now int64) {
	if channel.ChannelInfo.IsMultiKey {
		recoverOneCard429MultiKeyChannelIfDue(ctx, channel, now)
		return
	}

	if channel.Status != common.ChannelStatusAutoDisabled {
		return
	}
	info := channel.GetOtherInfo()
	if !onecard.Is429AutoRecoverInfo(info) {
		return
	}
	recoverAt, ok := onecard.ParseAutoRecoverAt(info)
	if !ok || recoverAt > now {
		return
	}

	EnableChannel(channel.Id, "", channel.Name)

	latest, err := model.GetChannelById(channel.Id, true)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("OneCard 429 auto-recover: reload channel #%d failed: %v", channel.Id, err))
		return
	}
	cleaned := onecard.Clean429AutoRecoverInfo(latest.GetOtherInfo())
	latest.SetOtherInfo(cleaned)
	if err := latest.SaveWithoutKey(); err != nil {
		logger.LogError(ctx, fmt.Sprintf("OneCard 429 auto-recover: clean other_info failed: channel_id=%d, error=%v", channel.Id, err))
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("OneCard 429 auto-recover: channel #%d restored", channel.Id))
}

func recoverOneCard429MultiKeyChannelIfDue(ctx context.Context, channel *model.Channel, now int64) {
	info := channel.GetOtherInfo()
	keyIndexes := onecard.Due429AutoRecoverKeyIndexes(info, now)
	if len(keyIndexes) == 0 {
		return
	}

	lock := model.GetChannelPollingLock(channel.Id)
	lock.Lock()
	defer lock.Unlock()

	latest, err := model.GetChannelById(channel.Id, true)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("OneCard 429 auto-recover: reload multi-key channel #%d failed: %v", channel.Id, err))
		return
	}
	if !latest.ChannelInfo.IsMultiKey {
		return
	}

	restored := make([]int, 0, len(keyIndexes))
	for _, keyIndex := range keyIndexes {
		if keyIndex < 0 || keyIndex >= latest.ChannelInfo.MultiKeySize {
			continue
		}
		if latest.ChannelInfo.MultiKeyStatusList != nil {
			status, exists := latest.ChannelInfo.MultiKeyStatusList[keyIndex]
			if !exists || status != common.ChannelStatusAutoDisabled {
				continue
			}
			delete(latest.ChannelInfo.MultiKeyStatusList, keyIndex)
		}
		if latest.ChannelInfo.MultiKeyDisabledReason != nil {
			delete(latest.ChannelInfo.MultiKeyDisabledReason, keyIndex)
		}
		if latest.ChannelInfo.MultiKeyDisabledTime != nil {
			delete(latest.ChannelInfo.MultiKeyDisabledTime, keyIndex)
		}
		restored = append(restored, keyIndex)
	}
	if len(restored) == 0 {
		return
	}

	statusChanged := false
	if latest.Status == common.ChannelStatusAutoDisabled && len(latest.ChannelInfo.MultiKeyStatusList) < latest.ChannelInfo.MultiKeySize {
		latest.Status = common.ChannelStatusEnabled
		statusChanged = true
	}

	cleaned := onecard.Clean429AutoRecoverKeyInfo(latest.GetOtherInfo(), restored...)
	latest.SetOtherInfo(cleaned)
	if err := latest.SaveWithoutKey(); err != nil {
		logger.LogError(ctx, fmt.Sprintf("OneCard 429 auto-recover: save multi-key channel #%d failed: %v", latest.Id, err))
		return
	}
	model.InitChannelCache()
	if statusChanged {
		if err := model.UpdateAbilityStatus(latest.Id, true); err != nil {
			logger.LogError(ctx, fmt.Sprintf("OneCard 429 auto-recover: enable ability failed: channel_id=%d, error=%v", latest.Id, err))
		}
	}

	logger.LogInfo(ctx, fmt.Sprintf("OneCard 429 auto-recover: channel #%d keys restored: %v", latest.Id, restored))
}
