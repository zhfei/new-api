package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertOneCardPlanForConsumeTest(t *testing.T, id int) {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:                      id,
		Title:                   fmt.Sprintf("OneCard Plan %d", id),
		PriceAmount:             1,
		Currency:                "USD",
		DurationUnit:            SubscriptionDurationDay,
		DurationValue:           1,
		QuotaResetPeriod:        SubscriptionResetCustom,
		QuotaResetCustomSeconds: 86400,
		Enabled:                 true,
		TotalAmount:             1000,
		ProductType:             "day_card",
		PoolGroup:               "auto",
	}
	require.NoError(t, DB.Create(plan).Error)
}

func insertUserSubscriptionForConsumeTest(t *testing.T, id int, userID int, planID int, amountTotal int64, amountUsed int64, endTime int64, nextResetTime int64) {
	t.Helper()
	sub := &UserSubscription{
		Id:            id,
		UserId:        userID,
		PlanId:        planID,
		Status:        "active",
		StartTime:     endTime - 86400,
		EndTime:       endTime,
		AmountTotal:   amountTotal,
		AmountUsed:    amountUsed,
		NextResetTime: nextResetTime,
	}
	require.NoError(t, DB.Create(sub).Error)
}

func getUserSubscriptionForConsumeTest(t *testing.T, id int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", id).First(&sub).Error)
	return sub
}

func TestValidateOneCardSubscriptionPlan_DurationRules(t *testing.T) {
	tests := []struct {
		name        string
		productType string
		unit        string
		value       int
		wantErr     bool
	}{
		{name: "day card", productType: "day_card", unit: SubscriptionDurationDay, value: 1},
		{name: "week card", productType: "week_card", unit: SubscriptionDurationDay, value: 7},
		{name: "month card", productType: "month_card", unit: SubscriptionDurationMonth, value: 1},
		{name: "bad day card duration", productType: "day_card", unit: SubscriptionDurationDay, value: 7, wantErr: true},
		{name: "bad week card unit", productType: "week_card", unit: SubscriptionDurationMonth, value: 1, wantErr: true},
		{name: "bad month card value", productType: "month_card", unit: SubscriptionDurationMonth, value: 2, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOneCardSubscriptionPlan(SubscriptionPlan{
				ProductType:             tt.productType,
				DurationUnit:            tt.unit,
				DurationValue:           tt.value,
				TotalAmount:             1000,
				QuotaResetPeriod:        SubscriptionResetCustom,
				QuotaResetCustomSeconds: 86400,
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPreConsumeUserSubscription_SelectsEarliestCurrentPeriodEnd(t *testing.T) {
	truncateTables(t)

	userID := 9101
	now := common.GetTimestamp()
	insertOneCardPlanForConsumeTest(t, 91011)
	insertOneCardPlanForConsumeTest(t, 91012)
	insertUserSubscriptionForConsumeTest(t, 910101, userID, 91011, 1000, 0, now+30*86400, now+16*3600)
	insertUserSubscriptionForConsumeTest(t, 910102, userID, 91012, 1000, 0, now+7*86400, now+3*3600)

	result, err := PreConsumeUserSubscription("onecard-period-order", userID, "gpt-5", 0, 200)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 910102, result.UserSubscriptionId)
	assert.EqualValues(t, 200, result.PreConsumed)
	assert.EqualValues(t, 200, getUserSubscriptionForConsumeTest(t, 910102).AmountUsed)
	assert.EqualValues(t, 0, getUserSubscriptionForConsumeTest(t, 910101).AmountUsed)
}

func TestPreConsumeUserSubscription_AllowsSingleCardOverdraft(t *testing.T) {
	truncateTables(t)

	userID := 9102
	now := common.GetTimestamp()
	insertOneCardPlanForConsumeTest(t, 91021)
	insertUserSubscriptionForConsumeTest(t, 910201, userID, 91021, 1000, 900, now+86400, now+3600)

	result, err := PreConsumeUserSubscription("onecard-overdraft", userID, "gpt-5", 0, 300)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 910201, result.UserSubscriptionId)
	assert.EqualValues(t, 900, result.AmountUsedBefore)
	assert.EqualValues(t, 1200, result.AmountUsedAfter)
	assert.EqualValues(t, 1200, getUserSubscriptionForConsumeTest(t, 910201).AmountUsed)
}

func TestPreConsumeUserSubscription_SkipsCardsWithoutPositiveCurrentRemain(t *testing.T) {
	truncateTables(t)

	userID := 9103
	now := common.GetTimestamp()
	insertOneCardPlanForConsumeTest(t, 91031)
	insertOneCardPlanForConsumeTest(t, 91032)
	insertUserSubscriptionForConsumeTest(t, 910301, userID, 91031, 1000, 1000, now+86400, now+3600)
	insertUserSubscriptionForConsumeTest(t, 910302, userID, 91032, 1000, 500, now+2*86400, now+2*3600)

	result, err := PreConsumeUserSubscription("onecard-skip-empty", userID, "gpt-5", 0, 200)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 910302, result.UserSubscriptionId)
	assert.EqualValues(t, 1000, getUserSubscriptionForConsumeTest(t, 910301).AmountUsed)
	assert.EqualValues(t, 700, getUserSubscriptionForConsumeTest(t, 910302).AmountUsed)
}
