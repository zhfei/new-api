package setting

import (
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

var (
	subscriptionFirstGroups     = []string{"free", "plus", "pro", "auto"}
	officialPriceRequiredGroups = []string{"free", "plus", "pro", "auto"}

	oneCardMu sync.RWMutex
)

func OneCardEnabled() bool {
	return true
}

func SetOneCardEnabled(enabled bool) {
	// 一卡通项目的核心业务固定启用，外部配置不能关闭。
}

func SubscriptionFirstGroups2JsonString() string {
	oneCardMu.RLock()
	defer oneCardMu.RUnlock()
	return mustMarshalStringSlice(subscriptionFirstGroups)
}

func OfficialPriceRequiredGroups2JsonString() string {
	oneCardMu.RLock()
	defer oneCardMu.RUnlock()
	return mustMarshalStringSlice(officialPriceRequiredGroups)
}

func UpdateSubscriptionFirstGroupsByJSONString(jsonStr string) error {
	groups, err := parseGroupList(jsonStr)
	if err != nil {
		return err
	}
	oneCardMu.Lock()
	defer oneCardMu.Unlock()
	subscriptionFirstGroups = groups
	return nil
}

func UpdateOfficialPriceRequiredGroupsByJSONString(jsonStr string) error {
	groups, err := parseGroupList(jsonStr)
	if err != nil {
		return err
	}
	oneCardMu.Lock()
	defer oneCardMu.Unlock()
	officialPriceRequiredGroups = groups
	return nil
}

func IsSubscriptionFirstGroup(group string) bool {
	oneCardMu.RLock()
	defer oneCardMu.RUnlock()
	return containsGroup(subscriptionFirstGroups, group)
}

func IsOfficialPriceRequiredGroup(group string) bool {
	oneCardMu.RLock()
	defer oneCardMu.RUnlock()
	return containsGroup(officialPriceRequiredGroups, group)
}

func IsOneCardGroup(group string) bool {
	switch strings.TrimSpace(group) {
	case "free", "plus", "pro", "auto":
		return true
	default:
		return false
	}
}

func RequiredOneCardAutoGroups() []string {
	return []string{"free", "plus", "pro"}
}

func ValidateOneCardAutoGroups(groups []string) bool {
	required := RequiredOneCardAutoGroups()
	if len(groups) != len(required) {
		return false
	}
	for i := range required {
		if groups[i] != required[i] {
			return false
		}
	}
	return true
}

func parseGroupList(jsonStr string) ([]string, error) {
	var groups []string
	if err := common.UnmarshalJsonStr(jsonStr, &groups); err != nil {
		return nil, err
	}
	cleaned := make([]string, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		cleaned = append(cleaned, group)
	}
	return cleaned, nil
}

func containsGroup(groups []string, group string) bool {
	group = strings.TrimSpace(group)
	for _, item := range groups {
		if item == group {
			return true
		}
	}
	return false
}

func mustMarshalStringSlice(groups []string) string {
	bytes, err := common.Marshal(groups)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}
