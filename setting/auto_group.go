package setting

import (
	"github.com/QuantumNous/new-api/common"
)

var autoGroups = []string{
	"free",
	"plus",
	"pro",
}

var DefaultUseAutoGroup = false

func ContainsAutoGroup(group string) bool {
	for _, autoGroup := range autoGroups {
		if autoGroup == group {
			return true
		}
	}
	return false
}

func UpdateAutoGroupsByJsonString(jsonString string) error {
	autoGroups = make([]string, 0)
	return common.Unmarshal([]byte(jsonString), &autoGroups)
}

func EnsureAutoGroups(groups []string) bool {
	if len(autoGroups) == len(groups) {
		matched := true
		for i := range groups {
			if autoGroups[i] != groups[i] {
				matched = false
				break
			}
		}
		if matched {
			return false
		}
	}
	autoGroups = append([]string(nil), groups...)
	return true
}

func AutoGroups2JsonString() string {
	jsonBytes, err := common.Marshal(autoGroups)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func GetAutoGroups() []string {
	return autoGroups
}
