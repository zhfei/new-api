package onecard

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
)

const (
	AutoRecoverReason429       = "429"
	AutoRecoverStatusReason429 = "onecard_429_auto_disabled"

	OtherInfoStatusReasonKey      = "status_reason"
	OtherInfoStatusTimeKey        = "status_time"
	OtherInfoAutoRecoverReasonKey = "onecard_auto_recover_reason"
	OtherInfoAutoRecoverAtKey     = "onecard_auto_recover_at"
	OtherInfoAutoRecoverModelKey  = "onecard_auto_recover_model"
	OtherInfoAutoRecoverGroupKey  = "onecard_auto_recover_group"
	OtherInfoAutoRecoverErrorKey  = "onecard_auto_recover_error"
	OtherInfoAutoRecoverKeysKey   = "onecard_auto_recover_keys"
	OtherInfoAutoRecoverKeyIndex  = "key_index"
	default429RecoverDuration     = 5 * time.Hour
	weekly429RecoverDuration      = 7 * 24 * time.Hour
)

type AutoRecoverPolicy interface {
	Match(channel ChannelInfo) bool
	ShouldAutoDisable(ctx *RequestContext, err *types.NewAPIError) bool
	RecoverAt(ctx *RequestContext, err *types.NewAPIError, now int64) int64
	Reason(ctx *RequestContext, err *types.NewAPIError) string
}

type BaseAutoRecoverPolicy struct{}

func (p *BaseAutoRecoverPolicy) Match(channel ChannelInfo) bool {
	return true
}

func (p *BaseAutoRecoverPolicy) ShouldAutoDisable(ctx *RequestContext, err *types.NewAPIError) bool {
	return err != nil && err.StatusCode == http.StatusTooManyRequests
}

func (p *BaseAutoRecoverPolicy) RecoverAt(ctx *RequestContext, err *types.NewAPIError, now int64) int64 {
	return time.Unix(now, 0).Add(default429RecoverDuration).Unix()
}

func (p *BaseAutoRecoverPolicy) Reason(ctx *RequestContext, err *types.NewAPIError) string {
	return AutoRecoverStatusReason429
}

type Codex429RecoverPolicy struct {
	BaseAutoRecoverPolicy
}

func (p *Codex429RecoverPolicy) Match(channel ChannelInfo) bool {
	return channel.Type == constant.ChannelTypeCodex
}

func (p *Codex429RecoverPolicy) RecoverAt(ctx *RequestContext, err *types.NewAPIError, now int64) int64 {
	message := ""
	if err != nil {
		message = strings.ToLower(err.Error())
	}
	recoverAfter := default429RecoverDuration
	switch {
	case strings.Contains(message, "7d"),
		strings.Contains(message, "7 days"),
		strings.Contains(message, "weekly limit"):
		recoverAfter = weekly429RecoverDuration
	case strings.Contains(message, "5h"),
		strings.Contains(message, "5 hours"),
		strings.Contains(message, "try again in 5 hours"):
		recoverAfter = default429RecoverDuration
	}
	return time.Unix(now, 0).Add(recoverAfter).Unix()
}

type NoopRecoverPolicy struct{}

func (p *NoopRecoverPolicy) Match(channel ChannelInfo) bool {
	return true
}

func (p *NoopRecoverPolicy) ShouldAutoDisable(ctx *RequestContext, err *types.NewAPIError) bool {
	return false
}

func (p *NoopRecoverPolicy) RecoverAt(ctx *RequestContext, err *types.NewAPIError, now int64) int64 {
	return 0
}

func (p *NoopRecoverPolicy) Reason(ctx *RequestContext, err *types.NewAPIError) string {
	return ""
}

type AutoRecoverPolicyRegistry struct {
	policies []AutoRecoverPolicy
	fallback AutoRecoverPolicy
}

func NewAutoRecoverPolicyRegistry() *AutoRecoverPolicyRegistry {
	return &AutoRecoverPolicyRegistry{
		policies: []AutoRecoverPolicy{
			&Codex429RecoverPolicy{},
		},
		fallback: &BaseAutoRecoverPolicy{},
	}
}

func (r *AutoRecoverPolicyRegistry) Match(channel ChannelInfo) AutoRecoverPolicy {
	for _, policy := range r.policies {
		if policy.Match(channel) {
			return policy
		}
	}
	if r.fallback != nil {
		return r.fallback
	}
	return &NoopRecoverPolicy{}
}

func ShouldAutoDisable429(ctx *RequestContext, channel ChannelInfo, err *types.NewAPIError) bool {
	policy := NewAutoRecoverPolicyRegistry().Match(channel)
	return policy.ShouldAutoDisable(ctx, err)
}

func Build429AutoRecoverInfo(ctx *RequestContext, channel ChannelInfo, err *types.NewAPIError, now int64) map[string]interface{} {
	policy := NewAutoRecoverPolicyRegistry().Match(channel)
	recoverAt := policy.RecoverAt(ctx, err, now)
	errorMessage := ""
	if err != nil {
		errorMessage = err.ErrorWithStatusCode()
	}
	return map[string]interface{}{
		OtherInfoStatusReasonKey:      policy.Reason(ctx, err),
		OtherInfoStatusTimeKey:        now,
		OtherInfoAutoRecoverReasonKey: AutoRecoverReason429,
		OtherInfoAutoRecoverAtKey:     recoverAt,
		OtherInfoAutoRecoverModelKey:  requestModel(ctx),
		OtherInfoAutoRecoverGroupKey:  requestGroup(ctx),
		OtherInfoAutoRecoverErrorKey:  truncateAutoRecoverError(errorMessage),
	}
}

func Is429AutoRecoverInfo(info map[string]interface{}) bool {
	if info == nil {
		return false
	}
	return fmt.Sprint(info[OtherInfoAutoRecoverReasonKey]) == AutoRecoverReason429
}

func ParseAutoRecoverAt(info map[string]interface{}) (int64, bool) {
	if info == nil {
		return 0, false
	}
	value, ok := info[OtherInfoAutoRecoverAtKey]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case int64:
		return v, v > 0
	case int:
		return int64(v), v > 0
	case float64:
		return int64(v), v > 0
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return parsed, err == nil && parsed > 0
	default:
		return 0, false
	}
}

func Set429AutoRecoverKeyInfo(info map[string]interface{}, keyIndex int, recoverInfo map[string]interface{}) map[string]interface{} {
	if info == nil {
		info = make(map[string]interface{})
	}
	keys := autoRecoverKeys(info)
	entry := cloneInfo(recoverInfo)
	entry[OtherInfoAutoRecoverKeyIndex] = keyIndex
	keys[strconv.Itoa(keyIndex)] = entry
	info[OtherInfoAutoRecoverKeysKey] = keys
	return info
}

func Due429AutoRecoverKeyIndexes(info map[string]interface{}, now int64) []int {
	keys := autoRecoverKeys(info)
	indexes := make([]int, 0, len(keys))
	for rawIndex, rawEntry := range keys {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok || !Is429AutoRecoverInfo(entry) {
			continue
		}
		recoverAt, ok := ParseAutoRecoverAt(entry)
		if !ok || recoverAt > now {
			continue
		}
		keyIndex, err := strconv.Atoi(rawIndex)
		if err != nil {
			continue
		}
		indexes = append(indexes, keyIndex)
	}
	return indexes
}

func Has429AutoRecoverKeys(info map[string]interface{}) bool {
	return len(autoRecoverKeys(info)) > 0
}

func Clean429AutoRecoverKeyInfo(info map[string]interface{}, keyIndexes ...int) map[string]interface{} {
	if info == nil {
		info = make(map[string]interface{})
	}
	keys := autoRecoverKeys(info)
	for _, keyIndex := range keyIndexes {
		delete(keys, strconv.Itoa(keyIndex))
	}
	if len(keys) == 0 {
		delete(info, OtherInfoAutoRecoverKeysKey)
		return Clean429AutoRecoverInfo(info)
	}
	info[OtherInfoAutoRecoverKeysKey] = keys
	return info
}

func Clean429AutoRecoverInfo(info map[string]interface{}) map[string]interface{} {
	if info == nil {
		info = make(map[string]interface{})
	}
	delete(info, OtherInfoAutoRecoverReasonKey)
	delete(info, OtherInfoAutoRecoverAtKey)
	delete(info, OtherInfoAutoRecoverModelKey)
	delete(info, OtherInfoAutoRecoverGroupKey)
	delete(info, OtherInfoAutoRecoverErrorKey)
	delete(info, OtherInfoAutoRecoverKeysKey)
	return info
}

func autoRecoverKeys(info map[string]interface{}) map[string]interface{} {
	if info == nil {
		return make(map[string]interface{})
	}
	rawKeys, ok := info[OtherInfoAutoRecoverKeysKey]
	if !ok || rawKeys == nil {
		return make(map[string]interface{})
	}
	switch keys := rawKeys.(type) {
	case map[string]interface{}:
		return keys
	case map[string]map[string]interface{}:
		converted := make(map[string]interface{}, len(keys))
		for key, value := range keys {
			converted[key] = value
		}
		return converted
	default:
		return make(map[string]interface{})
	}
}

func cloneInfo(info map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(info))
	for key, value := range info {
		cloned[key] = value
	}
	return cloned
}

func requestModel(ctx *RequestContext) string {
	if ctx == nil {
		return ""
	}
	return ctx.Model
}

func requestGroup(ctx *RequestContext) string {
	if ctx == nil {
		return ""
	}
	if ctx.TokenGroup != "" {
		return ctx.TokenGroup
	}
	return ctx.UserGroup
}

func truncateAutoRecoverError(message string) string {
	const maxLen = 512
	message = strings.TrimSpace(message)
	if len(message) <= maxLen {
		return message
	}
	return message[:maxLen]
}
