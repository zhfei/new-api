package controller

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---- Shared types ----

type SubscriptionPlanDTO struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

type BillingPreferenceRequest struct {
	BillingPreference string `json:"billing_preference"`
}

type AdminSubscriptionOrderView struct {
	Id              int     `json:"id"`
	UserId          int     `json:"user_id"`
	PlanId          int     `json:"plan_id"`
	Money           float64 `json:"money"`
	TradeNo         string  `json:"trade_no"`
	PaymentMethod   string  `json:"payment_method"`
	PaymentProvider string  `json:"payment_provider"`
	Status          string  `json:"status"`
	CreateTime      int64   `json:"create_time"`
	CompleteTime    int64   `json:"complete_time"`
}

type AdminSubscriptionOrderUserView struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type AdminSubscriptionOrderItem struct {
	Order AdminSubscriptionOrderView      `json:"order"`
	Plan  *model.SubscriptionPlan         `json:"plan,omitempty"`
	User  *AdminSubscriptionOrderUserView `json:"user,omitempty"`
}

type AdminOneCardProductStats struct {
	ProductType      string  `json:"product_type"`
	PlanCount        int64   `json:"plan_count"`
	EnabledPlanCount int64   `json:"enabled_plan_count"`
	OrderCount       int64   `json:"order_count"`
	OrderRevenue     float64 `json:"order_revenue"`
	ActiveCardCount  int64   `json:"active_card_count"`
	ActiveAmount     int64   `json:"active_amount"`
	ActiveUsed       int64   `json:"active_used"`
	ActiveRemain     int64   `json:"active_remain"`
}

// ---- User APIs ----

func GetSubscriptionPlans(c *gin.Context) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiSuccess(c, []SubscriptionPlanDTO{})
		return
	}

	var plans []model.SubscriptionPlan
	if err := model.DB.Where("enabled = ?", true).Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		if model.IsOneCardSubscriptionProduct(p.ProductType) {
			if err := validateOneCardPlan(p); err != nil {
				continue
			}
		}
		result = append(result, SubscriptionPlanDTO{
			Plan: p,
		})
	}
	common.ApiSuccess(c, result)
}

func GetSubscriptionSelf(c *gin.Context) {
	userId := c.GetInt("id")
	settingMap, _ := model.GetUserSetting(userId, false)
	pref := common.NormalizeBillingPreference(settingMap.BillingPreference)

	// Get all subscriptions (including expired)
	allSubscriptions, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		allSubscriptions = []model.SubscriptionSummary{}
	}

	// Get active subscriptions for backward compatibility
	activeSubscriptions, err := model.GetAllActiveUserSubscriptions(userId)
	if err != nil {
		activeSubscriptions = []model.SubscriptionSummary{}
	}

	common.ApiSuccess(c, gin.H{
		"billing_preference": pref,
		"subscriptions":      activeSubscriptions, // all active subscriptions
		"all_subscriptions":  allSubscriptions,    // all subscriptions including expired
	})
}

func UpdateSubscriptionPreference(c *gin.Context) {
	userId := c.GetInt("id")
	var req BillingPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	pref := common.NormalizeBillingPreference(req.BillingPreference)

	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	current := user.GetSetting()
	current.BillingPreference = pref
	user.SetSetting(current)
	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"billing_preference": pref})
}

// ---- Admin APIs ----

func AdminListSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		result = append(result, SubscriptionPlanDTO{
			Plan: p,
		})
	}
	common.ApiSuccess(c, result)
}

func AdminListSubscriptionOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	query := model.DB.Model(&model.SubscriptionOrder{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if paymentProvider := strings.TrimSpace(c.Query("payment_provider")); paymentProvider != "" {
		query = query.Where("payment_provider = ?", paymentProvider)
	}
	if userId, _ := strconv.Atoi(c.Query("user_id")); userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if planId, _ := strconv.Atoi(c.Query("plan_id")); planId > 0 {
		query = query.Where("plan_id = ?", planId)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	var orders []model.SubscriptionOrder
	if err := query.Order("id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&orders).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	planIds := make([]int, 0)
	userIds := make([]int, 0)
	planIdSet := map[int]bool{}
	userIdSet := map[int]bool{}
	for _, order := range orders {
		if order.PlanId > 0 && !planIdSet[order.PlanId] {
			planIdSet[order.PlanId] = true
			planIds = append(planIds, order.PlanId)
		}
		if order.UserId > 0 && !userIdSet[order.UserId] {
			userIdSet[order.UserId] = true
			userIds = append(userIds, order.UserId)
		}
	}

	plans := map[int]model.SubscriptionPlan{}
	if len(planIds) > 0 {
		var planRows []model.SubscriptionPlan
		if err := model.DB.Where("id IN ?", planIds).Find(&planRows).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		for _, plan := range planRows {
			plans[plan.Id] = plan
		}
	}

	users := map[int]AdminSubscriptionOrderUserView{}
	if len(userIds) > 0 {
		var userRows []AdminSubscriptionOrderUserView
		if err := model.DB.Model(&model.User{}).
			Select("id, username, display_name, email").
			Where("id IN ?", userIds).
			Find(&userRows).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		for _, user := range userRows {
			users[user.Id] = user
		}
	}

	items := make([]AdminSubscriptionOrderItem, 0, len(orders))
	for _, order := range orders {
		item := AdminSubscriptionOrderItem{
			Order: AdminSubscriptionOrderView{
				Id:              order.Id,
				UserId:          order.UserId,
				PlanId:          order.PlanId,
				Money:           order.Money,
				TradeNo:         order.TradeNo,
				PaymentMethod:   order.PaymentMethod,
				PaymentProvider: order.PaymentProvider,
				Status:          order.Status,
				CreateTime:      order.CreateTime,
				CompleteTime:    order.CompleteTime,
			},
		}
		if plan, ok := plans[order.PlanId]; ok {
			item.Plan = &plan
		}
		if user, ok := users[order.UserId]; ok {
			item.User = &user
		}
		items = append(items, item)
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func AdminGetOneCardStats(c *gin.Context) {
	productTypes := []string{"day_card", "week_card", "month_card"}
	statsMap := make(map[string]*AdminOneCardProductStats, len(productTypes))
	for _, productType := range productTypes {
		statsMap[productType] = &AdminOneCardProductStats{ProductType: productType}
	}

	type planStatsRow struct {
		ProductType      string
		PlanCount        int64
		EnabledPlanCount int64
	}
	var planRows []planStatsRow
	if err := model.DB.Model(&model.SubscriptionPlan{}).
		Select("product_type, COUNT(*) AS plan_count, SUM(CASE WHEN enabled = ? THEN 1 ELSE 0 END) AS enabled_plan_count", true).
		Where("product_type IN ?", productTypes).
		Group("product_type").
		Scan(&planRows).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	for _, row := range planRows {
		if stat, ok := statsMap[row.ProductType]; ok {
			stat.PlanCount = row.PlanCount
			stat.EnabledPlanCount = row.EnabledPlanCount
		}
	}

	type orderStatsRow struct {
		ProductType  string
		OrderCount   int64
		OrderRevenue float64
	}
	var orderRows []orderStatsRow
	if err := model.DB.Table("subscription_orders AS o").
		Select("p.product_type AS product_type, COUNT(o.id) AS order_count, COALESCE(SUM(o.money), 0) AS order_revenue").
		Joins("JOIN subscription_plans AS p ON p.id = o.plan_id").
		Where("p.product_type IN ? AND o.status = ?", productTypes, common.TopUpStatusSuccess).
		Group("p.product_type").
		Scan(&orderRows).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	for _, row := range orderRows {
		if stat, ok := statsMap[row.ProductType]; ok {
			stat.OrderCount = row.OrderCount
			stat.OrderRevenue = row.OrderRevenue
		}
	}

	type activeStatsRow struct {
		ProductType     string
		ActiveCardCount int64
		ActiveAmount    int64
		ActiveUsed      int64
		ActiveRemain    int64
	}
	var activeRows []activeStatsRow
	now := common.GetTimestamp()
	if err := model.DB.Table("user_subscriptions AS s").
		Select("p.product_type AS product_type, COUNT(s.id) AS active_card_count, COALESCE(SUM(s.amount_total), 0) AS active_amount, COALESCE(SUM(s.amount_used), 0) AS active_used, COALESCE(SUM(s.amount_total - s.amount_used), 0) AS active_remain").
		Joins("JOIN subscription_plans AS p ON p.id = s.plan_id").
		Where("p.product_type IN ? AND s.status = ? AND s.end_time > ?", productTypes, "active", now).
		Group("p.product_type").
		Scan(&activeRows).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	for _, row := range activeRows {
		if stat, ok := statsMap[row.ProductType]; ok {
			stat.ActiveCardCount = row.ActiveCardCount
			stat.ActiveAmount = row.ActiveAmount
			stat.ActiveUsed = row.ActiveUsed
			stat.ActiveRemain = row.ActiveRemain
		}
	}

	stats := make([]AdminOneCardProductStats, 0, len(productTypes))
	for _, productType := range productTypes {
		stats = append(stats, *statsMap[productType])
	}
	common.ApiSuccess(c, gin.H{"items": stats})
}

type AdminUpsertSubscriptionPlanRequest struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

func normalizeOneCardPlanFields(plan *model.SubscriptionPlan) {
	if plan == nil {
		return
	}
	plan.ProductType = strings.TrimSpace(plan.ProductType)
	plan.PoolGroup = strings.TrimSpace(plan.PoolGroup)
	plan.DisplayBadge = strings.TrimSpace(plan.DisplayBadge)
	if model.IsOneCardSubscriptionProduct(plan.ProductType) {
		plan.QuotaResetPeriod = model.SubscriptionResetCustom
		plan.QuotaResetCustomSeconds = 86400
	}
}

func validateOneCardPlan(plan model.SubscriptionPlan) error {
	if err := model.ValidateOneCardSubscriptionPlan(plan); err != nil {
		return err
	}
	if plan.PoolGroup != "" {
		switch plan.PoolGroup {
		case "free", "plus", "pro", "auto":
		default:
			return fmt.Errorf("默认展示池组必须是 free、plus、pro 或 auto")
		}
	}
	return nil
}

func AdminCreateSubscriptionPlan(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Plan.Id = 0
	if strings.TrimSpace(req.Plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if req.Plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if req.Plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	if req.Plan.Currency == "" {
		req.Plan.Currency = "USD"
	}
	req.Plan.Currency = "USD"
	if req.Plan.DurationUnit == "" {
		req.Plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if req.Plan.DurationValue <= 0 && req.Plan.DurationUnit != model.SubscriptionDurationCustom {
		req.Plan.DurationValue = 1
	}
	if req.Plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if req.Plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	normalizeOneCardPlanFields(&req.Plan)
	if err := validateOneCardPlan(req.Plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	req.Plan.UpgradeGroup = strings.TrimSpace(req.Plan.UpgradeGroup)
	if req.Plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	req.Plan.QuotaResetPeriod = model.NormalizeResetPeriod(req.Plan.QuotaResetPeriod)
	if req.Plan.QuotaResetPeriod == model.SubscriptionResetCustom && req.Plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}
	err := model.DB.Create(&req.Plan).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(req.Plan.Id)
	common.ApiSuccess(c, req.Plan)
}

func AdminUpdateSubscriptionPlan(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if req.Plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if req.Plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	req.Plan.Id = id
	if req.Plan.Currency == "" {
		req.Plan.Currency = "USD"
	}
	req.Plan.Currency = "USD"
	if req.Plan.DurationUnit == "" {
		req.Plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if req.Plan.DurationValue <= 0 && req.Plan.DurationUnit != model.SubscriptionDurationCustom {
		req.Plan.DurationValue = 1
	}
	if req.Plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if req.Plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	normalizeOneCardPlanFields(&req.Plan)
	if err := validateOneCardPlan(req.Plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	req.Plan.UpgradeGroup = strings.TrimSpace(req.Plan.UpgradeGroup)
	if req.Plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	req.Plan.QuotaResetPeriod = model.NormalizeResetPeriod(req.Plan.QuotaResetPeriod)
	if req.Plan.QuotaResetPeriod == model.SubscriptionResetCustom && req.Plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		// update plan (allow zero values updates with map)
		updateMap := map[string]interface{}{
			"title":                      req.Plan.Title,
			"subtitle":                   req.Plan.Subtitle,
			"price_amount":               req.Plan.PriceAmount,
			"currency":                   req.Plan.Currency,
			"duration_unit":              req.Plan.DurationUnit,
			"duration_value":             req.Plan.DurationValue,
			"custom_seconds":             req.Plan.CustomSeconds,
			"enabled":                    req.Plan.Enabled,
			"sort_order":                 req.Plan.SortOrder,
			"stripe_price_id":            req.Plan.StripePriceId,
			"creem_product_id":           req.Plan.CreemProductId,
			"max_purchase_per_user":      req.Plan.MaxPurchasePerUser,
			"total_amount":               req.Plan.TotalAmount,
			"upgrade_group":              req.Plan.UpgradeGroup,
			"quota_reset_period":         req.Plan.QuotaResetPeriod,
			"quota_reset_custom_seconds": req.Plan.QuotaResetCustomSeconds,
			"product_type":               req.Plan.ProductType,
			"pool_group":                 req.Plan.PoolGroup,
			"display_badge":              req.Plan.DisplayBadge,
			"metadata":                   req.Plan.Metadata,
			"updated_at":                 common.GetTimestamp(),
		}
		if err := tx.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminUpdateSubscriptionPlanStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

func AdminUpdateSubscriptionPlanStatus(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpdateSubscriptionPlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Update("enabled", *req.Enabled).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminBindSubscriptionRequest struct {
	UserId int `json:"user_id"`
	PlanId int `json:"plan_id"`
}

func AdminBindSubscription(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req AdminBindSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserId <= 0 || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(req.UserId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: user subscription management ----

func AdminListUserSubscriptions(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	subs, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, subs)
}

type AdminCreateUserSubscriptionRequest struct {
	PlanId int `json:"plan_id"`
}

// AdminCreateUserSubscription creates a new user subscription from a plan (no payment).
func AdminCreateUserSubscription(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminCreateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(userId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminInvalidateUserSubscription cancels a user subscription immediately.
func AdminInvalidateUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminInvalidateUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminDeleteUserSubscription hard-deletes a user subscription.
func AdminDeleteUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminDeleteUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}
