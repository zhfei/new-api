package controller

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	paymentservice "github.com/QuantumNous/new-api/service/payment"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

type SubscriptionAlipayF2FPayRequest struct {
	PlanId int `json:"plan_id"`
}

func SubscriptionRequestAlipayF2F(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req SubscriptionAlipayF2FPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	provider := alipayF2FProvider()
	if err := provider.ValidateConfig(); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if err := validateOneCardPlan(*plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	userId := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	tradeNo := fmt.Sprintf("SUBUSR%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())
	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodAlipayF2F,
		PaymentProvider: model.PaymentProviderAlipayF2F,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	result, err := provider.CreatePayment(c.Request.Context(), paymentservice.PaymentCreateContext{
		TradeNo:   tradeNo,
		Amount:    plan.PriceAmount,
		Subject:   setting.AlipayF2FSubject,
		Body:      setting.AlipayF2FBody,
		NotifyURL: alipayF2FSubscriptionNotifyURL(),
		ReturnURL: alipayF2FSubscriptionReturnURL(),
	})
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderAlipayF2F)
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝当面付 订阅预下单失败 user_id=%d trade_no=%s plan_id=%d error=%q", userId, tradeNo, plan.Id, err.Error()))
		common.ApiErrorMsg(c, err.Error())
		return
	}

	order.ProviderPayload = common.GetJsonString(gin.H{
		"qr_code":       result.QRCode,
		"create_result": result,
	})
	if err := order.Update(); err != nil {
		common.ApiErrorMsg(c, "保存订单失败")
		return
	}

	c.JSON(200, gin.H{
		"message": "success",
		"data": buildAlipayF2FResponse(
			result,
			"/api/subscription/alipay-f2f/status",
			"/api/subscription/alipay-f2f/qrcode",
		),
	})
}

func AlipayF2FSubscriptionNotify(c *gin.Context) {
	if !isAlipayF2FTopUpEnabled() {
		writeAlipayF2FNotify(c, false)
		return
	}
	params := paymentservice.ReadFormOrQueryParams(c.Request)
	provider := alipayF2FProvider()
	if err := provider.VerifyCallback(params); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝当面付订阅 webhook 验签失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		writeAlipayF2FNotify(c, false)
		return
	}
	result, err := provider.ParseCallback(params)
	if err != nil {
		writeAlipayF2FNotify(c, false)
		return
	}
	if result.Status != paymentservice.CallbackStatusOK {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝当面付订阅 webhook 忽略非成功状态 trade_no=%s status=%s client_ip=%s", result.TradeNo, result.Status, c.ClientIP()))
		writeAlipayF2FNotify(c, true)
		return
	}

	LockOrder(result.TradeNo)
	defer UnlockOrder(result.TradeNo)
	order := model.GetSubscriptionOrderByTradeNo(result.TradeNo)
	if order == nil || order.PaymentProvider != model.PaymentProviderAlipayF2F {
		writeAlipayF2FNotify(c, false)
		return
	}
	if err := validateAlipayF2FCallback(result, order.Money); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝当面付订阅 webhook 校验失败 trade_no=%s error=%q", result.TradeNo, err.Error()))
		writeAlipayF2FNotify(c, false)
		return
	}
	providerPayload := mergeAlipayF2FProviderPayload(order.ProviderPayload, result.RawPayload)
	if err := model.CompleteSubscriptionOrder(result.TradeNo, providerPayload, model.PaymentProviderAlipayF2F, model.PaymentMethodAlipayF2F); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝当面付订阅履约失败 trade_no=%s error=%q", result.TradeNo, err.Error()))
		writeAlipayF2FNotify(c, false)
		return
	}
	writeAlipayF2FNotify(c, true)
}
