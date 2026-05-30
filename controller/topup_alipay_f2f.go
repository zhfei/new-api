package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	paymentservice "github.com/QuantumNous/new-api/service/payment"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func RequestAlipayF2FAmount(c *gin.Context) {
	var req AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	minTopUp := int64(setting.AlipayF2FMinTopUp)
	if req.Amount < minTopUp {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", minTopUp)})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": fmt.Sprintf("%.2f", payMoney)})
}

func RequestAlipayF2FPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req EpayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	minTopUp := int64(setting.AlipayF2FMinTopUp)
	if req.Amount < minTopUp {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", minTopUp)})
		return
	}
	provider := alipayF2FProvider()
	if err := provider.ValidateConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	tradeNo := fmt.Sprintf("USR%dNO%s%d", id, common.GetRandomString(6), time.Now().Unix())
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodAlipayF2F,
		PaymentProvider: model.PaymentProviderAlipayF2F,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝当面付 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	result, err := provider.CreatePayment(c.Request.Context(), paymentservice.PaymentCreateContext{
		TradeNo:   tradeNo,
		Amount:    payMoney,
		Subject:   setting.AlipayF2FSubject,
		Body:      setting.AlipayF2FBody,
		NotifyURL: alipayF2FTopUpNotifyURL(),
		ReturnURL: alipayF2FTopUpReturnURL(),
	})
	if err != nil {
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderAlipayF2F, common.TopUpStatusFailed)
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝当面付 预下单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}

	topUp.ProviderPayload = common.GetJsonString(gin.H{
		"qr_code":       result.QRCode,
		"create_result": result,
	})
	if err := topUp.Update(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝当面付 保存充值订单二维码失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "保存订单失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝当面付 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f", id, tradeNo, req.Amount, payMoney))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": buildAlipayF2FResponse(
			result,
			"/api/user/alipay-f2f/status",
			"/api/user/alipay-f2f/qrcode",
		),
	})
}

func AlipayF2FTopUpNotify(c *gin.Context) {
	if !isAlipayF2FTopUpEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝当面付充值 webhook 被拒绝 reason=disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		writeAlipayF2FNotify(c, false)
		return
	}
	params := paymentservice.ReadFormOrQueryParams(c.Request)
	provider := alipayF2FProvider()
	if err := provider.VerifyCallback(params); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝当面付充值 webhook 验签失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		writeAlipayF2FNotify(c, false)
		return
	}
	result, err := provider.ParseCallback(params)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝当面付充值 webhook 解析失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		writeAlipayF2FNotify(c, false)
		return
	}
	if result.Status != paymentservice.CallbackStatusOK {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝当面付充值 webhook 忽略非成功状态 trade_no=%s status=%s client_ip=%s", result.TradeNo, result.Status, c.ClientIP()))
		writeAlipayF2FNotify(c, true)
		return
	}

	LockOrder(result.TradeNo)
	defer UnlockOrder(result.TradeNo)
	var logUserId int
	var logQuotaToAdd int
	var logMoney float64
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		var topUp model.TopUp
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("trade_no = ?", result.TradeNo).First(&topUp).Error; err != nil {
			return err
		}
		if topUp.PaymentProvider != model.PaymentProviderAlipayF2F {
			return model.ErrPaymentMethodMismatch
		}
		if err := validateAlipayF2FCallback(result, topUp.Money); err != nil {
			return err
		}
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return fmt.Errorf("支付宝当面付充值订单状态异常: %s", topUp.Status)
		}
		quotaToAdd := int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
		if quotaToAdd <= 0 {
			return fmt.Errorf("支付宝当面付充值额度无效")
		}
		topUp.PaymentMethod = model.PaymentMethodAlipayF2F
		topUp.ProviderPayload = mergeAlipayF2FProviderPayload(topUp.ProviderPayload, result.RawPayload)
		topUp.CompleteTime = time.Now().Unix()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(&topUp).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}
		logUserId = topUp.UserId
		logQuotaToAdd = quotaToAdd
		logMoney = topUp.Money
		return nil
	})
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝当面付充值 webhook 履约失败 trade_no=%s error=%q", result.TradeNo, err.Error()))
		writeAlipayF2FNotify(c, false)
		return
	}
	if logUserId > 0 && logQuotaToAdd > 0 {
		go func(userId int, quotaToAdd int) {
			if err := model.IncrUserQuotaCache(userId, int64(quotaToAdd)); err != nil {
				common.SysLog("failed to increase user quota cache: " + err.Error())
			}
		}(logUserId, logQuotaToAdd)
	}
	if logUserId > 0 {
		model.RecordTopupLog(logUserId, fmt.Sprintf("支付宝当面付充值成功，充值金额: %v，支付金额：%.2f", logger.FormatQuota(logQuotaToAdd), logMoney), c.ClientIP(), model.PaymentMethodAlipayF2F, model.PaymentProviderAlipayF2F)
	}
	writeAlipayF2FNotify(c, true)
}
